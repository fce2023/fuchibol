# Design Spec: Smooth Video Source Transitions & WebRTC Optimization

## Goals
1. Implement a dual video element setup in the player to allow seamless crossfading when switching stream URLs or stream types (HLS <-> WebRTC).
2. Configure a playback jitter buffer in WebRTC receivers (`playoutDelayHint`) to stabilize incoming streams from OBS, reducing lagging/stuttering.
3. Update the streamer guide with recommended OBS settings for WebRTC compatibility (Tune: zerolatency, Max B-frames: 0).

## 1. Dual Video Player Architecture
In `VideoPlayer.vue`, we will replace the single `<video>` element with two:
- `videoRefA`
- `videoRefB`

We will track:
- `activeVideo` ('A' or 'B' or null)
- Player instance states for both (HLS and RTCPeerConnection)

### Transition Algorithm:
1. When `streamUrl` or `isLive` changes:
   - Identify the target video element (the one currently inactive/hidden).
   - Initialize the new player instance (HLS.js or WebRTC connection) bound to the target video element.
   - The active video element continues playing the current stream (if any).
2. Listen to the `playing` event on the target video element.
3. When the target starts playing:
   - Apply CSS class to fade in the target video (opacity `0` -> `1`) and fade out the old active video (opacity `1` -> `0`).
   - Hide the loading spinner.
   - Update `activeVideo` reference to the target.
   - After a short delay (e.g. 500ms), stop playback and destroy resources of the old active video.

## 2. WebRTC Jitter Buffer (playoutDelayHint)
To mitigate network and encoder jitter from OBS streams over UDP:
- Set `receiver.playoutDelayHint = 0.5` on all WebRTC receivers in `RTCPeerConnection.ontrack`.
- This tells the browser to buffer ~500ms of frames, stabilizing the frame delivery and eliminating stutter.

## 3. Streamer Guide (OBS Settings)
Add instructions in `/docs/04_GUIA_STREAMERS.md` to configure:
- **Rate Control**: CBR
- **Keyframe Interval**: 2s
- **Tune**: `zerolatency`
- **Max B-frames**: `0`
