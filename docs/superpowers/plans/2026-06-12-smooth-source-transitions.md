# Smooth Video Source Transitions & WebRTC Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a dual-video-element crossfade system for seamless source transitions and configure WebRTC receivers with a playout delay of 500ms to eliminate OBS streaming jitter.

**Architecture:** Use two layered HTML5 video elements (Video A and Video B) in Vue. Load new sources on the inactive video in the background, crossfade opacities once the new source starts playing, and then clean up the old source. Optimize WebRTC by setting `playoutDelayHint = 0.5`.

**Tech Stack:** Vue 3, HTML5 Video API, WebRTC RTCPeerConnection API, Hls.js, Nginx.

---

### Task 1: Update OBS Streamer Guide

**Files:**
- Modify: `docs/04_GUIA_STREAMERS.md:15-30`

- [ ] **Step 1: Edit the streamer guide to add WebRTC-optimized settings**
  Add explicit instructions for Zero Latency Tuning and disabling B-frames in OBS Studio.

  ```markdown
  *   **Codificador (Encoder):** H.264 (x264 o NVIDIA NVENC).
  *   **Control de Frecuencia (Rate Control):** CBR (Constant Bitrate).
  *   **Tasa de Bits (Bitrate):** 
      *   Para 1080p 60fps: `6000 Kbps`
      *   Para 720p 60fps: `4500 Kbps`
      *   Para 720p 30fps: `3000 Kbps`
  *   **Intervalo de Fotogramas Clave (Keyframe Interval):** `2 s` (¡MUY IMPORTANTE para WebRTC y HLS de baja latencia!).
  *   **Perfil (Profile):** `baseline` (para máxima compatibilidad con WebRTC) o `main`.
  *   **Sintonización (Tune):** `zerolatency` (¡CRÍTICO para evitar lag en WebRTC!).
  *   **Fotogramas B Máximos (Max B-frames):** `0` (¡CRÍTICO! Los B-frames no se soportan bien en tiempo real en WebRTC y causan tirones visuales).
  ```

- [ ] **Step 2: Commit changes**

  ```bash
  git add docs/04_GUIA_STREAMERS.md
  git commit -m "docs: optimize streamer guide for WebRTC latency and smooth play"
  ```

---

### Task 2: Implement Dual Video Crossfade & WebRTC Jitter Buffer

**Files:**
- Modify: `frontend/src/components/VideoPlayer.vue`

- [ ] **Step 1: Replace template with dual video elements**
  Modify `<template>` in `VideoPlayer.vue` to contain `videoRefA` and `videoRefB`.

  ```html
  <video 
    ref="videoRefA" 
    class="video-element" 
    :class="{ 'active': activeVideo === 'A' }"
    playsinline
    webkit-playsinline
    :controls="activeVideo === 'A' && showControls"
  ></video>

  <video 
    ref="videoRefB" 
    class="video-element" 
    :class="{ 'active': activeVideo === 'B' }"
    playsinline
    webkit-playsinline
    :controls="activeVideo === 'B' && showControls"
  ></video>
  ```

