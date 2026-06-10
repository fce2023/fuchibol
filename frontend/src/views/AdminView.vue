<template>
  <div class="admin-layout">
    <!-- Navbar -->
    <header class="navbar glass-panel">
      <div class="brand">
        <router-link to="/" class="logo">⚽ FUCHIBOL</router-link>
      </div>
      <div class="user-actions">
        <span v-if="currentUser" class="username-badge">@{{ currentUser.username }}</span>
        <router-link to="/" class="btn btn-primary btn-sm" style="margin-right:8px;">Ver Canal Público</router-link>
        <button v-if="currentUser" @click="logout" class="btn btn-secondary btn-sm">Salir</button>
      </div>
    </header>

    <!-- Main Content -->
    <main class="content-container">
      <div class="admin-header">
        <h1>Panel de Administración</h1>
      </div>

      <div class="admin-grid">
        <!-- User Profile Section -->
        <div class="streamer-panel glass-card">
          <div class="panel-header">
            <h3>Mi Perfil</h3>
          </div>
          <form class="profile-form" @submit.prevent="updateProfile">
            <div class="form-group">
              <label>Nombre de Usuario</label>
              <input v-model="profileData.username" type="text" class="input-field" autocomplete="username" placeholder="Username">
            </div>
            <div class="form-group">
              <label>Nombre de tu Canal</label>
              <input v-model="profileData.channelName" type="text" class="input-field" placeholder="Ej: Canal de Deportes">
            </div>
            <div class="form-group">
              <label>Correo Electrónico</label>
              <input v-model="profileData.email" type="email" class="input-field" autocomplete="email" placeholder="Email">
            </div>
            <div class="form-group">
              <label>Enlace Grupo WhatsApp</label>
              <input v-model="profileData.whatsappLink" type="url" class="input-field" placeholder="https://chat.whatsapp.com/...">
            </div>
            <div class="form-group">
              <label>Enlace TikTok</label>
              <input v-model="profileData.tiktokLink" type="url" class="input-field" placeholder="https://www.tiktok.com/@...">
            </div>
            <div class="form-group">
              <label>Nueva Contraseña (opcional)</label>
              <input v-model="profileData.password" type="password" class="input-field" autocomplete="new-password" placeholder="Dejar en blanco para no cambiar">
            </div>
            <button type="submit" class="btn btn-primary btn-sm" :disabled="isUpdatingProfile">
              {{ isUpdatingProfile ? 'Guardando...' : 'Guardar Cambios' }}
            </button>
          </form>
        </div>

        <!-- Streamer Dashboard / Keys Panel -->
        <div class="streamer-panel glass-card">
          <div class="panel-header">
            <h3>Claves de Transmisión (Privado)</h3>
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
                :value="formattedStreamKey || '••••••••••••••••••••••••••••••••'" 
                class="input-field key-input" 
              />
              <button @click="showKey = !showKey" class="btn btn-secondary">
                {{ showKey ? 'Ocultar' : 'Ver' }}
              </button>
              <button v-if="formattedStreamKey" @click="copyText(formattedStreamKey)" class="btn btn-secondary">Copiar</button>
              <button v-else @click="loadKey" class="btn btn-primary btn-sm">Revelar</button>
            </div>
            <span class="help-text">Instrucciones: Copia esta clave completa y pégala en el campo "Clave de retransmisión" de tu OBS.</span>
          </div>

          <div class="obs-optimizations mt-4 p-4 rounded-md" style="background: rgba(30,30,40,0.5); border-left: 4px solid var(--primary-color);">
            <h4 style="color: var(--primary-color); margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.5rem;">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>
              Optimizaciones para Ultra-Baja Latencia
            </h4>
            <ul style="font-size: 0.85rem; color: #a0a0b0; padding-left: 1.5rem; list-style-type: square; margin-bottom: 0;">
              <li><strong>Salida (Avanzado) -> Codificador:</strong> x264 (o NVENC).</li>
              <li><strong>Control de Frecuencia:</strong> CBR.</li>
              <li><strong>Intervalo de fotogramas clave:</strong> 1 segundo <em>(Crítico para WebRTC)</em>.</li>
              <li><strong>Perfil:</strong> baseline.</li>
              <li><strong>Sintonizar (Tune):</strong> zerolatency.</li>
              <li><strong>B-Frames:</strong> 0 <em>(Los cuadros B aumentan la latencia)</em>.</li>
            </ul>
          </div>
        </div>

        <!-- IPTV Manager Panel -->
        <div class="streamer-panel glass-card iptv-manager">
          <div class="panel-header">
            <h3>Gestor de Transmisión Externa (IPTV)</h3>
            <div class="restream-controls">
              <span v-if="isPausedByObs" class="status-badge paused">PAUSADO (OBS transmitiendo)</span>
              <span v-else-if="isRestreaming" class="status-badge running">EN DIRECTO (Decodificando)</span>
              <span v-else class="status-badge stopped">Apagado</span>
              
              <button v-if="!isRestreaming && !isPausedByObs" @click="startRestream" class="btn btn-primary btn-sm" :disabled="!activeIptvUrl || isProcessing">
                {{ isProcessing ? 'Iniciando...' : '▶ Decodificar' }}
              </button>
              <button v-else @click="stopRestream" class="btn btn-secondary btn-sm danger-text" :disabled="isProcessing">
                {{ isProcessing ? 'Deteniendo...' : '⏹ Detener' }}
              </button>
            </div>
          </div>
          <p class="panel-desc">Añade enlaces IPTV (HTTP/HTTPS) que se mostrarán automáticamente cuando no estés transmitiendo por OBS.</p>
          
          <div class="iptv-add-form">
            <input type="url" v-model="newIptvUrl" placeholder="https://ejemplo.com/stream.m3u8" class="input-field" />
            <button @click="addIptvUrl" class="btn btn-primary btn-sm" :disabled="!newIptvUrl.trim()">Añadir</button>
          </div>
          
          <div v-if="iptvUrls.length > 0" class="iptv-list">
            <div v-for="(url, index) in iptvUrls" :key="index" :class="['iptv-item', { active: url === activeIptvUrl }]">
              <input type="radio" :id="'url-'+index" :value="url" v-model="activeIptvUrl" @change="saveIptvSettings" />
              <label :for="'url-'+index" class="iptv-url-label">{{ url }}</label>
              <button @click="removeIptvUrl(index)" class="btn btn-secondary btn-sm danger-text">Eliminar</button>
            </div>
          </div>
          <div v-else class="empty-state">
            <p>No has guardado ninguna URL de IPTV.</p>
          </div>
        </div>

        <!-- Agenda Manager Panel -->
        <div class="streamer-panel glass-card agenda-manager">
          <div class="panel-header">
            <h3>Gestor de Agenda (Eventos)</h3>
            <button @click="addAgendaEvent" class="btn btn-primary btn-sm">Añadir Evento</button>
          </div>
          <p class="panel-desc">Configura los eventos que aparecerán debajo del video en la portada.</p>
          
          <div v-if="agendaEvents.length > 0" class="agenda-list">
            <div v-for="(event, index) in agendaEvents" :key="index" class="agenda-item">
              <div class="agenda-fields">
                <input type="text" v-model="event.flags" placeholder="Banderas (ej: 🇦🇷🇮🇸)" class="input-field small-input" />
                <input type="text" v-model="event.teams" placeholder="Equipos (ej: ARG vs ISL)" class="input-field" />
                <input type="text" v-model="event.time" placeholder="Hora/Lugar (ej: Hoy · Alabama)" class="input-field" />
                <label class="checkbox-label">
                  <input type="checkbox" v-model="event.isActive" /> Activo (En Vivo)
                </label>
              </div>
              <button @click="removeAgendaEvent(index)" class="btn btn-secondary btn-sm danger-text">X</button>
            </div>
          </div>
          <div v-else class="empty-state">
            <p>No hay eventos programados.</p>
          </div>
          <div class="panel-footer" style="margin-top: 15px; display: flex; justify-content: flex-end;">
            <button @click="saveAgendaSettings" class="btn btn-primary btn-sm" :disabled="isSavingAgenda">
              {{ isSavingAgenda ? 'Guardando...' : 'Guardar Agenda' }}
            </button>
          </div>
        </div>

        <!-- Followers Manager Panel -->
        <div class="streamer-panel glass-card followers-manager">
          <div class="panel-header">
            <h3>Mis Seguidores ({{ followers.length }})</h3>
          </div>
          <p class="panel-desc">Usuarios que han decidido seguir tu canal.</p>
          
          <div v-if="followers.length > 0" class="followers-list">
            <div v-for="follower in followers" :key="follower.id" class="follower-item">
              <div class="follower-avatar">{{ follower.username.substring(0, 1).toUpperCase() }}</div>
              <div class="follower-info">
                <span class="follower-username">@{{ follower.username }}</span>
              </div>
            </div>
          </div>
          <div v-else class="empty-state">
            <p>Aún no tienes seguidores. ¡Anima a tu audiencia a seguirte!</p>
          </div>
        </div>

        <div class="info-card glass-card">
          <h3>Gestión de Moderadores</h3>
          <p>La gestión de roles estará disponible en una próxima actualización.</p>
          <p>Para probar asignar un moderador ahora, utiliza la API: <code>POST /api/v1/admin/users/{user_id}/role</code> con el token de administrador.</p>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const currentUser = ref(null)

