<template>
  <div class="channel-layout">
    <!-- Navbar -->
    <header class="navbar glass-panel">
      <div class="brand">
        <router-link to="/" class="logo">⚽ FUCHIBOL</router-link>
      </div>
      <div class="user-actions">
        <span v-if="currentUser" class="username-badge">@{{ currentUser.username }}</span>
        <button v-if="currentUser" @click="logout" class="btn btn-secondary btn-sm">Salir</button>
        <router-link v-else to="/" class="btn btn-primary btn-sm">Iniciar Sesión</router-link>
      </div>
    </header>

    <!-- Main Content Grid -->
    <main v-if="channelInfo" class="content-grid">
      <!-- Player and Stream Info Column -->
      <section class="stream-section">
        <div class="video-container glass-card">
          <VideoPlayer :streamUrl="playbackUrl" :isLive="channelInfo.is_live" />
        </div>

        <div class="channel-info glass-card">
          <div class="channel-meta">
            <div class="avatar">
              {{ routeUser.substring(0, 2).toUpperCase() }}
            </div>
            <div class="details">
              <h1 class="stream-title">{{ channelInfo.name }}</h1>
              <div class="details-row">
                <span class="streamer-name">@{{ routeUser }}</span>
                <span v-if="channelInfo.category" class="category-badge">{{ channelInfo.category }}</span>
                <span :class="['live-status-badge', { 'is-live': channelInfo.is_live }]">
                  {{ channelInfo.is_live ? 'EN VIVO' : 'OFFLINE' }}
                </span>
                <span class="viewer-count">
                  <span :class="['dot', { 'red': channelInfo.is_live }]"></span> 
                  {{ viewerCount }} espectadores
                </span>
              </div>
            </div>
          </div>
          <div class="actions">
            <button class="btn btn-secondary">Seguir</button>
          </div>
        </div>

        <!-- Description Section -->
        <div class="description-card glass-card">
          <h3>Sobre el Streamer</h3>
          <p>{{ channelInfo.description || 'Este canal no tiene descripción.' }}</p>
        </div>

        <!-- Streamer Dashboard / Keys Panel (Only shown to channel owner) -->
        <div v-if="isOwner" class="streamer-panel glass-card">
          <div class="panel-header">
            <h3>Panel de Transmisión (Privado)</h3>
            <button @click="rotateKey" class="btn btn-secondary btn-sm danger-text">Rotar Clave</button>
          </div>
          <p class="panel-desc">Configura tu software de transmisión (ej. OBS) con estos valores:</p>
          <div class="key-field">
            <label>URL del Servidor (Ingesta):</label>
            <div class="input-copy-group">
              <input type="text" readonly :value="serverUrl" class="input-field" />
              <button @click="copyText(serverUrl)" class="btn btn-secondary">Copiar</button>
            </div>
          </div>
          <div class="key-field">
            <label>Clave de Transmisión (Stream Key):</label>
            <div class="input-copy-group">
              <input 
                :type="showKey ? 'text' : 'password'" 
                readonly 
                :value="streamKey || '••••••••••••••••••••••••••••••••'" 
                class="input-field key-input" 
              />
              <button @click="showKey = !showKey" class="btn btn-secondary">
                {{ showKey ? 'Ocultar' : 'Ver' }}
              </button>
              <button v-if="streamKey" @click="copyText(streamKey)" class="btn btn-secondary">Copiar</button>
              <button v-else @click="loadKey" class="btn btn-primary btn-sm">Revelar</button>
            </div>
            <span class="help-text">Instrucciones: El HLS seguro requiere transmitir al nombre: <code>channel_{{ channelInfo.id }}?key=[Tu_Clave]</code></span>
          </div>
        </div>
      </section>

      <!-- Chat Column -->
      <aside class="chat-section glass-panel">
        <ChatBox :channelId="channelInfo.id" @viewer-update="handleViewerUpdate" />
      </aside>
    </main>
    <div v-else class="loading-screen">
      <div class="spinner"></div>
      <p>Cargando canal...</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import VideoPlayer from '../components/VideoPlayer.vue'
import ChatBox from '../components/ChatBox.vue'

const route = useRoute()
const router = useRouter()

const routeUser = computed(() => route.params.username || '')
const currentUser = ref(null)
const channelInfo = ref(null)
const playbackUrl = ref('')
const viewerCount = ref(0)

// Streamer credentials
const showKey = ref(false)
const streamKey = ref('')
const serverUrl = ref('rtmp://localhost/live')

const isOwner = computed(() => {
  return currentUser.value && currentUser.value.username.toLowerCase() === routeUser.value.toLowerCase()
})

const handleViewerUpdate = (count) => {
  viewerCount.value = count
}

const logout = () => {
  localStorage.removeItem('fuchibol_user')
  currentUser.value = null
  router.push('/')
}

const copyText = (text) => {
  navigator.clipboard.writeText(text)
  alert('¡Copiado al portapapeles!')
}

