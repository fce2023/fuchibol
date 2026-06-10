<template>
  <div class="fuchibbol-layout">
    <!-- Top Bar -->
    <div class="topbar">
      <div class="logo">
        <div class="logo-dot"></div>
        <span class="logo-text">FUCHIBBOL</span>
      </div>
      <div class="topbar-right">
        <div class="user-badge" v-if="currentUser">
          <div class="user-avatar">{{ currentUser.username.substring(0, 1).toUpperCase() }}</div>
          <span class="user-name">@{{ currentUser.username }}</span>
        </div>
        <button @click="installApp" class="follow-btn" style="padding: 5px 10px; margin-right: 8px; font-size: 11px;">Instalar App</button>
        <router-link v-if="currentUser && ['admin', 'streamer'].includes(currentUser.role)" to="/admin" class="exit-btn" style="text-decoration:none; margin-right: 8px;">Panel</router-link>
        <button v-if="currentUser" @click="logout" class="exit-btn">Salir</button>
        <router-link v-else to="/login" class="exit-btn" style="text-decoration:none; color: #00e87a; border-color: #00e87a;">Entrar</router-link>
      </div>
    </div>

    <template v-if="channelInfo">
      <div class="home-grid">
        <div class="home-main">
          <!-- Video Player -->
      <div class="video-wrap">
        <VideoPlayer :streamUrl="playbackUrl" :isLive="channelInfo.is_live" />
        <div class="video-overlay" v-if="channelInfo.is_live"></div>
        <div class="video-top" v-if="channelInfo.is_live">
          <div class="live-badge">
            <div class="live-dot"></div>
            <span class="live-text">EN VIVO</span>
          </div>
          <div class="viewers">
            <div class="viewers-dot"></div>
            <span class="viewers-count">{{ viewerCount }} espectadores</span>
          </div>
        </div>
      </div>

      <!-- Channel Row -->
      <div class="channel-row">
        <div class="channel-icon">{{ channelOwner.substring(0, 2).toUpperCase() }}</div>
        <div class="channel-info">
          <div class="channel-name">{{ channelInfo.name }}</div>
          <div class="channel-meta">
            <span class="channel-handle">@{{ channelOwner }}</span>
            <span class="channel-followers">· {{ followerCount }} seguidores</span>
            <span class="channel-live-tag" v-if="channelInfo.is_live">VIVO</span>
          </div>
        </div>
        <button class="follow-btn" @click="toggleFollow" :style="isFollowing ? 'background: rgba(0,232,122,0.1)' : ''">
          {{ isFollowing ? 'Siguiendo' : 'Seguir' }}
        </button>
      </div>

      <!-- About -->
      <div class="section-label">SOBRE EL CANAL</div>
      <div class="streamer-bio">{{ channelInfo.description || 'Este canal no tiene descripción.' }}</div>

        </div>
        <div class="home-sidebar">
      <!-- Match Banner (Agenda) -->
      <div class="match-banner" v-for="(event, index) in parsedAgendaEvents" :key="index">
        <div class="match-flags">{{ event.flags }}</div>
        <div class="match-info">
          <div class="match-title">{{ event.teams }}</div>
          <div class="match-time" :style="event.isActive ? 'color: #e83d00;' : ''">{{ event.time }}</div>
        </div>
        <div class="match-pill" v-if="event.isActive" style="color: #e83d00; background: rgba(232,61,0,0.1); border-color: rgba(232,61,0,0.3);">AHORA</div>
        <div class="match-pill" v-else>PRÓXIMO</div>
      </div>


          <!-- Chat -->
      <ChatBox 
        :channelId="channelInfo.id" 
        :currentUser="currentUser" 
        @viewer-update="handleViewerUpdate" 
      />
        </div>
      </div>
    </template>

    <div v-else class="video-placeholder" style="flex-direction: column; height: 100vh;">
      <div class="logo-dot"></div>
      <div style="margin-top:15px; color:#c0c0d8; font-size:14px;">Cargando estadio...</div>
    </div>
    
    <!-- WHATSAPP FLOATING BUTTON -->
    <a 
      v-if="channelInfo && channelInfo.whatsapp_link" 
      :href="channelInfo.whatsapp_link" 
      target="_blank" 
      class="wsp-float-btn"
      title="Únete a nuestro grupo de WhatsApp"
    >
      <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
        <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
      </svg>
    </a>

    <!-- TIKTOK FLOATING BUTTON -->
    <a 
      v-if="channelInfo && channelInfo.tiktok_link" 
      :href="channelInfo.tiktok_link" 
      target="_blank" 
      class="tiktok-float-btn"
      title="Síguenos en TikTok"
    >
      <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
        <path d="M12.525.02c1.31-.02 2.61-.01 3.91-.02.08 1.53.63 3.09 1.75 4.17 1.12 1.11 2.7 1.62 4.24 1.79v4.03c-1.44-.05-2.89-.35-4.2-.97-.57-.26-1.1-.59-1.62-.93-.01 2.92.01 5.84-.02 8.75-.08 1.4-.54 2.79-1.35 3.94-1.31 1.92-3.58 3.17-5.91 3.21-1.43.08-2.86-.31-4.08-1.03-2.02-1.19-3.44-3.37-3.65-5.71-.02-.5-.03-1-.01-1.49.18-1.9 1.12-3.72 2.58-4.96 1.66-1.44 3.98-2.13 6.15-1.72.02 1.48-.04 2.96-.04 4.44-.9-.32-1.98-.23-2.81.33-.85.51-1.44 1.43-1.58 2.41-.05.38-.05.77-.01 1.14.12 1.25.96 2.37 2.14 2.78.47.16.97.21 1.46.2.9-.03 1.76-.36 2.44-1.04.72-.73 1.1-1.74 1.11-2.75V0l.04.02z"/>
      </svg>
    </a>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import VideoPlayer from '../components/VideoPlayer.vue'