const channelId = ref('1')
const showKey = ref(false)
const streamKey = ref('')

const formattedStreamKey = computed(() => {
  if (!streamKey.value) return ''
  return `channel_${channelId.value}?key=${streamKey.value}`
})
const rtmpPort = 1936
const serverUrl = ref(`rtmp://${window.location.hostname || 'localhost'}:${rtmpPort}/live`)

const getRtmpServerUrl = () => {
  const hostname = window.location.hostname || 'localhost'
  return `rtmp://${hostname}:${rtmpPort}/live`
}

// IPTV refs
const iptvUrls = ref([])
const activeIptvUrl = ref(null)
const newIptvUrl = ref('')
const isRestreaming = ref(false)
const isPausedByObs = ref(false)
let statusInterval = null

// Profile refs
const profileData = ref({
  username: '',
  email: '',
  password: '',
  channelName: '',
  whatsappLink: '',
  tiktokLink: ''
})
const isUpdatingProfile = ref(false)

// Agenda refs
const agendaEvents = ref([])
const isSavingAgenda = ref(false)

// Followers refs
const followers = ref([])

const addAgendaEvent = () => {
  agendaEvents.value.push({
    flags: '',
    teams: '',
    time: '',
    isActive: false
  })
}

const removeAgendaEvent = (index) => {
  agendaEvents.value.splice(index, 1)
}

