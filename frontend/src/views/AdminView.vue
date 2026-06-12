<template>
  <div class="admin-layout">

    <!-- ── SIDEBAR (desktop only) ── -->
    <aside class="sidebar">
      <div class="sidebar-logo">
        <div class="logo-dot"></div>
        <span class="logo-text">FUCHIBOL</span>
      </div>
      <nav class="sidebar-nav">
        <a 
          v-for="item in navItems" 
          :key="item.id"
          href="#"
          :class="['sidebar-nav-item', { active: activeTab === item.id }]"
          @click.prevent="selectTab(item.id)"
        >
          <i :class="item.icon"></i> {{ item.label }}
        </a>
      </nav>
      <div class="sidebar-footer">
        <div class="user-row">
          <div class="avatar-sm">{{ currentUser?.username?.substring(0, 1).toUpperCase() }}</div>
          <div>
            <div class="user-name">@{{ currentUser?.username }}</div>
            <div class="user-role">{{ currentUser?.role || 'ADMIN' }}</div>
          </div>
        </div>
        <button class="btn-logout" @click="logout">
          <i class="ti ti-logout"></i> Cerrar sesión
        </button>
      </div>
    </aside>

    <!-- ── MAIN ── -->
    <div class="main">

      <!-- Mobile top bar -->
      <header class="topbar">
        <div class="logo-row">
          <div class="logo-dot"></div>
          <span class="logo-text">FUCHIBOL</span>
        </div>
        <div class="topbar-right">
          <div class="icon-btn" @click="router.push('/')"><i class="ti ti-world"></i></div>
          <div class="avatar-sm" @click="activeTab = 'profile'">{{ currentUser?.username?.substring(0, 1).toUpperCase() }}</div>
        </div>
      </header>

      <!-- Desktop top bar -->
      <div class="desktop-topbar">
        <div>
          <div class="desktop-title">{{ currentNavLabel }}</div>
          <div class="desktop-sub">{{ currentNavDesc }}</div>
        </div>
        <button class="btn-public-desktop" @click="router.push('/')">
          <i class="ti ti-external-link" style="font-size:14px"></i>
          Ir al sitio público
        </button>
      </div>

      <!-- Mobile hero -->
      <div class="hero">
        <div class="hero-title">{{ currentNavLabel }}</div>
        <div class="hero-sub">{{ currentNavDesc }}</div>
        <div v-if="activeTab === 'dashboard' && (isRestreaming || isPausedByObs)" class="badge-live"><span class="live-dot"></span>Canal en directo</div>
      </div>

      <div class="tab-content-container">
        <!-- TAB: DASHBOARD -->
        <div v-if="activeTab === 'dashboard'" class="tab-pane">
          <!-- Stats -->
          <div class="stats-grid">
            <div class="stat-card">
              <div class="stat-icon-row">
                <div class="stat-icon teal"><i class="ti ti-eye"></i></div>
                <span class="stat-label">Vistas totales</span>
              </div>
              <div class="stat-value">{{ analyticsSummary.total_views }}</div>
              <div class="stat-delta">Vistas acumuladas</div>
            </div>
            <div class="stat-card">
              <div class="stat-icon-row">
                <div class="stat-icon blue"><i class="ti ti-clock"></i></div>
                <span class="stat-label">Últimas 24h</span>
              </div>
              <div class="stat-value">{{ analyticsSummary.views_last_24h }}</div>
              <div class="stat-delta">Vistas recientes</div>
            </div>
            <div class="stat-card wide">
              <div class="stat-icon-row">
                <div class="stat-icon purple"><i class="ti ti-users"></i></div>
                <span class="stat-label">Seguidores</span>
              </div>
              <div class="stat-value">{{ followers.length }}</div>
              <div class="stat-delta">Comunidad de tu canal</div>
            </div>
          </div>

          <!-- Info cards -->
          <div class="section">
            <div class="bottom-grid">
              <div class="info-card">
                <div class="info-card-title">
                  <i class="ti ti-broadcast"></i> Estado del canal
                </div>
                <div class="status-row">
                  <span class="status-label">Señal actual</span>
                  <span v-if="isRestreaming || isPausedByObs" class="badge-live" style="margin-top:0;font-size:11px;padding:3px 10px">
                    <span class="live-dot"></span>En directo
                  </span>
                  <span v-else class="status-val">Offline</span>
                </div>
                <div class="status-row">
                  <span class="status-label">Fuentes IPTV</span>
                  <span class="status-val">{{ iptvUrlsObj.length }} guardadas</span>
                </div>
              </div>

              <div class="info-card">
                <div class="info-card-title">
                  <i class="ti ti-world"></i> Tráfico por país
                </div>
                <div v-if="analyticsSummary.top_countries.length > 0" class="top-countries-list mt-2">
                  <div v-for="c in analyticsSummary.top_countries" :key="c.country" class="status-row" style="padding: 5px 0; border: none;">
                    <span class="status-label">{{ c.country }}</span>
                    <span class="status-val">{{ c.count }} vistas</span>
                  </div>
                </div>
                <div v-else class="empty-state">
                  <i class="ti ti-map-2 empty-icon"></i>
                  <div class="empty-text">Aún no hay datos de tráfico.<br>Transmite para ver de dónde te ven.</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- TAB: PROFILE -->
        <div v-if="activeTab === 'profile'" class="tab-pane section">
          <div class="info-card">
            <form class="profile-form" @submit.prevent="updateProfile">
              <div class="form-grid">
                <div class="form-group">
                  <label>Nombre de Usuario</label>
                  <input v-model="profileData.username" type="text" class="input-field" autocomplete="username">
                </div>
                <div class="form-group">
                  <label>Nombre de tu Canal</label>
                  <input v-model="profileData.channelName" type="text" class="input-field">
                </div>
                <div class="form-group">
                  <label>Correo Electrónico</label>
                  <input v-model="profileData.email" type="email" class="input-field" autocomplete="email">
                </div>
                <div class="form-group">
                  <label>Nueva Contraseña (opcional)</label>
                  <input v-model="profileData.password" type="password" class="input-field" autocomplete="new-password" placeholder="Dejar en blanco para no cambiar">
                </div>
              </div>
              
              <div class="divider"></div>
              
              <div class="form-grid">
                <div class="form-group">
                  <label>Enlace Grupo WhatsApp</label>
                  <input v-model="profileData.whatsappLink" type="url" class="input-field" placeholder="https://chat.whatsapp.com/...">
                </div>
                <div class="form-group">
                  <label>Enlace TikTok</label>
                  <input v-model="profileData.tiktokLink" type="url" class="input-field" placeholder="https://www.tiktok.com/@...">
                </div>
                <div class="form-group logo-upload-group">
                  <label>Logo del Canal (JPG/PNG)</label>
                  <div class="logo-preview-container">
                    <img v-if="profileData.logoUrl" :src="profileData.logoUrl" class="logo-preview" alt="Logo actual">
                    <div v-else class="logo-placeholder">Sin Logo</div>
                    <div class="upload-controls">
                      <input type="file" ref="logoInput" accept="image/jpeg, image/png" @change="handleLogoSelect" class="hidden-input">
                      <button type="button" @click="$refs.logoInput.click()" class="btn btn-secondary btn-sm">Seleccionar</button>
                      <button type="button" v-if="selectedLogoFile" @click="uploadLogo" class="btn btn-primary btn-sm ml-2" :disabled="isUploadingLogo">
                        {{ isUploadingLogo ? 'Subiendo...' : 'Optimizar y Subir' }}
                      </button>
                    </div>
                  </div>
                  <small v-if="selectedLogoFile" class="file-name-hint">Archivo: {{ selectedLogoFile.name }}</small>
                </div>
              </div>
              
              <div class="form-actions mt-4">
                <button type="submit" class="btn btn-primary w-full" :disabled="isUpdatingProfile">
                  {{ isUpdatingProfile ? 'Guardando...' : 'Guardar Cambios' }}
                </button>
              </div>
            </form>
          </div>
        </div>

        <!-- TAB: STREAM -->
        <div v-if="activeTab === 'stream'" class="tab-pane section">
          <!-- Keys Panel -->
          <div class="info-card mb-4">
            <div class="panel-header flex-between mb-4">
              <div class="info-card-title mb-0"><i class="ti ti-key"></i> Claves OBS</div>
              <button @click="rotateKey" class="btn btn-secondary btn-sm danger-text">Rotar Clave</button>
            </div>
            
            <div class="key-field mb-4">
              <label>Servidor RTMP</label>
              <div class="input-copy-group">
                <input type="text" readonly :value="serverUrl" class="input-field" />
                <button @click="copyText(serverUrl)" class="btn btn-secondary btn-sm">Copiar</button>
              </div>
            </div>
            
            <div class="key-field mb-4">
              <label>Clave de Transmisión</label>
              <div class="input-copy-group">
                <input 
                  :type="showKey ? 'text' : 'password'" 
                  readonly 
                  :value="formattedStreamKey || '••••••••••••••••••••••••••••••••'" 
                  class="input-field key-input" 
                />
                <button @click="showKey = !showKey" class="btn btn-secondary btn-sm">
                  {{ showKey ? 'Ocultar' : 'Ver' }}
                </button>
                <button v-if="formattedStreamKey" @click="copyText(formattedStreamKey)" class="btn btn-secondary btn-sm">Copiar</button>
                <button v-else @click="loadKey" class="btn btn-primary btn-sm">Revelar</button>
              </div>
            </div>
          </div>

          <!-- IPTV Manager -->
          <div class="info-card">
            <div class="panel-header flex-between mb-4">
              <div class="info-card-title mb-0"><i class="ti ti-device-tv"></i> Gestor IPTV</div>
              <div class="status-controls">
                <span v-if="isPausedByObs" class="status-badge paused">PAUSADO</span>
                <span v-else-if="isRestreaming" class="status-badge running">ACTIVO</span>
                <span v-else class="status-badge stopped">APAGADO</span>
              </div>
            </div>
            
            <div class="iptv-add-form-new mb-4">
              <div class="form-group mb-2">
                <label>Nombre del Canal TV</label>
                <input type="text" v-model="newIptvName" placeholder="Ej: ESPN Premium" class="input-field" />
              </div>
              <div class="form-group mb-2">
                <label>URL del Stream (m3u8)</label>
                <div class="input-copy-group">
                  <input type="url" v-model="newIptvUrl" placeholder="https://..." class="input-field" />
                  <button @click="addIptvUrl" class="btn btn-primary btn-sm" :disabled="!newIptvUrl.trim()">Añadir</button>
                </div>
              </div>
            </div>

            <div class="iptv-list-new mb-4">
              <div v-for="(item, index) in iptvUrlsObj" :key="index" :class="['iptv-item-new', { active: item.url === activeIptvUrl }]">
                <template v-if="editingIptvIndex === index">
                  <div class="edit-form-inline">
                    <input type="text" v-model="editingIptvData.name" class="input-field mb-2" placeholder="Nombre">
                    <input type="url" v-model="editingIptvData.url" class="input-field mb-2" placeholder="URL">
                    <div class="edit-actions">
                      <button @click="saveEditedIptv" class="btn btn-primary btn-sm">OK</button>
                      <button @click="cancelEditingIptv" class="btn btn-secondary btn-sm">Cancel</button>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div class="item-radio">
                    <input type="radio" :id="'url-'+index" :value="item.url" v-model="activeIptvUrl" @change="saveIptvSettings" />
                  </div>
                  <label :for="'url-'+index" class="item-details">
                    <span class="item-name">{{ item.name || 'Sin Nombre' }}</span>
                    <span class="item-url">{{ item.url }}</span>
                  </label>
                  <div class="item-actions">
                    <button @click="startEditingIptv(index)" class="btn-icon"><i class="ti ti-edit"></i></button>
                    <button @click="removeIptvUrl(index)" class="btn-icon danger"><i class="ti ti-trash"></i></button>
                  </div>
                </template>
              </div>
            </div>
            
            <div class="panel-footer mt-4">
              <button v-if="!isRestreaming && !isPausedByObs" @click="startRestream" class="btn btn-primary w-full" :disabled="!activeIptvUrl || isProcessing">
                {{ isProcessing ? 'Iniciando...' : 'Iniciar Decodificador' }}
              </button>
              <button v-else @click="stopRestream" class="btn btn-secondary danger-text w-full" :disabled="isProcessing">
                {{ isProcessing ? 'Deteniendo...' : 'Detener Decodificador' }}
              </button>
            </div>
          </div>
        </div>

        <!-- TAB: AGENDA -->
        <div v-if="activeTab === 'agenda'" class="tab-pane section">
          <div class="info-card">
            <div class="panel-header flex-between mb-4">
              <div class="info-card-title mb-0"><i class="ti ti-calendar-event"></i> Próximos Eventos</div>
              <button @click="addAgendaEvent" class="btn btn-primary btn-sm">+ Nuevo</button>
            </div>
            
            <div class="agenda-grid mt-4">
              <div v-for="(event, index) in agendaEvents" :key="index" class="agenda-card-new stat-card" style="padding: 16px;">
                <div class="event-fields">
                  <div class="form-group mb-2">
                    <label>Banderas</label>
                    <input type="text" v-model="event.flags" placeholder="🇦🇷 🇧🇷" class="input-field">
                  </div>
                  <div class="form-group mb-2">
                    <label>Equipos / Título</label>
                    <input type="text" v-model="event.teams" placeholder="Local vs Visitante" class="input-field">
                  </div>
                  <div class="form-group mb-2">
                    <label>Horario / Info</label>
                    <input type="text" v-model="event.time" placeholder="Hoy 20:00" class="input-field">
                  </div>
                </div>
                <div class="event-actions mt-4 flex-between">
                  <label class="toggle-switch">
                    <input type="checkbox" v-model="event.isActive">
                    <span class="slider"></span>
                    <span class="label">EN VIVO</span>
                  </label>
                  <button @click="removeAgendaEvent(index)" class="btn-icon danger">
                    <i class="ti ti-trash"></i>
                  </button>
                </div>
              </div>
            </div>
            
            <div v-if="agendaEvents.length === 0" class="empty-state">
              <i class="ti ti-calendar-off empty-icon"></i>
              <div class="empty-text">No hay eventos programados.</div>
            </div>

            <div class="form-actions mt-4">
              <button @click="saveAgendaSettings" class="btn btn-primary w-full" :disabled="isSavingAgenda">
                {{ isSavingAgenda ? 'Guardando...' : 'Guardar Agenda' }}
              </button>
            </div>
          </div>
        </div>

        <!-- TAB: FOLLOWERS -->
        <div v-if="activeTab === 'followers'" class="tab-pane section">
          <div class="info-card">
            <div class="info-card-title"><i class="ti ti-users"></i> Comunidad</div>
            <div class="followers-grid-new mt-4">
              <div v-for="follower in followers" :key="follower.id" class="follower-card-new stat-card" style="padding:10px; display:flex; gap:10px; align-items:center;">
                <div class="avatar-sm">{{ follower.username.substring(0, 1).toUpperCase() }}</div>
                <div class="info">
                  <div class="user-name">@{{ follower.username }}</div>
                  <div class="stat-delta">Desde {{ formatDate(follower.created_at) }}</div>
                </div>
              </div>
            </div>
            <div v-if="followers.length === 0" class="empty-state">
              <i class="ti ti-user-off empty-icon"></i>
              <div class="empty-text">Aún no tienes seguidores.</div>
            </div>
          </div>
        </div>
      </div>

    </div><!-- /main -->

    <!-- ── BOTTOM NAV (mobile only) ── -->
    <nav class="bottom-nav">
      <a 
        v-for="item in navItems" 
        :key="item.id"
        href="#"
        :class="['nav-tab', { active: activeTab === item.id }]"
        @click.prevent="selectTab(item.id)"
      >
        <i :class="item.icon"></i>
        <span>{{ item.label }}</span>
      </a>
    </nav>

  </div><!-- /app -->
