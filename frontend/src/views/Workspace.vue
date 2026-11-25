<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { auth } from '../store'
import Logo from '../components/Logo.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import DocumentPanel from '../components/DocumentPanel.vue'
import ChatPanel from '../components/ChatPanel.vue'

const router = useRouter()
const docs = ref([])
const collection = ref('')
const status = ref({ ready: false, chat_model: false, embed_model: false })
const notice = ref('')
let timer = null

const collections = computed(() => [...new Set(docs.value.map((d) => d.collection))].sort())
const visible = computed(() => (collection.value ? docs.value.filter((d) => d.collection === collection.value) : docs.value))

async function refresh() {
  try { docs.value = await api.listDocs() } catch (e) { if (e.status === 401) { auth.user = null; router.push({ name: 'login' }) } }
  try { status.value = await api.status() } catch { /* shown as starting */ }
}

async function upload(file, target) {
  notice.value = ''
  try { await api.upload(file, target) } catch (e) { notice.value = `${file.name}: ${e.message}` }
  await refresh()
}

async function remove(d) {
  if (!window.confirm(`Delete “${d.name}”?`)) return
  try { await api.deleteDoc(d.id) } catch (e) { notice.value = e.message }
  await refresh()
}

async function logout() { await auth.logout(); router.push({ name: 'home' }) }

onMounted(() => { refresh(); timer = setInterval(refresh, 4000) })
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <div class="shell">
    <aside class="side">
      <Logo />
      <nav aria-label="Collections">
        <div class="cap">COLLECTIONS</div>
        <button type="button" class="nav" :class="{ on: collection === '' }" @click="collection = ''">All documents</button>
        <button v-for="c in collections" :key="c" type="button" class="nav" :class="{ on: collection === c }" @click="collection = c">{{ c }}</button>
      </nav>
      <div class="foot">
        <div class="status" :class="{ ok: status.ready }">{{ status.ready ? 'Models ready' : 'Downloading AI models…' }}</div>
        <div class="who"><span :title="auth.user?.email">{{ auth.user?.name }}</span><ThemeToggle /></div>
        <button type="button" class="btn" @click="logout">Log out</button>
      </div>
    </aside>
    <div class="docs">
      <p v-if="notice" class="notice" role="alert">{{ notice }}</p>
      <DocumentPanel :docs="visible" :collection="collection" @upload="upload" @delete="remove" />
    </div>
    <main class="main"><ChatPanel :collection="collection" :has-docs="docs.some((d) => d.status === 'indexed')" :ready="status.ready" /></main>
  </div>
</template>

<style scoped>
.shell { display: grid; grid-template-columns: 220px 320px 1fr; height: 100%; }
.side { border-right: 1px solid var(--border); padding: 20px 14px; display: flex; flex-direction: column; gap: 18px; min-height: 0; overflow: auto; }
.cap { font-size: 11px; letter-spacing: 0.06em; color: var(--muted); padding: 6px 12px; }
.nav { display: block; width: 100%; text-align: left; border: 0; background: transparent; padding: 9px 12px; border-radius: 8px; color: var(--text-2); }
.nav:hover { background: var(--chip); } .nav.on { background: var(--accent); color: #fff; }
.foot { margin-top: auto; display: flex; flex-direction: column; gap: 10px; }
.status { font-size: 12px; color: var(--warn); padding: 0 12px; } .status.ok { color: var(--ok); }
.who { display: flex; justify-content: space-between; align-items: center; padding: 0 4px 0 12px; gap: 8px; } .who span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.docs { border-right: 1px solid var(--border); display: flex; flex-direction: column; min-height: 0; }
.main { min-width: 0; min-height: 0; }
.notice { margin: 12px 16px 0; background: var(--err-bg); color: var(--err); padding: 10px 12px; border-radius: 8px; font-size: 13px; }
@media (max-width: 1100px) { .shell { grid-template-columns: 1fr; height: auto; } .main { height: 80vh; } .side { border-right: 0; } }
</style>