const saveAgendaSettings = async () => {
  if (isSavingAgenda.value) return
  isSavingAgenda.value = true
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    const res = await fetch('/api/v1/channels/me', {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        agenda_events: JSON.stringify(agendaEvents.value)
      })
    })
    
    if (res.ok) {
      alert('Agenda guardada exitosamente.')
    } else {
      alert('Error al guardar la agenda.')
    }
  } catch(e) {
    console.error(e)
  } finally {
    isSavingAgenda.value = false
  }
}

const updateProfile = async () => {
  if (isUpdatingProfile.value) return
  isUpdatingProfile.value = true
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const user = tokenObj ? JSON.parse(tokenObj) : null
    const token = user ? user.token : ''
    
    // Update User Profile
    const userPayload = {
      username: profileData.value.username,
      email: profileData.value.email
    }
    if (profileData.value.password) {
      userPayload.password = profileData.value.password
    }

    const resUser = await fetch('/api/v1/auth/me', {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(userPayload)
    })
    
    // Update Channel Name & WhatsApp
    const resChannel = await fetch('/api/v1/channels/me', {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        name: profileData.value.channelName,
        whatsapp_link: profileData.value.whatsappLink,
        tiktok_link: profileData.value.tiktokLink
      })
    })

    if (resUser.ok && resChannel.ok) {
      const data = await resUser.json()
      const updatedUser = { 
        ...user, 
        username: data.user.username, 
        email: data.user.email,
        token: data.token // Guarda el nuevo token
      }
      localStorage.setItem('fuchibol_user', JSON.stringify(updatedUser))
      currentUser.value = updatedUser
      profileData.value.password = ''
      alert('Perfil y canal actualizados exitosamente.')
      // Recargar la página para que el Chat y todo el sistema tome el nuevo token
      window.location.reload()
    } else {
      alert('Hubo un problema al actualizar algunos datos.')
    }
  } catch(e) {
    console.error(e)
    alert('Error de conexión')
  } finally {
    isUpdatingProfile.value = false
  }
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

