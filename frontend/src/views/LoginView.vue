<template>
  <div class="login-wrapper">
    <div class="glass-panel login-card animate-slide-in">
      <div class="brand-header">
        <div class="logo">⚽ FUCHIBOL</div>
        <p class="subtitle">Plataforma de Streaming en Vivo</p>
      </div>

      <div class="tabs">
        <button 
          :class="['tab-btn', { active: isLogin }]" 
          @click="isLogin = true"
        >
          Iniciar Sesión
        </button>
        <button 
          :class="['tab-btn', { active: !isLogin }]" 
          @click="isLogin = false"
        >
          Registrarse
        </button>
      </div>

      <div v-if="successMsg" class="success-banner">
        {{ successMsg }}
      </div>

      <div v-if="errorMsg" class="error-banner">
        {{ errorMsg }}
      </div>

      <form @submit.prevent="handleSubmit">
        <div v-if="!isLogin" class="input-group">
          <label class="input-label">Usuario</label>
          <input 
            v-model="form.username" 
            type="text" 
            class="input-field" 
            placeholder="ej. streamer_pro" 
            required 
          />
        </div>

        <div class="input-group">
          <label class="input-label">Correo Electrónico o Usuario</label>
          <input 
            v-model="form.email" 
            type="text" 
            class="input-field" 
            placeholder="ej. usuario@fuchibol.com" 
            required 
          />
        </div>

        <div class="input-group">
          <label class="input-label">Contraseña</label>
          <input 
            v-model="form.password" 
            type="password" 
            class="input-field" 
            placeholder="••••••••" 
            required 
          />
        </div>

        <button type="submit" class="btn btn-primary w-full submit-btn glow-active" :disabled="loading">
          {{ loading ? 'Procesando...' : (isLogin ? 'Entrar' : 'Crear Cuenta') }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const isLogin = ref(true)
const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const form = reactive({
  username: '',
  email: '',
  password: ''
})

watch(isLogin, () => {
  errorMsg.value = ''
  successMsg.value = ''
})

const handleSubmit = async () => {
  errorMsg.value = ''
  successMsg.value = ''
  loading.value = true
  
  try {
    if (isLogin.value) {
      // API Login using OAuth2 form-encoding
      const bodyParams = new URLSearchParams()
      bodyParams.append('username', form.email) // Email input serves as username/email
      bodyParams.append('password', form.password)
      
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: bodyParams
      })
      
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.detail || 'Credenciales incorrectas.')
      }
      
      const data = await res.json()
      
      // Fetch current user details
      const meRes = await fetch('/api/v1/auth/me', {
        headers: {
          'Authorization': `Bearer ${data.access_token}`
        }
      })
      
      let username = form.email.split('@')[0]
      let id = 0
      let role = 'user'
      if (meRes.ok) {
        const meData = await meRes.json()
        username = meData.username
        role = meData.role
        id = meData.id
      }
      
      localStorage.setItem('fuchibol_user', JSON.stringify({
        id: id,
        username: username,
        role: role,
        token: data.access_token
      }))
      
      // Redirection logic based on role
      if (role === 'admin' || role === 'streamer') {
        router.push('/admin')
      } else {
        router.push('/')
      }
      
    } else {
      // API Register
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          email: form.email,
          username: form.username,
          password: form.password
        })
      })
      
      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.detail || 'Error al registrar usuario.')
      }
      
      successMsg.value = 'Registro exitoso. ¡Inicia sesión ahora!'
      isLogin.value = true
      form.password = ''
    }
  } catch (err) {
    errorMsg.value = err.message
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at top right, hsla(142, 80%, 50%, 0.08), transparent 40%),
              radial-gradient(circle at bottom left, hsla(190, 90%, 50%, 0.08), transparent 40%),
              hsl(var(--bg-primary));
  padding: 20px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  padding: 40px;
}

.brand-header {
  text-align: center;
  margin-bottom: 30px;
}

.logo {
  font-size: 28px;
  font-weight: 800;
  letter-spacing: -0.05em;
  background: linear-gradient(135deg, hsl(var(--primary)), hsl(var(--secondary)));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 6px;
}

.subtitle {
  color: hsl(var(--text-secondary));
  font-size: 14px;
}

.tabs {
  display: flex;
  border-bottom: 2px solid hsl(var(--border-color));
  margin-bottom: 25px;
}

.tab-btn {
  flex: 1;
  background: none;
  border: none;
  color: hsl(var(--text-muted));
  font-family: inherit;
  font-size: 15px;
  font-weight: 600;
  padding: 12px;
  cursor: pointer;
  transition: var(--transition-fast);
  position: relative;
}

.tab-btn.active {
  color: hsl(var(--primary));
}

.tab-btn.active::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  right: 0;
  height: 2px;
  background: hsl(var(--primary));
}

.success-banner {
  background: hsla(var(--success), 0.15);
  border: 1px solid hsl(var(--success));
  color: hsl(var(--success));
  padding: 10px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin-bottom: 20px;
  text-align: center;
}

.error-banner {
  background: hsla(var(--danger), 0.15);
  border: 1px solid hsl(var(--danger));
  color: #ff8080;
  padding: 10px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin-bottom: 20px;
  text-align: center;
}

.w-full {
  width: 100%;
}

.submit-btn {
  margin-top: 10px;
  padding: 12px;
}
</style>