import ChatBox from '../components/ChatBox.vue'

const router = useRouter()

const currentUser = ref(null)
const channelInfo = ref(null)
const channelOwner = ref('')
const playbackUrl = ref('')
const viewerCount = ref(0)
const followerCount = ref(0)
const isFollowing = ref(false)
const deferredPrompt = ref(null)
const isStandalone = ref(false)

const installApp = async () => {
  if (deferredPrompt.value) {
    deferredPrompt.value.prompt()
    const { outcome } = await deferredPrompt.value.userChoice
    if (outcome === 'accepted') {
      deferredPrompt.value = null
    }
  } else {
    alert("Para instalar Fuchibol como App en tu dispositivo:\n\nToca el menú de opciones (los tres puntos o el ícono de Compartir) y selecciona 'Añadir a la pantalla de inicio' o 'Instalar aplicación'.")
  }
}

const parsedAgendaEvents = computed(() => {
  if (!channelInfo.value || !channelInfo.value.agenda_events) return []
  try {
    return JSON.parse(channelInfo.value.agenda_events)
  } catch (e) {
    return []
  }
})

const handleViewerUpdate = (count) => {
  viewerCount.value = count
}

const toggleFollow = async () => {
  if (!currentUser.value) {
    router.push('/login')
    return
  }

  const method = isFollowing.value ? 'unfollow' : 'follow'
  try {
    const res = await fetch(`/api/v1/channels/${channelInfo.value.id}/${method}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${currentUser.value.token}`
      }
    })
    if (res.ok) {
      isFollowing.value = !isFollowing.value
      fetchFollowStatus()
    }
  } catch (err) {
    console.error(err)
  }
}

const fetchFollowStatus = async () => {
  if (!channelInfo.value) return
  try {
    const headers = {}
    if (currentUser.value) {
      headers['Authorization'] = `Bearer ${currentUser.value.token}`
    }
    const res = await fetch(`/api/v1/channels/${channelInfo.value.id}/following`, {
      headers
    })
    if (res.ok) {
      const data = await res.json()
      isFollowing.value = data.is_following
      followerCount.value = data.follower_count
    }
  } catch (err) {
    console.error(err)
  }
}