const loadKey = async () => {
  if (streamKey.value) {
    showKey.value = true
    return
  }

  if (!confirm('No se puede revelar la clave actual porque se guarda de forma segura. ¿Deseas generar una nueva clave y usarla ahora?')) {
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
      alert('Clave generada y cargada. Cópiala y configúrala en OBS.')
    } else {
      const data = await res.json()
      alert('Error: ' + (data.detail || 'No se pudo generar la clave.'))
    }
  } catch (err) {
    console.error(err)
    alert('Error al generar la clave.')
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

const checkAccess = () => {
  const userObj = localStorage.getItem('fuchibol_user')
  if (userObj) {
    const parsed = JSON.parse(userObj)
    if (parsed.role !== 'admin' && parsed.role !== 'streamer') {
      router.push('/')
      return
    }
    currentUser.value = parsed
    profileData.value.username = parsed.username
    profileData.value.email = parsed.email
    loadChannelSettings(parsed.token)
  } else {
    router.push('/login')
  }
}

const loadChannelSettings = async (token) => {
  try {
    const res = await fetch('/api/v1/channels/me', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (res.ok) {
      const data = await res.json()
      channelId.value = data.id
      profileData.value.channelName = data.name
      profileData.value.whatsappLink = data.whatsapp_link || ''
      profileData.value.tiktokLink = data.tiktok_link || ''
      activeIptvUrl.value = data.active_iptv_url
      try {
        iptvUrls.value = JSON.parse(data.iptv_urls || "[]")
      } catch(e) {
        iptvUrls.value = []
      }
      try {
        agendaEvents.value = JSON.parse(data.agenda_events || "[]")
      } catch(e) {
        agendaEvents.value = []
      }
      fetchFollowers(data.id, token)
    }
  } catch (err) {
    console.error('Failed to load channel settings', err)
  }
}

const fetchFollowers = async (id, token) => {
  try {
    const res = await fetch(`/api/v1/channels/${id}/followers_list`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (res.ok) {
      followers.value = await res.json()
    }
  } catch (err) {
    console.error('Failed to load followers', err)
  }
}

const saveIptvSettings = async () => {
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    const res = await fetch('/api/v1/channels/me', {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        active_iptv_url: activeIptvUrl.value,
        iptv_urls: JSON.stringify(iptvUrls.value)
      })
    })
    
    if (!res.ok) {
      alert('Error al guardar la configuración de IPTV')
    } else if (isRestreaming.value || isPausedByObs.value) {
      // Si el decodificador está activo o pausado, lo reiniciamos con la nueva URL automáticamente
      await startRestream()
    }
  } catch(e) {
    console.error(e)
  }
}

const addIptvUrl = () => {
  const url = newIptvUrl.value.trim()
  if (url && !iptvUrls.value.includes(url)) {
    iptvUrls.value.push(url)
    newIptvUrl.value = ''
    if (!activeIptvUrl.value) {
      activeIptvUrl.value = url
    }
    saveIptvSettings()
  }
}

const removeIptvUrl = (index) => {
  const url = iptvUrls.value[index]
  iptvUrls.value.splice(index, 1)
  if (activeIptvUrl.value === url) {
    activeIptvUrl.value = iptvUrls.value.length > 0 ? iptvUrls.value[0] : null
  }
  saveIptvSettings()
}

const checkRestreamStatus = async () => {
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    const res = await fetch('/api/v1/restream/status', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      // Sincronizar con los campos correctos del backend
      isRestreaming.value = data.is_active_restream
      isPausedByObs.value = data.is_paused_by_obs
    }
  } catch(e) {
    console.error(e)
  }
}

const isProcessing = ref(false)

const startRestream = async () => {
  if (!activeIptvUrl.value || isProcessing.value) return
  isProcessing.value = true
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    const res = await fetch('/api/v1/restream/start', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        iptv_url: activeIptvUrl.value
      })
    })
    
    if (res.ok) {
      isRestreaming.value = true
      await checkRestreamStatus()
    } else {
      const data = await res.json()
      alert('Error: ' + data.detail)
    }
  } catch(e) {
    console.error(e)
  } finally {
    isProcessing.value = false
  }
}

const stopRestream = async () => {
  if (isProcessing.value) return
  isProcessing.value = true
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    const res = await fetch('/api/v1/restream/stop', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    })
    
    if (res.ok) {
      isRestreaming.value = false
      await checkRestreamStatus()
      alert('Decodificador detenido.')
    }
  } catch(e) {
    console.error(e)
  } finally {
    isProcessing.value = false
  }
}

import { onBeforeUnmount } from 'vue'

onMounted(() => {
  serverUrl.value = getRtmpServerUrl()
  checkAccess()
  statusInterval = setInterval(checkRestreamStatus, 5000)
})

onBeforeUnmount(() => {
  if (statusInterval) clearInterval(statusInterval)
})
</script>

<style scoped>
.admin-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow-y: auto;
  background-color: hsl(var(--bg-primary));
}