</template>


<script setup>
import { ref, onMounted, computed, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const currentUser = ref(null)
const isSidebarCollapsed = ref(true) // Collapsed by default on mobile
const activeTab = ref('dashboard')

const navItems = [
  { id: 'dashboard', label: 'Dashboard', icon: 'ti ti-layout-dashboard', desc: 'Resumen de actividad y estadísticas' },
  { id: 'profile', label: 'Mi Perfil', icon: 'ti ti-user', desc: 'Información de tu cuenta y canal' },
  { id: 'stream', label: 'Transmisión', icon: 'ti ti-video', desc: 'Gestión de claves OBS y fuentes IPTV' },
  { id: 'agenda', label: 'Agenda', icon: 'ti ti-calendar-event', desc: 'Programación de eventos para tu canal' },
  { id: 'followers', label: 'Seguidores', icon: 'ti ti-users', desc: 'Comunidad de usuarios que te siguen' }
]

const currentNavLabel = computed(() => navItems.find(i => i.id === activeTab.value)?.label)
const currentNavDesc = computed(() => navItems.find(i => i.id === activeTab.value)?.desc)

const selectTab = (id) => {
  activeTab.value = id
  if (window.innerWidth < 1024) {
    isSidebarCollapsed.value = true
  }
}

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
const iptvUrlsObj = ref([])
const activeIptvUrl = ref(null)
const newIptvUrl = ref('')
const newIptvName = ref('')
const isRestreaming = ref(false)
const isPausedByObs = ref(false)
const editingIptvIndex = ref(-1)
const editingIptvData = ref({ name: '', url: '' })
let statusInterval = null

const startEditingIptv = (index) => {
  editingIptvIndex.value = index
  editingIptvData.value = { ...iptvUrlsObj.value[index] }
}

const saveEditedIptv = () => {
  if (editingIptvIndex.value > -1) {
    const oldUrl = iptvUrlsObj.value[editingIptvIndex.value].url
    iptvUrlsObj.value[editingIptvIndex.value] = { ...editingIptvData.value }
    if (activeIptvUrl.value === oldUrl) {
      activeIptvUrl.value = editingIptvData.value.url
    }
    saveIptvSettings()
    editingIptvIndex.value = -1
  }
}

const cancelEditingIptv = () => {
  editingIptvIndex.value = -1
}

// Profile refs
const profileData = ref({
  username: '',
  email: '',
  password: '',
  channelName: '',
  whatsappLink: '',
  tiktokLink: '',
  logoUrl: ''
})
const isUpdatingProfile = ref(false)

// Logo Upload refs
const logoInput = ref(null)
const selectedLogoFile = ref(null)
const isUploadingLogo = ref(false)

const handleLogoSelect = (event) => {
  const file = event.target.files[0]
  if (file) {
    selectedLogoFile.value = file
  }
}

const uploadLogo = async () => {
  if (!selectedLogoFile.value || isUploadingLogo.value) return
  isUploadingLogo.value = true
  
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    const formData = new FormData()
    formData.append('logo', selectedLogoFile.value)

    const res = await fetch('/api/v1/channels/logo', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      },
      body: formData
    })

    if (res.ok) {
      const data = await res.json()
      profileData.value.logoUrl = data.logo_url
      selectedLogoFile.value = null
      if (logoInput.value) logoInput.value.value = ''
      alert('Logo subido y optimizado a WebP exitosamente.')
    } else {
      alert('Hubo un error al subir el logo.')
    }
  } catch(e) {
    console.error(e)
    alert('Error de conexión al subir logo')
  } finally {
    isUploadingLogo.value = false
  }
}

