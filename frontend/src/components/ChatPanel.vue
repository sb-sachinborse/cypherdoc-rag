<script setup>
import { nextTick, onBeforeUnmount, ref } from 'vue'
import { api } from '../api'
import RichBlocks from './RichBlocks.vue'
import MarkdownText from './MarkdownText.vue'

const props = defineProps({ collection: String, hasDocs: Boolean, ready: Boolean })
const messages = ref([])
const input = ref('')
const busy = ref(false)
const scroller = ref(null)
let ctrl = null

// The model is told not to cite, but strip stray markers like [1] just in case.
const clean = (t) => t.replace(/\s*\[\d+\]/g, '')

const starters = ['Summarize my documents', 'What are the key figures?', 'List the main topics']

function toBottom() { nextTick(() => { if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight }) }

async function send(text) {
  const question = (text ?? input.value).trim()
  if (!question || busy.value) return
  const history = messages.value.filter((m) => m.content && !m.error).map((m) => ({ role: m.role, content: m.content }))
  messages.value.push({ role: 'user', content: question })
  messages.value.push({ role: 'assistant', content: '', sources: [], blocks: null, streaming: true, error: '', status: '', notice: '' })
  const reply = messages.value[messages.value.length - 1]
  input.value = ''
  busy.value = true
  ctrl = new AbortController()
  toBottom()
  try {
    await api.chatStream({ question, collection: props.collection, history }, {
      status: (s) => { reply.status = s.text; toBottom() },
      notice: (n) => { reply.notice = n.text; reply.status = '' },
      token: (t) => { reply.content += t.text; toBottom() },
      blocks: (b) => { reply.blocks = b; reply.status = ''; toBottom() },
      error: (e) => { reply.error = e.message },
    }, ctrl.signal)
  } catch (e) {
    if (e.name !== 'AbortError') reply.error = e.message
  } finally {
    reply.streaming = false
    reply.status = ''
    busy.value = false
    ctrl = null
    toBottom()
  }
}
function stop() { ctrl?.abort() }
onBeforeUnmount(stop)
</script>

<template>
  <section class="chat">
    <header class="head"><strong>AI Assistant</strong><span class="ok">● RAG {{ ready ? 'active' : 'starting…' }}</span></header>
    <div ref="scroller" class="scroll" aria-live="polite">
      <div v-if="!messages.length" class="hello">
        <p>Hello! Ask a question about your {{ collection ? 'collection “' + collection + '”' : 'documents' }}.</p>
        <p v-if="!hasDocs" class="muted">Upload a document on the left first.</p>
        <div v-else class="starters"><button v-for="s in starters" :key="s" type="button" class="suggest" :disabled="!ready" @click="send(s)">{{ s }}</button></div>
      </div>
      <div v-for="(m, i) in messages" :key="i" class="msg" :class="m.role">
        <div v-if="m.role === 'user'" class="bubble">{{ m.content }}</div>
        <div v-else class="ai">
          <div v-if="m.content" class="text"><MarkdownText :text="clean(m.content)" /><span v-if="m.streaming" class="caret"></span></div>
          <p v-else-if="m.streaming && !m.error" class="muted">Thinking…</p>
          <RichBlocks v-if="m.blocks" :blocks="m.blocks" />
          <p v-if="m.status" class="muted working"><span class="spin" aria-hidden="true"></span>{{ m.status }}</p>
          <p v-if="m.notice" class="muted">{{ m.notice }}</p>
          <p v-if="m.error" class="err" role="alert">{{ m.error }}</p>
        </div>
      </div>
    </div>
    <form class="composer" @submit.prevent="send()">
      <input v-model="input" class="field" maxlength="2000" placeholder="Ask a question about your documents…" aria-label="Ask a question" :disabled="busy" />
      <button v-if="busy" type="button" class="btn" @click="stop">Stop</button>
      <button v-else type="submit" class="btn primary" :disabled="!input.trim() || !ready" aria-label="Send">Send</button>
    </form>
    <small class="note muted">AI can make mistakes. Verify important information.</small>
  </section>
</template>

<style scoped>
.chat { display: flex; flex-direction: column; flex: 1; height: 100%; min-height: 0; }
.head { height: 60px; flex: none; display: flex; align-items: center; gap: 12px; padding: 0 24px; border-bottom: 1px solid var(--border); }
.ok { color: var(--ok); font-size: 12px; }
.scroll { flex: 1; overflow: auto; padding: 24px 28px; display: flex; flex-direction: column; gap: 18px; }
.hello { color: var(--text-2); display: flex; flex-direction: column; gap: 12px; }
.starters { display: flex; gap: 8px; flex-wrap: wrap; }
.suggest { padding: 7px 12px; border-radius: 99px; border: 1px solid var(--border-strong); background: var(--surface); font-size: 12px; }
.suggest:disabled { opacity: 0.5; cursor: not-allowed; }
.msg.user { align-self: flex-end; max-width: 560px; }
.bubble { background: var(--accent); color: #fff; padding: 11px 16px; border-radius: 14px 14px 4px 14px; white-space: pre-wrap; overflow-wrap: anywhere; }
.ai { max-width: 780px; display: flex; flex-direction: column; gap: 12px; }
.text { overflow-wrap: anywhere; color: var(--text-2); }
.err { color: var(--err); background: var(--err-bg); padding: 10px 12px; border-radius: 8px; }
.working { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.spin { width: 12px; height: 12px; border-radius: 50%; border: 2px solid var(--border-strong); border-top-color: var(--accent-text); animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.caret { display: inline-block; width: 2px; height: 14px; background: var(--accent-text); margin-left: 2px; vertical-align: -2px; animation: blink 1s steps(1) infinite; }
@keyframes blink { 50% { opacity: 0; } }
.composer { display: flex; gap: 8px; margin: 0 28px; padding: 6px 6px 6px 14px; border: 1px solid var(--border-strong); border-radius: 12px; background: var(--surface); }
.field { flex: 1; border: 0; background: transparent; outline: 0; min-width: 0; }
.note { text-align: center; padding: 8px 0 14px; font-size: 11px; }
@media (max-width: 1100px) { .head { display: none; } }
@media (max-width: 640px) {
  .scroll { padding: 16px 14px; gap: 16px; }
  .msg.user { max-width: 88%; }
  .composer { margin: 0 12px; }
  .field { font-size: 16px; }
  .note { padding: 6px 12px 8px; }
  .text { line-height: 1.6; }
}
</style>