.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  border-radius: 0;
  border-bottom: 1px solid hsla(var(--text-primary), 0.05);
  height: 64px;
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

.content-container {
  padding: 40px;
  max-width: 1000px;
  margin: 0 auto;
  width: 100%;
}

.admin-header h1 {
  font-size: 24px;
  margin-bottom: 24px;
}

.admin-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 24px;
}

.streamer-panel, .info-card {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.streamer-panel h3, .info-card h3 {
  font-size: 18px;
  font-weight: 700;
}

.panel-desc {
  font-size: 14px;
  color: hsl(var(--text-secondary));
}

.key-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.key-field label {
  font-size: 13px;
  font-weight: 600;
  color: hsl(var(--text-secondary));
}

.input-copy-group {
  display: flex;
  gap: 12px;
}

.input-copy-group input {
  flex: 1;
  padding: 10px 14px;
  font-size: 14px;
}

.key-input {
  font-family: monospace;
  letter-spacing: 2px;
}

.help-text {
  font-size: 12px;
  color: hsl(var(--text-muted));
  margin-top: 4px;
}

.help-text code {
  background: hsl(var(--bg-secondary));
  padding: 2px 6px;
  border-radius: 4px;
  color: hsl(var(--primary));
}

.danger-text {
  color: hsl(var(--danger));
  border-color: hsla(var(--danger), 0.2);
}

.danger-text:hover {
  background: hsla(var(--danger), 0.1);
  border-color: hsl(var(--danger));
}

.info-card p {
  color: hsl(var(--text-secondary));
  font-size: 14px;
  line-height: 1.5;
}

/* Profile Form Styles */
.profile-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 600;
  color: hsl(var(--text-secondary));
}

/* IPTV Manager Styles */
.iptv-manager {
  margin-top: 10px;
}

.restream-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-badge {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.status-badge.running {
  background: hsla(var(--success), 0.2);
  color: hsl(var(--success));
  border: 1px solid hsla(var(--success), 0.5);
  animation: pulse 2s infinite;
}

.status-badge.paused {
  background: hsla(var(--warning), 0.2);
  color: hsl(var(--warning));
  border: 1px solid hsla(var(--warning), 0.5);
}

.status-badge.stopped {
  background: hsla(0, 0%, 50%, 0.2);
  color: hsl(var(--text-muted));
}

@keyframes pulse {
  0% { box-shadow: 0 0 0 0 hsla(var(--success), 0.4); }
  70% { box-shadow: 0 0 0 6px hsla(var(--success), 0); }
  100% { box-shadow: 0 0 0 0 hsla(var(--success), 0); }
}

.iptv-add-form {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.iptv-add-form input {
  flex: 1;
}

.iptv-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: hsla(0,0%,0%,0.2);
  padding: 12px;
  border-radius: 8px;
}

.iptv-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: 6px;
  background: hsla(0,0%,100%,0.02);
  border: 1px solid transparent;
}

.iptv-item.active {
  border-color: hsla(var(--primary), 0.5);
  background: hsla(var(--primary), 0.05);
}

.iptv-url-label {
  flex: 1;
  font-family: monospace;
  font-size: 13px;
  word-break: break-all;
  cursor: pointer;
}

.empty-state {
  text-align: center;
  padding: 20px;
  color: hsl(var(--text-muted));
  font-size: 14px;
  font-style: italic;
  background: hsla(0,0%,0%,0.2);
  border-radius: 8px;
}

/* Agenda List Styles */
.agenda-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.agenda-item {
  display: flex;
  align-items: center;
  gap: 12px;
  background: hsla(0,0%,0%,0.2);
  padding: 12px;
  border-radius: 8px;
}

.agenda-fields {
  display: flex;
  flex: 1;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.agenda-fields .input-field {
  flex: 1;
  min-width: 120px;
}

.agenda-fields .small-input {
  flex: 0 0 80px;
  min-width: 80px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: hsl(var(--text-secondary));
  cursor: pointer;
}

/* Followers List Styles */
.followers-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}

.follower-item {
  display: flex;
  align-items: center;
  gap: 10px;
  background: hsla(0,0%,0%,0.2);
  padding: 10px;
  border-radius: 10px;
  border: 1px solid hsla(var(--text-primary), 0.05);
}

.follower-avatar {
  width: 32px;
  height: 32px;
  background: hsl(var(--primary));
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 14px;
}

.follower-username {
  font-size: 13px;
  font-weight: 600;
  color: hsl(var(--text-primary));
}
</style>
