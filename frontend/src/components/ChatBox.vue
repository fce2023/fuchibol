<template>
  <div class="chat-section">
    <div class="chat-header">
      <span class="chat-title">CHAT EN VIVO</span>
      <div class="chat-status">
        <div class="chat-status-dot" :style="socket && socket.readyState === 1 ? 'background: #00e87a;' : 'background: #e83d00;'"></div>
        <span class="chat-status-text" :style="socket && socket.readyState === 1 ? 'color: #00e87a;' : 'color: #e83d00;'">{{ socket && socket.readyState === 1 ? 'Conectado' : 'Desconectado' }}</span>
      </div>
    </div>
    <div class="chat-messages" ref="msgContainerRef">
      <div v-for="msg in messages" :key="msg.id || Math.random()" class="chat-msg" :style="msg.type === 'system' ? 'font-style: italic; color: #606080;' : 'font-style: normal; color: #c0c0d8;'">
        <template v-if="msg.type === 'system' || msg.type === 'system error'">
          {{ msg.content }}
        </template>
        <template v-else>
          <strong :style="{ color: getUserColor(msg.username), marginRight: '4px' }">{{ msg.username }}:</strong>
          <span>{{ msg.content }}</span>
          <button v-if="canModerate" @click="deleteMessage(msg.id)" style="margin-left:8px; background:none; border:none; color:#e83d00; cursor:pointer;">[x]</button>
        </template>
      </div>
    </div>
    <div class="chat-input-row">
      <input
        v-model="newMessage"
        @keydown.enter="sendMessage"
        class="chat-input"
        type="text"
        :placeholder="currentUser ? 'Escribe un mensaje...' : 'Inicia sesión para chatear'"
        :disabled="!currentUser"
        maxlength="200"
      />
      <button class="send-btn" @click="sendMessage" :disabled="!currentUser">Enviar</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'

const props = defineProps({
  channelId: {
    type: [Number, String],
    required: true
  },
  currentUser: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['viewer-update', 'stream-reload'])

const messages = ref([
  { type: 'system', content: 'Bienvenido al chat de Fuchibol. Sé respetuoso con los demás.' }
])
const newMessage = ref('')
const msgContainerRef = ref(null)
const slowmodeRemaining = ref(0)
const isConnected = ref(false)

const getUserColor = (username) => {
  if (!username) return 'hsl(var(--secondary))'
  let hash = 0
  for (let i = 0; i < username.length; i++) {
    hash = username.charCodeAt(i) + ((hash << 5) - hash)
  }
  const hue = Math.abs(hash % 360)
  return `hsl(${hue}, 70%, 65%)`
}

const canModerate = computed(() => {
  if (!props.currentUser) return false
  return ['admin', 'mod', 'streamer'].includes(props.currentUser.role)
})

let socket = null
let slowmodeTimer = null
let pingTimer = null

const scrollToBottom = async () => {
  await nextTick()
  const container = msgContainerRef.value
  if (container) {
    container.scrollTop = container.scrollHeight
  }
}

const connectSocket = () => {
  if (socket) {
    socket.close()
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  socket = new WebSocket(`${protocol}//${window.location.host}/ws/chat?channel=${props.channelId}`)

  socket.onopen = () => {
    console.log('Connected to native WebSocket chat')
    isConnected.value = true
    // Keep connection alive
    if (pingTimer) clearInterval(pingTimer)
    pingTimer = setInterval(() => {
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'ping' }))
      }
    }, 30000)
  }

  socket.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      
      if (msg.type === 'viewers' || msg.type === 'pong' || msg.type === 'ping') {
        if (msg.viewers !== undefined) {
          emit('viewer-update', msg.viewers)
        }
        return
      }

      if (msg.type === 'stream_reload') {
        emit('stream-reload')
        return
      }
      
      // Mapeamos el payload que manda Go a lo que espera la interfaz Vue
      messages.value.push({
        id: Date.now(),
        type: 'user',
        username: msg.username || 'Usuario',
        content: msg.content
      })
      scrollToBottom()
    } catch(err) {
      console.error(err)
    }
  }

  socket.onerror = (err) => {
    console.error('WebSocket Error', err)
    // Avoid spamming the chat window on minor network drops or server restarts
  }

  socket.onclose = () => {
    isConnected.value = false
    if (pingTimer) clearInterval(pingTimer)
    messages.value.push({ type: 'system', content: 'Desconectado del chat. Reconectando en 3s...' })
    scrollToBottom()
    
    // Auto-reconnect after 3 seconds
    setTimeout(() => {
      connectSocket()
    }, 3000)
  }
}