// Agenda refs
const agendaEvents = ref([])
const isSavingAgenda = ref(false)

// Analytics refs
const analyticsSummary = ref({
  total_views: 0,
  views_last_24h: 0,
  top_countries: []
})

const fetchAnalytics = async () => {
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    const res = await fetch('/api/v1/analytics/summary', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      analyticsSummary.value = {
        total_views: data.total_views || 0,
        views_last_24h: data.views_last_24h || 0,
        top_countries: data.top_countries || []
      }
    }
  } catch (err) {
    console.error('Failed to fetch analytics', err)
  }
}

// Followers refs
const followers = ref([])

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString()
}

const addAgendaEvent = () => {
  agendaEvents.value.unshift({
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
        token: data.token
      }
      localStorage.setItem('fuchibol_user', JSON.stringify(updatedUser))
      currentUser.value = updatedUser
      profileData.value.password = ''
      alert('Perfil actualizado exitosamente.')
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
  alert('¡Copiado!')
}

const loadKey = async () => {
  if (streamKey.value) {
    showKey.value = true
    return
  }
  if (!confirm('Deseas generar una nueva clave de transmisión?')) return

  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    const res = await fetch(`/api/v1/auth/rotate-stream-key`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      streamKey.value = data.stream_key
      showKey.value = true
    }
  } catch (err) {
    console.error(err)
  }
}