const logout = () => {
  localStorage.removeItem('fuchibol_user')
  currentUser.value = null
  router.push('/')
}

const fetchChannel = async () => {
  try {
    const primaryRes = await fetch(`/api/v1/channels/primary`)
    if (!primaryRes.ok) {
      throw new Error('No hay canales configurados')
    }
    const primaryData = await primaryRes.json()
    channelOwner.value = primaryData.username

    const res = await fetch(`/api/v1/channels/by-username/${primaryData.username}`)
    if (!res.ok) {
      throw new Error('Canal no encontrado')
    }
    const data = await res.json()
    
    // Fetch playback info before updating channelInfo to avoid reactivity flapping
    const playRes = await fetch(`/api/v1/streams/playback/${data.id}`)
    if (playRes.ok) {
      const playData = await playRes.json()
      
      // Setup playback URL using webrtc or hls based on stream_type
      const currentUrl = playData.stream_type === 'webrtc' ? playData.webrtc : playData.playback_url;
      if (playbackUrl.value !== currentUrl) {
        playbackUrl.value = currentUrl
      }
      
      data.is_live = playData.is_live
    }

    // Now update channelInfo to trigger a single, accurate reactive update
    channelInfo.value = data

    // Only fetch follow status once or when needed
    if (followerCount.value === 0) {
      fetchFollowStatus()
    }
  } catch (err) {
    console.error(err)
  }
}

let pollInterval = null

onMounted(() => {
  // Verificamos si la app ya está instalada o corriendo en modo standalone
  if (window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone) {
    isStandalone.value = true
  }

  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault()
    deferredPrompt.value = e
  })

  const userObj = localStorage.getItem('fuchibol_user')
  if (userObj) {
    currentUser.value = JSON.parse(userObj)
  }
  fetchChannel()
  pollInterval = setInterval(fetchChannel, 10000)
})

onBeforeUnmount(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
/* ── Top Bar ── */
.topbar {
  background: #0d0d18;
  padding: 14px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 0.5px solid #1a1a28;
}
.logo {
  display: flex;
  align-items: center;
  gap: 8px;
}
.logo-dot {
  width: 8px;
  height: 8px;
  background: #00e87a;
  border-radius: 50%;
  animation: pulse-dot 2s infinite;
}
@keyframes pulse-dot {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(0,232,122,0.4); }
  50%       { box-shadow: 0 0 0 6px rgba(0,232,122,0); }
}
.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 1px;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.user-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #1a1a2e;
  border: 0.5px solid #2a2a40;
  border-radius: 20px;
  padding: 5px 10px;
}
.user-avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #00e87a;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #002f18;
}
.user-name {
  font-size: 12px;
  color: #c0c0d8;
}
.exit-btn {
  font-size: 12px;
  color: #606080;
  border: 0.5px solid #2a2a40;
  border-radius: 12px;
  padding: 5px 10px;
  background: none;
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s;
}
.exit-btn:hover { color: #e0e0f0; border-color: #4a4a60; }

/* ── Video Player ── */
.video-wrap {
  position: relative;
  background: #000;
  overflow: hidden;
}
.video-placeholder {
  width: 100%;
  height: 210px;
  background: #080e1a;
  display: flex;
  align-items: center;
  justify-content: center;
}
.video-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to bottom, rgba(0,0,0,0.25) 0%, transparent 40%, rgba(0,0,0,0.65) 100%);
  pointer-events: none;
}
.video-top {
  position: absolute;
  top: 10px;
  left: 12px;
  right: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  z-index: 10;
}
.live-badge {
  display: flex;
  align-items: center;
  gap: 5px;
  background: #e83d00;
  border-radius: 4px;
  padding: 3px 8px;
}
.live-dot {
  width: 5px;
  height: 5px;
  background: #fff;
  border-radius: 50%;
  animation: pulse-dot 1.4s infinite;
}
.live-text {
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 1px;
}
.viewers {
  display: flex;
  align-items: center;
  gap: 5px;
  background: rgba(0,0,0,0.55);
  border-radius: 20px;
  padding: 3px 10px;
}
.viewers-dot {
  width: 6px;
  height: 6px;
  background: #00e87a;
  border-radius: 50%;
}
.viewers-count { font-size: 11px; color: rgba(255,255,255,0.85); font-weight: 500; }

