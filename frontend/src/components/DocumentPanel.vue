<script setup>
import { computed, ref } from 'vue'
const props = defineProps({ docs: Array, collection: String })
const emit = defineEmits(['upload', 'delete'])
const input = ref(null)
const target = ref('')
const dragging = ref(false)
const effective = computed(() => target.value.trim() || (props.collection !== '' ? props.collection : 'General'))

function pick(files) { for (const f of files) emit('upload', f, effective.value) }
function onDrop(e) { dragging.value = false; pick(e.dataTransfer.files) }
function onChange(e) { pick(e.target.files); e.target.value = '' }
function size(n) { return n > 1048576 ? (n / 1048576).toFixed(1) + ' MB' : Math.max(1, Math.round(n / 1024)) + ' KB' }
const label = { indexed: 'Indexed', processing: 'Processing', failed: 'Failed' }
const tone = { indexed: 'ok', processing: 'warn', failed: 'err' }
</script>

<template>
  <section class="panel">
    <h2>Knowledge</h2>
    <button type="button" class="drop" :class="{ on: dragging }" @click="input.click()" @dragover.prevent="dragging = true" @dragleave="dragging = false" @drop.prevent="onDrop">
      Click to upload or drag and drop<br /><small class="muted">PDF, TXT, MD or DOCX (max 10 MB)</small>
    </button>
    <input ref="input" class="sr-only" type="file" multiple accept=".pdf,.txt,.md,.docx" tabindex="-1" aria-label="Upload documents" @change="onChange" />
    <label class="coll"><span class="muted">Add to collection</span>
      <input v-model="target" class="input small" maxlength="60" :placeholder="effective" />
    </label>
    <ul class="list">
      <li v-for="d in docs" :key="d.id" class="item">
        <div class="meta">
          <strong :title="d.name">{{ d.name }}</strong>
          <small class="muted">{{ d.collection }} · {{ size(d.size) }}</small>
          <small v-if="d.error" class="errtext">{{ d.error }}</small>
        </div>
        <span class="chip" :class="tone[d.status]">{{ label[d.status] }}</span>
        <button type="button" class="icon-btn" :aria-label="'Delete ' + d.name" @click="emit('delete', d)">✕</button>
      </li>
      <li v-if="!docs.length" class="muted empty">No documents yet. Upload one to get started.</li>
    </ul>
  </section>
</template>

<style scoped>
.panel { padding: 20px 16px; display: flex; flex-direction: column; gap: 12px; min-height: 0; height: 100%; }
h2 { font-size: 15px; }
.drop { border: 1.5px dashed var(--border-strong); background: transparent; border-radius: 12px; padding: 22px 12px; color: var(--text-2); line-height: 1.6; }
.drop:hover, .drop.on { border-color: var(--accent); background: var(--chip); }
.coll { display: flex; flex-direction: column; gap: 6px; font-size: 12px; } .small { height: 36px; }
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; overflow: auto; }
.item { display: flex; align-items: center; gap: 8px; background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 10px 12px; }
.meta { flex: 1; min-width: 0; display: flex; flex-direction: column; } .meta strong { font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.errtext { color: var(--err); } .empty { padding: 12px 4px; }
</style>
