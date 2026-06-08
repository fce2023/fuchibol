<template>
  <div class="player-wrapper">
    <video 
      ref="videoRef" 
      class="video-element" 
      controls 
      autoplay 
      playsinline
    ></video>
    <div v-if="hasError" class="overlay error-overlay">
      <p class="error-msg">Conexión con la transmisión perdida.</p>
      <p class="retry-msg">Reconectando automáticamente...</p>
    </div>
    <div v-else-if="!isLive" class="overlay offline-overlay">
      <span class="live-dot"></span>
      <span class="offline-msg">EL STREAM ESTÁ OFFLINE</span>
      <p class="offline-desc">Sintonizando señal en vivo...</p>
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
  }
})

const videoRef = ref(null)
const hasError = ref(false)
let hlsInstance = null
let retryTimer = null

const initPlayer = () => {
  if (retryTimer) {
    clearTimeout(retryTimer)
    retryTimer = null
  }

  if (!props.isLive || !props.streamUrl) {
    destroyPlayer()
    return
  }

  const video = videoRef.value
  if (!video) return

  destroyPlayer()
  hasError.value = false

  if (Hls.isSupported()) {
    hlsInstance = new Hls({
      maxBufferLength: 10,
      maxMaxBufferLength: 15,
      enableWorker: true,
      lowLatencyMode: true,
    })

    hlsInstance.loadSource(props.streamUrl)
    hlsInstance.attachMedia(video)

    hlsInstance.on(Hls.Events.ERROR, (event, data) => {
      console.warn('HLS Error:', data)
      if (data.fatal) {
        hasError.value = true
        switch (data.type) {
          case Hls.ErrorTypes.NETWORK_ERROR:
            console.log('Network error, attempting recovery...')
            hlsInstance.startLoad()
            break;
          case Hls.ErrorTypes.MEDIA_ERROR:
            console.log('Media error, attempting recovery...')
            hlsInstance.recoverMediaError()
            break;
          default:
            console.log('Unrecoverable error, reloading stream in 3s...')
            retryTimer = setTimeout(initPlayer, 3000)
            break;
        }
      }
    })
  } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
    // Native HLS support
    video.src = props.streamUrl
    video.addEventListener('error', handleNativeError)
  }
}

const handleNativeError = () => {
  hasError.value = true
  console.log('Native HLS error, retrying in 3s...')
  retryTimer = setTimeout(initPlayer, 3000)
}

const destroyPlayer = () => {
  if (hlsInstance) {
    hlsInstance.destroy()
    hlsInstance = null
  }
  const video = videoRef.value
  if (video) {
    video.removeAttribute('src')
    video.load()
    video.removeEventListener('error', handleNativeError)
  }
}

watch(() => [props.streamUrl, props.isLive], () => {
  initPlayer()
}, { deep: true })

onMounted(() => {
  initPlayer()
})

onBeforeUnmount(() => {
  destroyPlayer()
  if (retryTimer) clearTimeout(retryTimer)
})
</script>

<style scoped>
.player-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
  background: #000;
}

.video-element {
  width: 100%;
  height: 100%;
  object-fit: contain;
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
  background: radial-gradient(circle, rgba(21, 24, 33, 0.95) 0%, rgba(8, 10, 15, 0.98) 100%);
  text-align: center;
  padding: 20px;
  z-index: 2;
}

.live-dot {
  width: 12px;
  height: 12px;
  background: hsl(var(--text-muted));
  border-radius: 50%;
  margin-bottom: 15px;
}

.offline-msg, .error-msg {
  font-size: 20px;
  font-weight: 800;
  letter-spacing: 0.05em;
  color: hsl(var(--text-secondary));
  margin-bottom: 8px;
}

.offline-desc, .retry-msg {
  font-size: 13px;
  color: hsl(var(--text-muted));
}

.retry-msg {
  color: hsl(var(--primary));
  font-weight: 600;
}
</style>