const sendMessage = () => {
  const text = newMessage.value.trim()
  if (!text || slowmodeRemaining.value > 0) return

  if (socket && socket.readyState === WebSocket.OPEN) {
    const payload = {
      channel_id: Number(props.channelId),
      content: text,
      user_id: props.currentUser ? props.currentUser.id : 0,
      username: props.currentUser ? props.currentUser.username : "Anonimo"
    }
    socket.send(JSON.stringify(payload))
    newMessage.value = ''
  } else {
    messages.value.push({ type: 'system error', content: 'No se pudo enviar el mensaje (sin conexión).' })
    scrollToBottom()
  }
}

const deleteMessage = (msgId) => {
  // Backend support pending
  console.log("Delete message", msgId)
}

watch(() => props.channelId, () => {
  connectSocket()
})

onMounted(() => {
  connectSocket()
})

onBeforeUnmount(() => {
  if (pingTimer) clearInterval(pingTimer)
  if (socket) {
    socket.close()
  }
  if (slowmodeTimer) clearInterval(slowmodeTimer)
})
</script>

<style scoped>
/* ── Chat ── */
.chat-section {
  background: #0d0d18;
  margin: 0 12px 14px;
  border-radius: 10px;
  border: 0.5px solid #1a1a28;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.chat-header {
  padding: 10px 14px;
  border-bottom: 0.5px solid #1a1a28;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.chat-title { font-size: 12px; font-weight: 700; color: #e0e0f0; letter-spacing: 0.5px; }
.chat-status { display: flex; align-items: center; gap: 5px; }
.chat-status-dot { width: 5px; height: 5px; border-radius: 50%; }
.chat-status-text { font-size: 10px; }
.chat-messages {
  padding: 10px 14px; 
  height: 350px;
  overflow-y: auto;
}
.chat-msg {
  font-size: 12px;
  color: #606080;
  line-height: 1.6;
  font-style: italic;
  word-wrap: break-word;
}
.chat-msg + .chat-msg { margin-top: 6px; }
.chat-input-row {
  display: flex;
  gap: 8px;
  padding: 10px 12px;
  border-top: 0.5px solid #1a1a28;
}
.chat-input {
  flex: 1;
  background: #13131f;
  border: 0.5px solid #2a2a40;
  border-radius: 20px;
  padding: 8px 14px;
  font-size: 12px;
  color: #c0c0d8;
  outline: none;
  font-family: inherit;
  transition: border-color 0.2s;
}
.chat-input::placeholder { color: #40405a; }
.chat-input:focus { border-color: rgba(0,232,122,0.4); }
.chat-input:disabled { opacity: 0.5; cursor: not-allowed; }
.send-btn {
  background: #00e87a;
  border: none;
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 12px;
  font-weight: 700;
  color: #002f18;
  cursor: pointer;
  transition: background 0.2s;
  white-space: nowrap;
}
.send-btn:hover:not(:disabled) { background: #00ff88; }
.send-btn:disabled { background: #1a1a28; color: #606080; cursor: not-allowed; }

@media (min-width: 1024px) {
  .chat-section {
    flex: 1;
    margin: 12px;
    height: auto;
    min-height: 400px;
  }
  .chat-messages {
    flex: 1;
    height: 0;
  }
}
</style>