const rotateKey = async () => {
  if (!confirm('¿Seguro que deseas rotar la clave? OBS se desconectará.')) return
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    const res = await fetch(`/api/v1/auth/rotate-stream-key`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      streamKey.value = data.stream_key
      showKey.value = true
      alert('Clave rotada exitosamente.')
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
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      channelId.value = data.id
      profileData.value.channelName = data.name
      profileData.value.whatsappLink = data.whatsapp_link || ''
      profileData.value.tiktokLink = data.tiktok_link || ''
      profileData.value.logoUrl = data.logo_url || ''
      activeIptvUrl.value = data.active_iptv_url
      
      // Parse IPTV URLs (handle both old string-list and new object-list formats)
      try {
        const raw = JSON.parse(data.iptv_urls || "[]")
        iptvUrlsObj.value = raw.map(item => {
          if (typeof item === 'string') return { url: item, name: 'Canal TV' }
          return item
        })
      } catch(e) {
        iptvUrlsObj.value = []
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
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      followers.value = data || []
    }
  } catch (err) {
    console.error('Failed to load followers', err)
  }
}

const saveIptvSettings = async () => {
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    const token = tokenObj ? JSON.parse(tokenObj).token : ''
    
    await fetch('/api/v1/channels/me', {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        active_iptv_url: activeIptvUrl.value,
        iptv_urls: JSON.stringify(iptvUrlsObj.value)
      })
    })
    
    if (isRestreaming.value || isPausedByObs.value) {
      await startRestream()
    }
  } catch(e) {
    console.error(e)
  }
}

