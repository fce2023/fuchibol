<template>
  <div class="player-container" :class="{ 'is-loading': isLoading && isLive }" @mouseenter="showControls = true" @mouseleave="showControls = false">
    <div class="player-aspect-ratio">
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

      <!-- Selector de Calidad (ABR) -->
      <div v-if="isLive && qualities.length > 1" class="quality-selector" :class="{ 'visible': showControls }">
        <select v-model="currentQuality" @change="changeQuality" class="quality-dropdown">
          <option :value="-1">Auto (Adaptativo)</option>
          <option v-for="(q, index) in qualities" :key="index" :value="index">
            {{ q.height }}p
          </option>
        </select>
        <div class="abr-badge" title="Adaptive Bitrate Activo">ABR</div>
      </div>

      <!-- Pantalla de Carga (Sincronizando) -->
      <div v-if="isLoading && isLive" class="overlay loading-overlay">
        <div class="spinner"></div>
        <p class="status-text">Sincronizando señal...</p>
      </div>

      <!-- Pantalla de Error -->
      <div v-if="hasError && isLive" class="overlay error-overlay">
        <div class="error-icon">⚠️</div>
        <h3 class="status-title">Error de Conexión</h3>
        <p class="status-text">Intentando reconectar automáticamente...</p>
        <button @click="initPlayer" class="retry-btn">Reintentar ahora</button>
      </div>

      <!-- Pantalla Offline -->
      <div v-if="!isLive" class="overlay offline-overlay">
        <div class="offline-logo">
          <div class="live-dot-pulse"></div>
          <span class="offline-tag">OFFLINE</span>
        </div>
        <h3 class="status-title">Esperando Transmisión</h3>
        <p class="status-text">La señal comenzará pronto. ¡No te muevas!</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import Hls from 'hls.js'

const props = defineProps({
  streamUrl: {
    type: String,
    required: true
  },
  isLive: {
    type: Boolean,
    default: false
  },
  channelId: {
    type: [Number, String],
    default: null
  }
})

const videoRefA = ref(null)
const videoRefB = ref(null)
const activeVideo = ref(null) // 'A' or 'B' or null
const hasError = ref(false)
const isLoading = ref(true)
const showControls = ref(false)
const qualities = ref([])
const currentQuality = ref(-1)

let hasTracked = false

// Track player structures
const players = {
  A: { hls: null, rtc: null, retryTimer: null, playingListener: null },
  B: { hls: null, rtc: null, retryTimer: null, playingListener: null }
}