- [ ] **Step 2: Update script logic in VideoPlayer.vue**
  Replace script section to implement dual player instances, crossfade timing, and WebRTC `playoutDelayHint = 0.5`.

  ```javascript
  const videoRefA = ref(null)
  const videoRefB = ref(null)
  const activeVideo = ref(null) // 'A' or 'B' or null
  const hasError = ref(false)
  const isLoading = ref(true)
  const showControls = ref(false)
  const qualities = ref([])
  const currentQuality = ref(-1)

  // Track player structures
  const players = {
    A: { hls: null, rtc: null, retryTimer: null, playingListener: null },
    B: { hls: null, rtc: null, retryTimer: null, playingListener: null }
  }

  const changeQuality = () => {
    const active = activeVideo.value
    if (active && players[active].hls) {
      players[active].hls.currentLevel = currentQuality.value;
    }
  }

  const destroyPlayerInstance = (key) => {
    const player = players[key]
    if (player.hls) {
      player.hls.destroy()
      player.hls = null
    }
    if (player.rtc) {
      player.rtc.close()
      player.rtc = null
    }
    if (player.retryTimer) {
      clearTimeout(player.retryTimer)
      player.retryTimer = null
    }
    const video = key === 'A' ? videoRefA.value : videoRefB.value
    if (video) {
      if (player.playingListener) {
        video.removeEventListener('playing', player.playingListener)
        player.playingListener = null
      }
      video.srcObject = null
      video.removeAttribute('src')
      video.load()
    }
  }

  const initPlayer = async () => {
    if (!props.isLive || !props.streamUrl) {
      destroyPlayerInstance('A')
      destroyPlayerInstance('B')
      activeVideo.value = null
      isLoading.value = false
      return
    }

    // Determine target video container (crossfade target)
    const targetKey = activeVideo.value === 'A' ? 'B' : 'A'
    const targetVideoEl = targetKey === 'A' ? videoRefA.value : videoRefB.value
    if (!targetVideoEl) return

    // Clean up any stale setups on the target container before loading
    destroyPlayerInstance(targetKey)
    hasError.value = false

    // If there is no active video yet, we are in initial loading state
    if (!activeVideo.value) {
      isLoading.value = true
    }

    // Set up playing event listener to trigger the crossfade transition
    players[targetKey].playingListener = () => {
      // Transition active reference
      const oldKey = activeVideo.value
      activeVideo.value = targetKey
      isLoading.value = false

      // Destroy old player resources after transition fades out
      if (oldKey && oldKey !== targetKey) {
        setTimeout(() => {
          destroyPlayerInstance(oldKey)
        }, 600)
      }
    }
    targetVideoEl.addEventListener('playing', players[targetKey].playingListener)

    // === WEBRTC LOGIC (OBS) ===
    if (props.streamUrl.startsWith('webrtc://')) {
      try {
        const rtcConnection = new RTCPeerConnection()
        players[targetKey].rtc = rtcConnection

        rtcConnection.addTransceiver("audio", {direction: "recvonly"})
        rtcConnection.addTransceiver("video", {direction: "recvonly"})

        rtcConnection.ontrack = (event) => {
          targetVideoEl.srcObject = event.streams[0]
          
          // Apply playoutDelayHint to jitter buffer (500ms of buffering for smooth play)
          if (event.receiver && 'playoutDelayHint' in event.receiver) {
            event.receiver.playoutDelayHint = 0.5
          }
        }

        const offer = await rtcConnection.createOffer()
        await rtcConnection.setLocalDescription(offer)

        const apiUrl = window.location.origin + "/rtc/v1/play/"
        const response = await fetch(apiUrl, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            api: apiUrl,
            streamurl: props.streamUrl,
            sdp: offer.sdp
          })
        })

        if (!response.ok) {
          throw new Error("WebRTC negotiation failed")
        }

        const data = await response.json()
        await rtcConnection.setRemoteDescription(new RTCSessionDescription({
          type: 'answer',
          sdp: data.sdp
        }))

        targetVideoEl.play().catch(e => console.log("Autoplay blocked"))

        rtcConnection.oniceconnectionstatechange = () => {
          if (rtcConnection && (rtcConnection.iceConnectionState === 'disconnected' || rtcConnection.iceConnectionState === 'failed')) {
            if (activeVideo.value === targetKey) {
              hasError.value = true
            }
            players[targetKey].retryTimer = setTimeout(initPlayer, 3000)
          }
        }
      } catch (err) {
        console.error("WebRTC Error:", err)
        if (activeVideo.value === targetKey || !activeVideo.value) {
          hasError.value = true
        }
        players[targetKey].retryTimer = setTimeout(initPlayer, 3000)
      }
      return
    }

    // === HLS LOGIC (IPTV PROXY) ===
    if (Hls.isSupported()) {
      const hlsInstance = new Hls({
        maxBufferLength: 30,
        maxMaxBufferLength: 60,
        enableWorker: true,
        lowLatencyMode: false,
        manifestLoadingMaxRetry: 10,
        fragLoadingMaxRetry: 10,
        levelLoadingMaxRetry: 10,
        liveSyncDurationCount: 5,
        capLevelToPlayerSize: true,
        abrEwmaDefaultEstimate: 500000,
      })
      players[targetKey].hls = hlsInstance

      hlsInstance.loadSource(props.streamUrl)
      hlsInstance.attachMedia(targetVideoEl)

      hlsInstance.on(Hls.Events.MANIFEST_PARSED, (event, data) => {
        if (data.levels && data.levels.length > 0) {
          qualities.value = data.levels.map(l => ({ height: l.height, bitrate: l.bitrate }))
        }
        targetVideoEl.play().catch(e => console.log("Autoplay blocked"))
      })

      hlsInstance.on(Hls.Events.ERROR, (event, data) => {
        if (data.fatal) {
          if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
            hlsInstance.recoverMediaError();
            return;
          }
          if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
            hlsInstance.startLoad();
            return;
          }
          if (activeVideo.value === targetKey || !activeVideo.value) {
            hasError.value = true
          }
          players[targetKey].retryTimer = setTimeout(initPlayer, 3000)
        }
      })
    } else if (targetVideoEl.canPlayType('application/vnd.apple.mpegurl')) {
      targetVideoEl.src = props.streamUrl
      targetVideoEl.addEventListener('loadedmetadata', () => {
        targetVideoEl.play().catch(e => console.log("Autoplay blocked"))
      })
    }
  }

  const destroyPlayer = () => {
    destroyPlayerInstance('A')
    destroyPlayerInstance('B')
    activeVideo.value = null
  }
  ```

- [ ] **Step 3: Update CSS styling in VideoPlayer.vue**
  Set up the transitions so that inactive video tags have `opacity: 0` and active ones have `opacity: 1` with a CSS transition.

  ```css
  .video-element {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    object-fit: contain;
    background: #000;
    opacity: 0;
    transition: opacity 0.5s ease-in-out;
    pointer-events: none;
  }

  .video-element.active {
    opacity: 1;
    pointer-events: auto;
  }
  ```

- [ ] **Step 4: Commit changes**

  ```bash
  git add frontend/src/components/VideoPlayer.vue
  git commit -m "feat: implement dual video crossfade player and playoutDelayHint for jitter buffering"
  ```

---

### Task 3: Build & Deploy Frontend changes

**Files:**
- None (deployment and testing task)

- [ ] **Step 1: Recompile the frontend Docker container**
  Run: `docker compose up -d --build frontend`
  Expected: Recompilation completes successfully, new container runs.

- [ ] **Step 2: Verify both HLS and WebRTC streams**
  Test switching between live feeds to verify the transition is visual crossfade instead of black screen and loading screen flash.