const addIptvUrl = () => {
  const url = newIptvUrl.value.trim()
  const name = newIptvName.value.trim() || 'Nuevo Canal'
  if (url && !iptvUrlsObj.value.some(i => i.url === url)) {
    iptvUrlsObj.value.push({ url, name })
    newIptvUrl.value = ''
    newIptvName.value = ''
    if (!activeIptvUrl.value) {
      activeIptvUrl.value = url
    }
    saveIptvSettings()
  }
}

const removeIptvUrl = (index) => {
  const item = iptvUrlsObj.value[index]
  iptvUrlsObj.value.splice(index, 1)
  if (activeIptvUrl.value === item.url) {
    activeIptvUrl.value = iptvUrlsObj.value.length > 0 ? iptvUrlsObj.value[0].url : null
  }
  saveIptvSettings()
}

const checkRestreamStatus = async () => {
  try {
    const tokenObj = localStorage.getItem('fuchibol_user')
    if (!tokenObj) return
    const token = JSON.parse(tokenObj).token
    const res = await fetch('/api/v1/restream/status', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
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
      body: JSON.stringify({ iptv_url: activeIptvUrl.value })
    })
    if (res.ok) {
      isRestreaming.value = true
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
    }
  } catch(e) {
    console.error(e)
  } finally {
    isProcessing.value = false
  }
}

