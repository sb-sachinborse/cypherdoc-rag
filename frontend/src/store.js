import { reactive } from 'vue'
import { api } from './api'

export const auth = reactive({
  user: null,
  loaded: false,
  async load() {
    try { this.user = await api.me() } catch { this.user = null }
    this.loaded = true
  },
  async login(email, password) { this.user = await api.login(email, password) },
  async register(name, email, password) { this.user = await api.register(name, email, password) },
  async logout() {
    try { await api.logout() } catch { /* session is cleared client-side either way */ }
    this.user = null
  },
})

function initialTheme() {
  try {
    const saved = localStorage.getItem('cd-theme')
    if (saved === 'light' || saved === 'dark') return saved
  } catch { /* storage unavailable */ }
  return window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

export const theme = reactive({
  mode: initialTheme(),
  apply() { document.documentElement.dataset.theme = this.mode },
  toggle() {
    this.mode = this.mode === 'dark' ? 'light' : 'dark'
    try { localStorage.setItem('cd-theme', this.mode) } catch { /* ignore */ }
    this.apply()
  },
})