const trackView = async () => {
  if (!props.channelId || hasTracked) return
  try {
    await fetch('/api/v1/analytics/track', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ channel_id: parseInt(props.channelId) })
    })
    hasTracked = true
  } catch (e) {
    console.error("Tracking error:", e)
  }
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

  // If there is no active video yet, we show loading screen
  if (!activeVideo.value) {
    isLoading.value = true
  }

  hasTracked = false

  // Set up playing event listener to trigger the crossfade transition
  players[targetKey].playingListener = () => {
    // Transition active reference
    const oldKey = activeVideo.value
    activeVideo.value = targetKey
    isLoading.value = false

    trackView()

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
        
        // Prioritize stability: set playoutDelayHint to 1.0 second to buffer incoming jitter
        if (event.receiver && 'playoutDelayHint' in event.receiver) {
          event.receiver.playoutDelayHint = 1.0
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
      maxBufferLength: 30, // Mayor buffer para absorber cortes
      maxMaxBufferLength: 60,
      enableWorker: true,
      lowLatencyMode: false, // DESACTIVAR baja latencia para IPTV pirata
      manifestLoadingMaxRetry: 10,
      fragLoadingMaxRetry: 10, // Reintentar fragmentos perdidos
      levelLoadingMaxRetry: 10,
      liveSyncDurationCount: 5, // Mantenerse un poco atrás del borde en vivo
      capLevelToPlayerSize: true,
      abrEwmaDefaultEstimate: 500000,
    })
    players[targetKey].hls = hlsInstance

    hlsInstance.loadSource(props.streamUrl)
    hlsInstance.attachMedia(targetVideoEl)

    hlsInstance.on(Hls.Events.MANIFEST_PARSED, (event, data) => {
      // Cargar calidades disponibles para el ABR
      if (data.levels && data.levels.length > 0) {
        qualities.value = data.levels.map(l => ({ height: l.height, bitrate: l.bitrate }))
      }
      targetVideoEl.play().catch(e => console.log("Autoplay blocked, waiting for interaction"))
    })

    hlsInstance.on(Hls.Events.ERROR, (event, data) => {
      if (data.fatal) {
        // Errores de medios se recuperan en silencio
        if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
          console.warn("IPTV Media Error (Salto de tiempo). Recuperando en silencio...");
          hlsInstance.recoverMediaError();
          return;
        }

        // Errores de red se recuperan en silencio intentando cargar de nuevo
        if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
          console.warn("IPTV Network Error. Reintentando carga en silencio...");
          hlsInstance.startLoad();
          return;
        }

        // Si es otro error fatal, mostramos pantalla de error
        if (activeVideo.value === targetKey || !activeVideo.value) {
          hasError.value = true
        }
        players[targetKey].retryTimer = setTimeout(initPlayer, 3000)
      }
    })

    // Detener la descarga de fragmentos si el usuario pone pausa
    targetVideoEl.addEventListener('pause', () => {
      if (hlsInstance) {
        hlsInstance.stopLoad()
      }
    })

    // Reanudar la descarga y forzar la sincronización al en vivo si da play
    targetVideoEl.addEventListener('play', () => {
      if (hlsInstance) {
        hlsInstance.startLoad()
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

watch([() => props.streamUrl, () => props.isLive], (newVals, oldVals) => {
  // Only re-init if the URL or live status actually changed
  if (newVals[0] !== oldVals[0] || newVals[1] !== oldVals[1]) {
    initPlayer()
  }
})

onMounted(() => {
  initPlayer()
})

onBeforeUnmount(() => {
  destroyPlayer()
})
</script>

<style scoped>
.player-container {
  width: 100%;
  border-radius: 12px;
  overflow: hidden;
  background: #000;
  box-shadow: 0 20px 50px rgba(0,0,0,0.5);
  border: 1px solid rgba(255,255,255,0.05);
}

.player-aspect-ratio {
  position: relative;
  width: 100%;
  padding-top: 56.25%; /* 16:9 Aspect Ratio */
}

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

.overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 15, 0.85);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  text-align: center;
  padding: 20px;
  z-index: 10;
  transition: all 0.3s ease;
}

/* Quality Selector */
.quality-selector {
  position: absolute;
  top: 15px;
  right: 15px;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 5;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.quality-selector.visible {
  opacity: 1;
}

.quality-dropdown {
  background: rgba(0, 0, 0, 0.6);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.2);
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  backdrop-filter: blur(4px);
  cursor: pointer;
  outline: none;
}

.quality-dropdown:hover {
  background: rgba(0, 0, 0, 0.8);
  border-color: var(--green);
}

.abr-badge {
  background: var(--green);
  color: #000;
  font-size: 10px;
  font-weight: 800;
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.5px;
}

/* Offline Styles */
.offline-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  background: rgba(255,255,255,0.05);
  padding: 6px 14px;
  border-radius: 20px;
  border: 1px solid rgba(255,255,255,0.1);
}

.live-dot-pulse {
  width: 10px;
  height: 10px;
  background: #444;
  border-radius: 50%;
}

.offline-tag {
  font-size: 12px;
  font-weight: 800;
  color: #888;
  letter-spacing: 1px;
}

.status-title {
  font-size: 22px;
  font-weight: 800;
  margin-bottom: 8px;
  color: #fff;
  text-shadow: 0 2px 10px rgba(0,0,0,0.5);
}

.status-text {
  font-size: 14px;
  color: rgba(255,255,255,0.6);
  max-width: 280px;
}

/* Loading Spinner */
.spinner {
  width: 50px;
  height: 50px;
  border: 3px solid rgba(var(--primary-rgb), 0.1);
  border-top-color: hsl(var(--primary));
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Error Styles */
.error-icon {
  font-size: 40px;
  margin-bottom: 10px;
}

.retry-btn {
  margin-top: 20px;
  padding: 10px 24px;
  border-radius: 30px;
  background: hsl(var(--primary));
  color: #fff;
  border: none;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.2s;
}

.retry-btn:hover {
  transform: scale(1.05);
}

/* Mobile Optimizations */
@media (max-width: 768px) {
  .status-title {
    font-size: 18px;
  }
  .status-text {
    font-size: 12px;
  }
  .player-container {
    border-radius: 0; /* Full width on mobile */
  }
}
</style>