onMounted(() => {
  serverUrl.value = getRtmpServerUrl()
  checkAccess()
  fetchAnalytics()
  statusInterval = setInterval(checkRestreamStatus, 5000)
})

onBeforeUnmount(() => {
  if (statusInterval) clearInterval(statusInterval)
})
</script>

<style scoped>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

.admin-layout {
  --bg-base:    #0d0f14;
  --bg-surface: #111318;
  --bg-card:    #161a22;
  --border:     rgba(255,255,255,0.07);
  --text-primary:   #ffffff;
  --text-secondary: rgba(255,255,255,0.5);
  --text-muted:     rgba(255,255,255,0.25);
  --accent:     #00e676;
  --accent-bg:  rgba(0,230,118,0.12);
  --teal:       #1D9E75;
  --teal-bg:    rgba(29,158,117,0.15);
  --blue:       #378ADD;
  --blue-bg:    rgba(55,138,221,0.15);
  --purple:     #7F77DD;
  --purple-bg:  rgba(127,119,221,0.15);
  --red:        #f87171;
  --red-bg:     rgba(239,68,68,0.08);
  --red-border: rgba(239,68,68,0.3);
  --radius-md:  8px;
  --radius-lg:  12px;
  --radius-xl:  16px;

  background: var(--bg-base);
  color: var(--text-primary);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  height: 100vh;
  height: 100dvh;
  width: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ─── MOBILE LAYOUT (default) ─── */
.admin-layout {
  max-width: 100%;
  margin: 0;
}
.sidebar, .desktop-topbar {
  display: none;
}

/* TOP BAR */
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  background: var(--bg-surface);
  border-bottom: 0.5px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 10;
}
.logo-row { display: flex; align-items: center; gap: 8px; }
.logo-dot {
  width: 9px; height: 9px;
  border-radius: 50%;
  background: var(--accent);
}
.logo-text {
  font-size: 14px; font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.08em;
}
.topbar-right { display: flex; align-items: center; gap: 10px; }
.icon-btn {
  width: 36px; height: 36px;
  border-radius: var(--radius-md);
  background: rgba(255,255,255,0.06);
  border: 0.5px solid rgba(255,255,255,0.1);
  display: flex; align-items: center; justify-content: center;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 18px;
  transition: background 0.15s;
}
.icon-btn:hover { background: rgba(255,255,255,0.1); }
.avatar-sm {
  width: 36px; height: 36px;
  border-radius: 50%;
  background: var(--accent-bg);
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 600;
  color: var(--accent);
  cursor: pointer;
}

/* MAIN SCROLL AREA */
.main {
  flex: 1;
  overflow-y: scroll;
  padding-bottom: 80px;
  min-height: 0;
}

/* HERO */
.hero { padding: 20px 16px 8px; }
.hero-title { font-size: 22px; font-weight: 600; color: var(--text-primary); }
.hero-sub { font-size: 12px; color: var(--text-secondary); margin-top: 3px; }

.badge-live {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 4px 11px;
  border-radius: 20px;
  background: var(--accent-bg);
  color: var(--accent);
  font-size: 11px; font-weight: 600;
  margin-top: 10px;
}
.live-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--accent);
  animation: pulse 1.8s ease-in-out infinite;
}
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.3} }

