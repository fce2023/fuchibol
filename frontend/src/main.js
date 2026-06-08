import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

const LoginView = () => import('./views/LoginView.vue')
const ChannelView = () => import('./views/ChannelView.vue')

const routes = [
  { path: '/', name: 'login', component: LoginView },
  { path: '/:username', name: 'channel', component: ChannelView }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
