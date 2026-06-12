package services

import (
	"log"
)

// The proxy now handles IPTV natively without FFmpeg/SRS re-encoding.
// These functions are kept as no-ops to maintain API compatibility.

func StartRestream(channelID uint, iptvURL string) error {
	log.Printf("[IPTV] Channel %d enabled native IPTV proxy to %s", channelID, iptvURL)
	return nil
}

func StopRestream(channelID uint) {
	log.Printf("[IPTV] Channel %d disabled native IPTV proxy", channelID)
}

func StatusRestream(channelID uint) bool {
	// The frontend/backend now relies on the DB field `IptvEnabled` rather than an active process.
	// But to keep the admin UI happy, we return true if it's supposed to be running.
	// Actually, the admin UI queries the DB directly for status now.
	return false 
}

func ResumeRestreams() {
	log.Printf("[IPTV] FFmpeg restreamer disabled. Using optimized native proxy.")
}