const fetchChannel = async () => {
  try {
    const res = await fetch(`/api/v1/channels/by-username/${routeUser.value}`)
    if (!res.ok) {
      throw new Error('Canal no encontrado')
    }
    const data = await res.json()
    channelInfo.value = data
    
    // Fetch playback info
    const playRes = await fetch(`/api/v1/streams/playback/${data.id}`)
    if (playRes.ok) {
      const playData = await playRes.json()
      playbackUrl.value = playData.playback_url
      channelInfo.value.is_live = playData.is_live
    }
  } catch (err) {
    console.error(err)
    alert('No se pudo cargar la información del canal.')
  }
}

const loadKey = async () => {
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    const res = await fetch(`/api/v1/auth/channel/key`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (res.ok) {
      const data = await res.json()
      alert(data.info)
    }
  } catch (err) {
    console.error(err)
  }
}

const rotateKey = async () => {
  if (!confirm('¿Seguro que deseas rotar la clave de transmisión? OBS se desconectará si estás transmitiendo.')) {
    return
  }
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    const res = await fetch(`/api/v1/auth/rotate-stream-key`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (res.ok) {
      const data = await res.json()
      streamKey.value = data.stream_key
      showKey.value = true
      alert('Clave de transmisión rotada exitosamente.')
    }
  } catch (err) {
    console.error(err)
  }
}

onMounted(() => {
  const userObj = localStorage.getItem('fuchibol_user')
  if (userObj) {
    currentUser.value = JSON.parse(userObj)
  }
  fetchChannel()
})

watch(() => route.params.username, () => {
  channelInfo.value = null
  playbackUrl.value = ''
  viewerCount.value = 0
  streamKey.value = ''
  showKey.value = false
  fetchChannel()
})
</script>

<style scoped>
.channel-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  border-radius: 0;
  border-bottom: 1px solid hsla(var(--text-primary), 0.05);
  height: 64px;
  z-index: 10;
}

.logo {
  font-size: 20px;
  font-weight: 800;
  text-decoration: none;
  background: linear-gradient(135deg, hsl(var(--primary)), hsl(var(--secondary)));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.username-badge {
  font-size: 14px;
  font-weight: 600;
  color: hsl(var(--primary));
  margin-right: 16px;
}

.btn-sm {
  padding: 6px 12px;
  font-size: 13px;
}

.content-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 20px;
  padding: 20px;
  overflow: hidden;
  height: calc(100vh - 64px);
}

@media (max-width: 1024px) {
  .content-grid {
    grid-template-columns: 1fr;
    grid-template-rows: 1fr 300px;
  }
}

.stream-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow-y: auto;
  padding-right: 4px;
}

.video-container {
  aspect-ratio: 16/9;
  background: #000;
  border-radius: var(--radius-md);
  overflow: hidden;
  position: relative;
}

.channel-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px;
}

.channel-meta {
  display: flex;
  align-items: center;
  gap: 16px;
}

.avatar {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, hsl(var(--primary)), hsl(var(--secondary)));
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: hsl(var(--bg-primary));
}

.stream-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 4px;
}

.details-row {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  flex-wrap: wrap;
}

.streamer-name {
  font-weight: 600;
  color: hsl(var(--text-primary));
}

.category-badge {
  background: hsla(var(--primary), 0.1);
  color: hsl(var(--primary));
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 600;
}

.live-status-badge {
  font-size: 11px;
  font-weight: 800;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  background: hsl(var(--bg-tertiary));
  color: hsl(var(--text-muted));
}

.live-status-badge.is-live {
  background: hsl(var(--danger));
  color: white;
}

.viewer-count {
  color: hsl(var(--text-muted));
  display: flex;
  align-items: center;
  gap: 6px;
}

.dot {
  width: 8px;
  height: 8px;
  background: hsl(var(--text-muted));
  border-radius: 50%;
}

.dot.red {
  background: hsl(var(--danger));
  box-shadow: 0 0 8px hsl(var(--danger));
}

.description-card {
  padding: 20px;
}

.description-card h3 {
  font-size: 15px;
  font-weight: 700;
  margin-bottom: 8px;
  text-transform: uppercase;
  color: hsl(var(--text-secondary));
}

.description-card p {
  font-size: 14px;
  color: hsl(var(--text-secondary));
  line-height: 1.6;
}

.streamer-panel {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: 20px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.danger-text {
  color: hsl(var(--danger));
  border-color: hsla(var(--danger), 0.2);
}

.danger-text:hover {
  background: hsla(var(--danger), 0.1);
  border-color: hsl(var(--danger));
}

.streamer-panel h3 {
  font-size: 16px;
  font-weight: 700;
}

.panel-desc {
  font-size: 13px;
  color: hsl(var(--text-secondary));
}

.key-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.key-field label {
  font-size: 12px;
  font-weight: 600;
  color: hsl(var(--text-secondary));
}

.input-copy-group {
  display: flex;
  gap: 8px;
}

.input-copy-group input {
  flex: 1;
  padding: 8px 12px;
  font-size: 13px;
}

.key-input {
  font-family: monospace;
}

.help-text {
  font-size: 11px;
  color: hsl(var(--text-muted));
}

.help-text code {
  background: hsl(var(--bg-secondary));
  padding: 2px 4px;
  border-radius: 4px;
}

.loading-screen {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid hsla(var(--text-primary), 0.1);
  border-top-color: hsl(var(--primary));
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