/* STAT GRID */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 16px 16px 0;
}
.stat-card {
  background: var(--bg-card);
  border: 0.5px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 14px;
}
.stat-card.wide { grid-column: 1 / -1; }
.stat-icon-row {
  display: flex; align-items: center; gap: 8px;
  margin-bottom: 10px;
}
.stat-icon {
  width: 32px; height: 32px;
  border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center;
  font-size: 15px;
}
.stat-icon.teal  { background: var(--teal-bg);   color: var(--teal); }
.stat-icon.blue  { background: var(--blue-bg);   color: var(--blue); }
.stat-icon.purple{ background: var(--purple-bg); color: var(--purple); }
.stat-label {
  font-size: 10px;
  color: var(--text-muted);
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
.stat-value { font-size: 28px; font-weight: 600; color: var(--text-primary); line-height: 1.1; }
.stat-delta { font-size: 11px; color: var(--text-muted); margin-top: 3px; }

/* INFO CARDS */
.section { padding: 12px 16px 0; }
.info-card {
  background: var(--bg-card);
  border: 0.5px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 16px;
  margin-bottom: 10px;
}
.info-card-title {
  font-size: 11px; font-weight: 600;
  color: var(--text-secondary);
  display: flex; align-items: center; gap: 7px;
  margin-bottom: 14px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.info-card-title i { font-size: 14px; }

.status-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 9px 0;
  border-bottom: 0.5px solid rgba(255,255,255,0.05);
  font-size: 13px;
}
.status-row:last-child { border-bottom: none; }
.status-label { color: var(--text-secondary); }
.status-val { color: var(--text-primary); font-weight: 500; }

.empty-state {
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  padding: 20px 0; gap: 8px;
}
.empty-icon { font-size: 28px; color: rgba(255,255,255,0.1); }
.empty-text {
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
  line-height: 1.6;
}

/* PUBLIC BUTTON */
.btn-public {
  display: flex; align-items: center; justify-content: center; gap: 8px;
  margin: 4px 16px 12px;
  padding: 12px;
  border-radius: var(--radius-lg);
  background: var(--text-primary);
  color: var(--bg-base);
  font-size: 14px; font-weight: 600;
  cursor: pointer;
  border: none;
  width: calc(100% - 32px);
  transition: opacity 0.15s;
}
.btn-public:hover { opacity: 0.88; }
.btn-public i { font-size: 15px; }

/* BOTTOM NAV */
.bottom-nav {
  display: flex;
  border-top: 0.5px solid var(--border);
  background: var(--bg-surface);
  position: fixed;
  bottom: 0; left: 0; right: 0;
  margin: 0;
  z-index: 10;
}
.nav-tab {
  flex: 1;
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  padding: 10px 4px 8px;
  gap: 3px;
  font-size: 10px;
  color: var(--text-muted);
  cursor: pointer;
  border-top: 2px solid transparent;
  transition: color 0.15s, border-color 0.15s;
  text-decoration: none;
}
.nav-tab i { font-size: 20px; }
.nav-tab.active { color: var(--accent); border-top-color: var(--accent); }
.nav-tab:hover:not(.active) { color: var(--text-secondary); }

/* Buttons & Inputs */
.btn-primary {
  background: var(--accent);
  color: var(--bg-base);
  border: none;
  padding: 10px 16px;
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: opacity 0.2s;
}
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary:hover:not(:disabled) { opacity: 0.9; }

.btn-secondary {
  background: rgba(255,255,255,0.05);
  color: var(--text-primary);
  border: 0.5px solid var(--border);
  padding: 10px 16px;
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-secondary:hover { background: rgba(255,255,255,0.1); }
.btn-sm { padding: 6px 12px; font-size: 12px; }

.input-field {
  width: 100%;
  background: rgba(0,0,0,0.2);
  border: 1px solid var(--border);
  color: var(--text-primary);
  padding: 10px 12px;
  border-radius: var(--radius-md);
  font-size: 13px;
  outline: none;
}
.input-field:focus { border-color: var(--accent); }
.form-group label {
  display: block;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 6px;
  font-weight: 500;
}
.input-copy-group { display: flex; gap: 8px; }

/* Utilities */
.w-full { width: 100%; }
.mt-2 { margin-top: 8px; }
.mt-4 { margin-top: 16px; }
.mb-2 { margin-bottom: 8px; }
.mb-4 { margin-bottom: 16px; }
.flex-between { display: flex; justify-content: space-between; align-items: center; }
.danger-text { color: var(--red); }
.btn-icon { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 4px; border-radius: var(--radius-md); }
.btn-icon:hover { background: rgba(255,255,255,0.05); color: var(--text-primary); }
.btn-icon.danger:hover { color: var(--red); background: var(--red-bg); }

/* Forms */
.form-grid { display: flex; flex-direction: column; gap: 16px; }
.divider { height: 1px; background: var(--border); margin: 24px 0; }

/* Toggle Switch */
.toggle-switch { display: flex; align-items: center; gap: 10px; cursor: pointer; }
.toggle-switch input { display: none; }
.toggle-switch .slider { width: 36px; height: 18px; background: rgba(255,255,255,0.1); border-radius: 20px; position: relative; transition: 0.3s; }
.toggle-switch .slider:after { content: ''; position: absolute; width: 14px; height: 14px; background: #fff; border-radius: 50%; top: 2px; left: 2px; transition: 0.3s; }
.toggle-switch input:checked + .slider { background: var(--accent); }
.toggle-switch input:checked + .slider:after { left: 20px; }
.toggle-switch .label { font-size: 12px; font-weight: 600; color: var(--text-secondary); }
.toggle-switch input:checked ~ .label { color: var(--accent); }

/* IPTV Styles */
.iptv-item-new {
  display: flex; align-items: center; gap: 12px;
  padding: 12px;
  background: rgba(255,255,255,0.02);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  margin-bottom: 8px;
}
.iptv-item-new.active { border-color: var(--accent); background: var(--accent-bg); }
.item-details { flex: 1; cursor: pointer; overflow: hidden; }
.item-name { display: block; font-size: 13px; font-weight: 600; }
.item-url { display: block; font-size: 11px; color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

/* Logo Upload */
.logo-preview-container { display: flex; align-items: center; gap: 12px; background: rgba(0,0,0,0.1); padding: 12px; border-radius: var(--radius-md); border: 1px dashed var(--border); }
.logo-preview, .logo-placeholder { width: 50px; height: 50px; border-radius: 8px; object-fit: cover; }
.logo-placeholder { background: rgba(255,255,255,0.05); display: flex; align-items: center; justify-content: center; font-size: 10px; color: var(--text-secondary); }
.hidden-input { display: none; }
.file-name-hint { display: block; margin-top: 6px; font-size: 11px; color: var(--accent); }

/* Badges */
.status-badge { font-size: 10px; padding: 3px 8px; border-radius: 4px; font-weight: 600; }
.status-badge.running { background: var(--accent-bg); color: var(--accent); }
.status-badge.paused { background: var(--purple-bg); color: var(--purple); }
.status-badge.stopped { background: rgba(255,255,255,0.05); color: var(--text-secondary); }

/* ─── DESKTOP LAYOUT ─── */
@media (min-width: 768px) {
  .admin-layout {
    max-width: 100%;
    flex-direction: row;
    height: 100vh;
  }

  /* Show sidebar, hide bottom nav */
  .bottom-nav { display: none; }
  .main { padding-bottom: 0; }

  .sidebar {
    display: flex !important;
    flex-direction: column;
    width: 220px;
    min-width: 220px;
    background: var(--bg-surface);
    border-right: 0.5px solid var(--border);
    height: 100vh;
    position: sticky;
    top: 0;
  }
  .sidebar-logo {
    display: flex; align-items: center; gap: 10px;
    padding: 20px 20px 22px;
    border-bottom: 0.5px solid var(--border);
  }
  .sidebar-logo .logo-dot { width: 10px; height: 10px; }
  .sidebar-logo .logo-text { font-size: 15px; }
  .sidebar-nav { padding: 14px 12px; flex: 1; }
  .sidebar-nav-item {
    display: flex; align-items: center; gap: 10px;
    padding: 9px 12px;
    border-radius: var(--radius-md);
    font-size: 13.5px;
    color: var(--text-secondary);
    cursor: pointer;
    margin-bottom: 2px;
    transition: background 0.15s, color 0.15s;
    text-decoration: none;
  }
  .sidebar-nav-item i { font-size: 17px; }
  .sidebar-nav-item:hover { background: rgba(255,255,255,0.05); color: rgba(255,255,255,0.8); }
  .sidebar-nav-item.active { background: var(--accent-bg); color: var(--accent); }

  .sidebar-footer {
    border-top: 0.5px solid var(--border);
    padding: 16px 18px;
  }
  .user-row {
    display: flex; align-items: center; gap: 10px;
    margin-bottom: 12px;
  }
  .user-name { font-size: 13px; color: var(--text-primary); font-weight: 500; }
  .user-role { font-size: 11px; color: var(--accent); letter-spacing: 0.04em; }
  .btn-logout {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 12px;
    border-radius: var(--radius-md);
    border: 0.5px solid var(--red-border);
    color: var(--red);
    font-size: 13px;
    cursor: pointer;
    background: transparent;
    width: 100%;
    transition: background 0.15s;
  }
  .btn-logout:hover { background: var(--red-bg); }

  /* Desktop main area */
  .topbar { display: none; }

  .main {
    flex: 1;
    background: var(--bg-base);
    overflow-y: scroll;
    height: 100vh;
  }

  .desktop-topbar {
    display: flex !important;
    justify-content: space-between;
    align-items: flex-start;
    padding: 28px 28px 0;
    margin-bottom: 24px;
  }
  .desktop-title { font-size: 24px; font-weight: 600; color: var(--text-primary); }
  .desktop-sub { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }

  .btn-public-desktop {
    display: flex !important; align-items: center; gap: 8px;
    padding: 10px 18px;
    border-radius: var(--radius-md);
    background: var(--text-primary);
    color: var(--bg-base);
    font-size: 13px; font-weight: 600;
    cursor: pointer; border: none;
    transition: opacity 0.15s;
    white-space: nowrap;
    height: fit-content;
  }
  .btn-public-desktop:hover { opacity: 0.88; }

  /* Desktop stats: 3 columns */
  .stats-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    padding: 0 28px;
    margin-bottom: 6px;
  }
  .stat-card.wide { grid-column: auto; }

  /* Desktop bottom cards: 2 columns */
  .bottom-grid {
    display: grid !important;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
    padding: 14px 28px 28px;
  }
  .bottom-grid .info-card { margin-bottom: 0; }
  .form-grid { grid-template-columns: 1fr 1fr; }
  
  .section { padding: 0 28px; }

  /* Hide mobile elements on desktop */
  .hero { display: none; }
  .btn-public { display: none; }
}
</style>
