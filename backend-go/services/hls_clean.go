package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// CleanHLSDir is where the Chromecast/Smart-TV-friendly HLS is written.
// It lives inside the shared srs_hls volume so nginx serves the .ts segments
// at /hls/live_clean/*.ts (same handling as /hls/live/).
const CleanHLSDir = "/app/srs_hls/live_clean"

var (
	cleanMu   sync.Mutex
	cleanJobs = map[string]context.CancelFunc{}
)

// StartCleanHLS launches a clean HLS pipeline for the raw OBS stream that SRS
// receives via RTMP. See runCleanLoop for why we clean the bitstream.
func StartCleanHLS(streamName string) {
	inputArgs := []string{"-i", fmt.Sprintf("rtmp://srs:1935/live/%s", streamName)}
	startClean(streamName, inputArgs)
}

// startClean deduplicates by streamName and launches the keep-alive loop.
func startClean(streamName string, inputArgs []string) {
	if streamName == "" {
		return
	}

	cleanMu.Lock()
	if _, ok := cleanJobs[streamName]; ok {
		cleanMu.Unlock()
		return // already running
	}
	ctx, cancel := context.WithCancel(context.Background())
	cleanJobs[streamName] = cancel
	cleanMu.Unlock()

	go runCleanLoop(ctx, streamName, inputArgs)
}

// runCleanLoop keeps an ffmpeg alive that republishes a Chromecast/Smart-TV
// friendly HLS. It strips the HRD "buffering period" SEI (NAL type 6) that
// OBS/x264 in CBR mode emits ordered before the SPS — strict receivers
// (Chromecast/Shaka) reject it — and copies the rest (no re-encode).
func runCleanLoop(ctx context.Context, streamName string, inputArgs []string) {
	if err := os.MkdirAll(CleanHLSDir, 0o777); err != nil {
		log.Printf("[cleanhls] mkdir %s error: %v", CleanHLSDir, err)
	}

	playlist := fmt.Sprintf("%s/%s.m3u8", CleanHLSDir, streamName)
	// Timestamped segment names so a restart never collides with (or replays)
	// a previous run's files. delete_segments trims the rolled-off ones.
	segPattern := fmt.Sprintf("%s/%s-%%Y%%m%%d%%H%%M%%S.ts", CleanHLSDir, streamName)

	// Wipe leftovers from a previous session ONCE, before the first launch.
	// (Not inside the retry loop: a transient reconnect must not blow away the
	// live window the TV is currently playing.)
	cleanStreamFiles(streamName)

	for {
		if ctx.Err() != nil {
			return
		}

		args := append([]string{"-loglevel", "warning"}, inputArgs...)
		args = append(args,
			"-c", "copy",
			"-bsf:v", "filter_units=remove_types=6",
			"-f", "hls",
			// Shorter segments = lower live latency on the TV. Actual segment
			// length can't go below OBS's keyframe interval with -c copy, so for
			// minimal delay set OBS keyframe interval to 1-2s.
			"-hls_time", "2",
			// Short advertised window so the Chromecast (which starts at the
			// window's oldest segment) begins close to the live edge...
			"-hls_list_size", "6",
			// ...but retain extra rolled-off segments on disk so a slightly
			// behind player never hits a deleted segment and stalls.
			"-hls_delete_threshold", "12",
			"-hls_flags", "delete_segments+omit_endlist+program_date_time",
			"-hls_segment_type", "mpegts",
			"-strftime", "1",
			"-hls_segment_filename", segPattern,
			playlist,
		)

		cmd := exec.CommandContext(ctx, "ffmpeg", args...)

		log.Printf("[cleanhls] launching ffmpeg for %s", streamName)
		err := cmd.Run()

		if ctx.Err() != nil {
			return // cancelled via StopCleanHLS
		}

		// ffmpeg exited on its own (e.g. stream not publishable yet, or a
		// transient RTMP drop). Retry until we're told to stop.
		log.Printf("[cleanhls] ffmpeg for %s exited (err=%v), retrying in 2s", streamName, err)
		time.Sleep(2 * time.Second)
	}
}

// StopCleanHLS stops the ffmpeg for a stream and removes its leftover files so
// stale segments are never served after the stream ends.
func StopCleanHLS(streamName string) {
	if streamName == "" {
		return
	}

	cleanMu.Lock()
	cancel, ok := cleanJobs[streamName]
	if ok {
		delete(cleanJobs, streamName)
	}
	cleanMu.Unlock()

	if ok {
		cancel()
		log.Printf("[cleanhls] stopped clean HLS for %s", streamName)
	}
	// No file cleanup here on purpose: when switching source (OBS→IPTV) a new
	// run starts immediately and wipes/reuses the directory itself. Deleting
	// files here would race with (and destroy) the new run's fresh segments.
}

// cleanStreamFiles removes the playlist and segments for a stream so a new run
// never mixes with a previous run's files.
func cleanStreamFiles(streamName string) {
	matches, _ := filepath.Glob(fmt.Sprintf("%s/%s*", CleanHLSDir, streamName))
	for _, f := range matches {
		_ = os.Remove(f)
	}
}

// CleanHLSReady reports whether a non-empty clean playlist currently exists.
func CleanHLSReady(streamName string) bool {
	if streamName == "" {
		return false
	}
	info, err := os.Stat(fmt.Sprintf("%s/%s.m3u8", CleanHLSDir, streamName))
	return err == nil && info.Size() > 0
}

// CleanHLSFresh reports whether the clean playlist exists AND is still being
// updated by a live ffmpeg (mtime within a few target durations). A merely
// existing playlist is not enough: StopCleanHLS leaves files on disk on
// purpose, so after a stream ends a stale playlist survives and — if served —
// freezes the player forever (it keeps polling a manifest that never advances).
func CleanHLSFresh(streamName string) bool {
	if streamName == "" {
		return false
	}
	info, err := os.Stat(fmt.Sprintf("%s/%s.m3u8", CleanHLSDir, streamName))
	if err != nil || info.Size() == 0 {
		return false
	}
	// 5x hls_time (2s): generous enough for jitter, strict enough that a dead
	// pipeline is detected within seconds.
	return time.Since(info.ModTime()) < 10*time.Second
}
