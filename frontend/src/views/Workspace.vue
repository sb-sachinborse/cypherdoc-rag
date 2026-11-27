<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { auth } from '../store'
import Logo from '../components/Logo.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import DocumentPanel from '../components/DocumentPanel.vue'
import ChatPanel from '../components/ChatPanel.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const router = useRouter()
const docs = ref([])
const collection = ref('')
const status = ref({ ready: false, chat_model: false, embed_model: false })
const notice = ref('')
const toDelete = ref(null)
const tab = ref('chat') // phones show one pane at a time
let timer = null

const collections = computed(() => [...new Set(docs.value.map((d) => d.collection))].sort())
const processing = computed(() => docs.value.filter((d) => d.status === 'processing').length)
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

function askDelete(d) { toDelete.value = d }
async function confirmDelete() {
  const d = toDelete.value
  toDelete.value = null
  if (!d) return
  try { await api.deleteDoc(d.id) } catch (e) { notice.value = e.message }
  await refresh()
}

async function logout() { await auth.logout(); router.push({ name: 'home' }) }

onMounted(() => { refresh(); timer = setInterval(refresh, 4000) })
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <div class="shell">
    <header class="mbar">
      <Logo />
      <div class="mright">
        <span class="mstatus" :class="{ ok: status.ready }">● {{ status.ready ? 'Ready' : 'Loading models…' }}</span>
        <ThemeToggle />
        <button type="button" class="icon-btn" aria-label="Log out" @click="logout">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" /></svg>
        </button>
      </div>
    </header>
    <nav class="mchips" aria-label="Collections">
      <button type="button" class="pill" :class="{ on: collection === '' }" @click="collection = ''">All</button>
      <button v-for="c in collections" :key="c" type="button" class="pill" :class="{ on: collection === c }" @click="collection = c">{{ c }}</button>
    </nav>

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

    <div class="docs" :class="{ off: tab !== 'docs' }">
      <p v-if="notice" class="notice" role="alert">{{ notice }}</p>
      <DocumentPanel :docs="visible" :collection="collection" @upload="upload" @delete="askDelete" />
    </div>
    <main class="main" :class="{ off: tab !== 'chat' }">
      <ChatPanel :collection="collection" :has-docs="docs.some((d) => d.status === 'indexed')" :ready="status.ready" />
    </main>

    <nav class="tabs" aria-label="Sections">
      <button type="button" :class="{ on: tab === 'chat' }" :aria-current="tab === 'chat' ? 'page' : undefined" @click="tab = 'chat'">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" /></svg>
        Chat
      </button>
      <button type="button" :class="{ on: tab === 'docs' }" :aria-current="tab === 'docs' ? 'page' : undefined" @click="tab = 'docs'">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z" /><path d="M14 3v5h5" /></svg>
        Documents
        <span v-if="processing" class="badge" :aria-label="processing + ' processing'">{{ processing }}</span>
      </button>
    </nav>

    <ConfirmDialog
      :open="!!toDelete"
      title="Delete document?"
      :message="toDelete ? `“${toDelete.name}” and its indexed content will be permanently removed.` : ''"
      confirm-text="Delete"
      @confirm="confirmDelete"
      @cancel="toDelete = null"
    />
  </div>
</template>

<style scoped>
.shell { display: grid; grid-template-columns: 220px 320px 1fr; height: 100%; }
.mbar, .mchips, .tabs { display: none; }
.side { border-right: 1px solid var(--border); padding: 20px 14px; display: flex; flex-direction: column; gap: 18px; min-height: 0; overflow: auto; }
.cap { font-size: 11px; letter-spacing: 0.06em; color: var(--muted); padding: 6px 12px; }
.nav { display: block; width: 100%; text-align: left; border: 0; background: transparent; padding: 9px 12px; border-radius: 8px; color: var(--text-2); }
.nav:hover { background: var(--chip); } .nav.on { background: var(--accent); color: #fff; }
.foot { margin-top: auto; display: flex; flex-direction: column; gap: 10px; }
.status { font-size: 12px; color: var(--warn); padding: 0 12px; } .status.ok { color: var(--ok); }
.who { display: flex; justify-content: space-between; align-items: center; padding: 0 4px 0 12px; gap: 8px; } .who span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.docs { border-right: 1px solid var(--border); display: flex; flex-direction: column; min-height: 0; min-width: 0; }
.main { min-width: 0; min-height: 0; display: flex; flex-direction: column; }
.notice { margin: 12px 16px 0; background: var(--err-bg); color: var(--err); padding: 10px 12px; border-radius: 8px; font-size: 13px; overflow-wrap: anywhere; }

/* Phones and small tablets: one pane at a time, bottom tab bar. */
@media (max-width: 1100px) {
  .shell { display: flex; flex-direction: column; height: 100vh; height: 100dvh; }
  .side { display: none; }
  .mbar { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 10px 14px; padding-top: max(10px, env(safe-area-inset-top)); border-bottom: 1px solid var(--border); flex: none; }
  .mright { display: flex; align-items: center; gap: 8px; min-width: 0; }
  .mstatus { font-size: 11px; color: var(--warn); white-space: nowrap; } .mstatus.ok { color: var(--ok); }
  .mchips { gap: 8px; overflow-x: auto; padding: 10px 14px; flex: none; scrollbar-width: none; border-bottom: 1px solid var(--border); }
  .mchips::-webkit-scrollbar { display: none; }
  .pill { flex: none; padding: 7px 14px; border-radius: 99px; border: 1px solid var(--border-strong); background: var(--surface); color: var(--text-2); font-size: 13px; white-space: nowrap; }
  .pill.on { background: var(--accent); border-color: var(--accent); color: #fff; }
  .docs, .main { flex: 1; border: 0; }
  .off { display: none; }
  .tabs { display: grid; grid-template-columns: 1fr 1fr; flex: none; border-top: 1px solid var(--border); background: var(--surface); padding-bottom: env(safe-area-inset-bottom); }
  .tabs button { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 9px 0 8px; border: 0; background: transparent; color: var(--muted); font-size: 11px; position: relative; }
  .tabs button.on { color: var(--accent-text); }
  .badge { position: absolute; top: 4px; left: calc(50% + 10px); min-width: 16px; height: 16px; padding: 0 4px; border-radius: 99px; background: var(--warn-bg); color: var(--warn); font-size: 10px; display: grid; place-items: center; }
}
</style>
