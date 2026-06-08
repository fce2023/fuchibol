<template>
  <div class="chat-wrapper">
    <div class="chat-header">
      <h2>Chat en Vivo</h2>
    </div>
    
    <div ref="msgContainerRef" class="chat-messages">
      <div v-for="(msg, index) in messages" :key="index" :class="['chat-msg', msg.type]">
        <span v-if="msg.username" class="sender">{{ msg.username }}:</span>
        <span class="msg-content">{{ msg.content }}</span>
      </div>
    </div>
    
    <div class="chat-input-area">
      <div v-if="slowmodeRemaining > 0" class="slowmode-warning">
        Modo lento activo. Espera {{ slowmodeRemaining }}s para escribir de nuevo.
      </div>
      <form @submit.prevent="sendMessage" class="input-wrapper">
        <input 
          v-model="newMessage" 
          type="text" 
          placeholder="Envía un mensaje..." 
          class="input-field chat-input" 
          :disabled="slowmodeRemaining > 0"
        />
        <button 
          type="submit" 
          class="btn btn-primary send-btn"
          :disabled="!newMessage.trim() || slowmodeRemaining > 0"
        >
          Enviar
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { io } from 'socket.io-client'

const props = defineProps({
  channelId: {
    type: [Number, String],
    required: true
  }
})

const emit = defineEmits(['viewer-update'])

const messages = ref([
  { type: 'system', content: 'Bienvenido al chat de Fuchibol. Sé respetuoso con los demás.' }
])
const newMessage = ref('')
const msgContainerRef = ref(null)
const slowmodeRemaining = ref(0)

let socket = null
let slowmodeTimer = null

const scrollToBottom = async () => {
  await nextTick()
  const container = msgContainerRef.value
  if (container) {
    container.scrollTop = container.scrollHeight
  }
}

const startSlowmodeCountdown = (seconds) => {
  if (slowmodeTimer) clearInterval(slowmodeTimer)
  slowmodeRemaining.value = seconds
  slowmodeTimer = setInterval(() => {
    slowmodeRemaining.value--
    if (slowmodeRemaining.value <= 0) {
      clearInterval(slowmodeTimer)
      slowmodeTimer = null
    }
  }, 1000)
}

const connectSocket = () => {
  if (socket) {
    socket.disconnect()
  }

  const tokenObj = localStorage.getItem('fuchibol_user')
  const token = tokenObj ? JSON.parse(tokenObj).token : ''

  socket = io(window.location.origin, {
    auth: {
      token
    }
  })

  socket.on('connect', () => {
    console.log('Connected to socket.io chat')
    socket.emit('join_channel', { channel_id: props.channelId })
  })

  socket.on('message', (msg) => {
    messages.value.push({
      type: 'user',
      username: msg.username,
      content: msg.content
    })
    scrollToBottom()
  })

  socket.on('viewer_update', (data) => {
    emit('viewer-update', data.viewers)
  })

  socket.on('error', (err) => {
    messages.value.push({
      type: 'system error',
      content: err.message
    })
    scrollToBottom()

    if (err.remaining) {
      startSlowmodeCountdown(err.remaining)
    }
  })

  socket.on('disconnect', () => {
    messages.value.push({ type: 'system', content: 'Desconectado del chat.' })
    scrollToBottom()
  })
}

const sendMessage = () => {
  const text = newMessage.value.trim()
  if (!text || slowmodeRemaining.value > 0) return

  if (socket && socket.connected) {
    socket.emit('send_message', { content: text })
    newMessage.value = ''
  } else {
    messages.value.push({ type: 'system error', content: 'No se pudo enviar el mensaje (sin conexión).' })
    scrollToBottom()
  }
}

watch(() => props.channelId, () => {
  connectSocket()
})

onMounted(() => {
  connectSocket()
})

onBeforeUnmount(() => {
  if (socket) {
    socket.emit('leave_channel')
    socket.disconnect()
  }
  if (slowmodeTimer) clearInterval(slowmodeTimer)
})
</script>

<style scoped>
.chat-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-header {
  padding: 16px 20px;
  border-bottom: 1px solid hsla(var(--text-primary), 0.05);
}

.chat-header h2 {
  font-size: 15px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: hsl(var(--text-secondary));
}

.chat-messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.chat-msg {
  font-size: 14px;
  line-height: 1.4;
  word-break: break-word;
}

.chat-msg.system {
  color: hsl(var(--text-muted));
  font-style: italic;
}

.chat-msg.error {
  color: hsl(var(--danger));
  font-weight: 500;
}

.sender {
  font-weight: 700;
  color: hsl(var(--secondary));
  margin-right: 6px;
}

.chat-input-area {
  padding: 20px;
  border-top: 1px solid hsla(var(--text-primary), 0.05);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.slowmode-warning {
  font-size: 12px;
  color: hsl(var(--warning));
  text-align: center;
  font-weight: 500;
}

.input-wrapper {
  display: flex;
  gap: 10px;
}

.chat-input {
  flex: 1;
  font-size: 14px;
  padding: 10px 14px;
}

.send-btn {
  padding: 10px 16px;
}
</style>