/* ── Match Banner ── */
.match-banner {
  background: #0e1528;
  border: 0.5px solid #1d2540;
  margin: 12px 12px 0;
  border-radius: 10px;
  padding: 10px 14px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.match-flags {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 20px;
}
.match-info { flex: 1; }
.match-title { font-size: 13px; font-weight: 600; color: #e0e0f0; }
.match-time  { font-size: 11px; color: #00e87a; margin-top: 2px; font-weight: 500; }
.match-pill {
  background: rgba(0,232,122,0.1);
  border: 0.5px solid rgba(0,232,122,0.3);
  border-radius: 20px;
  padding: 3px 10px;
  font-size: 10px;
  color: #00e87a;
  font-weight: 700;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

/* ── Channel Row ── */
.channel-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: #0d0d18;
  margin: 12px 12px 0;
  border-radius: 10px;
  border: 0.5px solid #1a1a28;
}
.channel-icon {
  width: 46px;
  height: 46px;
  background: #001a0d;
  border: 0.5px solid rgba(0,232,122,0.2);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  color: #00e87a;
  letter-spacing: 0.5px;
  flex-shrink: 0;
}
.channel-info { flex: 1; min-width: 0; }
.channel-name { font-size: 13px; font-weight: 600; color: #e0e0f0; }
.channel-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 3px;
  flex-wrap: wrap;
}
.channel-handle   { font-size: 11px; color: #505070; }
.channel-followers{ font-size: 11px; color: #505070; }
.channel-live-tag {
  background: #e83d00;
  border-radius: 3px;
  padding: 1px 6px;
  font-size: 9px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 0.5px;
}
.follow-btn {
  background: transparent;
  border: 1.5px solid #00e87a;
  border-radius: 20px;
  padding: 6px 14px;
  font-size: 12px;
  font-weight: 600;
  color: #00e87a;
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
  white-space: nowrap;
}
.follow-btn:hover { background: rgba(0,232,122,0.1); }

/* ── Section Label ── */
.section-label {
  font-size: 10px;
  font-weight: 700;
  color: #404060;
  letter-spacing: 1.5px;
  padding: 14px 16px 4px;
}
.streamer-bio {
  font-size: 13px;
  color: #8080a0;
  padding: 4px 16px 14px;
  line-height: 1.6;
}

/* Floating Buttons */
.wsp-float-btn,
.tiktok-float-btn {
  position: fixed;
  right: 20px;
  width: 50px;
  height: 50px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.3);
  transition: transform 0.2s;
  z-index: 1000;
}

.wsp-float-btn {
  bottom: 20px;
  background-color: #25D366;
}

.tiktok-float-btn {
  bottom: 80px;
  background-color: #000000;
  border: 1px solid rgba(255,255,255,0.2);
}

.wsp-float-btn:hover,
.tiktok-float-btn:hover {
  transform: scale(1.1);
}

@media (min-width: 1024px) {
  .home-grid {
    display: grid;
    grid-template-columns: 7fr 3fr;
    gap: 0;
    align-items: stretch;
  }
  .home-main {
    border-right: 0.5px solid #1a1a28;
    display: flex;
    flex-direction: column;
  }
  .home-sidebar {
    display: flex;
    flex-direction: column;
  }
  .video-wrap {
    height: auto;
    aspect-ratio: 16/9;
  }
}
</style>

