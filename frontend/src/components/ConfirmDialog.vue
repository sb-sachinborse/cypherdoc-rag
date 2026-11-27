<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  open: Boolean,
  title: { type: String, default: 'Are you sure?' },
  message: { type: String, default: '' },
  confirmText: { type: String, default: 'Delete' },
  danger: { type: Boolean, default: true },
})
const emit = defineEmits(['confirm', 'cancel'])
const cancelBtn = ref(null)
const dialog = ref(null)
let previous = null

function onKey(e) {
  if (e.key === 'Escape') { e.preventDefault(); emit('cancel'); return }
  if (e.key === 'Tab' && dialog.value) {
    // keep keyboard focus inside the dialog
    const items = dialog.value.querySelectorAll('button')
    const first = items[0], last = items[items.length - 1]
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus() }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus() }
  }
}

watch(() => props.open, async (isOpen) => {
  if (isOpen) {
    previous = document.activeElement
    document.addEventListener('keydown', onKey)
    await nextTick()
    cancelBtn.value?.focus()
  } else {
    document.removeEventListener('keydown', onKey)
    previous?.focus?.()
  }
})
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="backdrop" @mousedown.self="emit('cancel')">
      <div ref="dialog" class="dialog" role="alertdialog" aria-modal="true" aria-labelledby="cd-title" aria-describedby="cd-msg">
        <div class="icon" :class="{ danger }" aria-hidden="true">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14M10 11v6M14 11v6" /></svg>
        </div>
        <h3 id="cd-title">{{ title }}</h3>
        <p id="cd-msg" class="muted">{{ message }}</p>
        <div class="actions">
          <button ref="cancelBtn" type="button" class="btn" @click="emit('cancel')">Cancel</button>
          <button type="button" class="btn" :class="danger ? 'danger' : 'primary'" @click="emit('confirm')">{{ confirmText }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.backdrop { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; padding: 20px; background: rgba(5, 7, 12, 0.6); backdrop-filter: blur(3px); animation: fade 0.15s ease-out; }
.dialog { width: 100%; max-width: 400px; background: var(--surface); border: 1px solid var(--border-strong); border-radius: 16px; padding: 24px; box-shadow: 0 24px 60px rgba(0, 0, 0, 0.35); animation: pop 0.18s ease-out; }
.icon { width: 44px; height: 44px; border-radius: 12px; display: grid; place-items: center; background: var(--chip); color: var(--accent-text); margin-bottom: 14px; }
.icon.danger { background: var(--err-bg); color: var(--err); }
h3 { font-size: 18px; margin-bottom: 6px; }
p { overflow-wrap: anywhere; }
.actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 22px; }
.btn.danger { background: #dc2626; border-color: #dc2626; color: #fff; }
.btn.danger:hover { filter: brightness(1.1); }
@keyframes fade { from { opacity: 0; } }
@keyframes pop { from { opacity: 0; transform: translateY(8px) scale(0.97); } }
@media (prefers-reduced-motion: reduce) { .backdrop, .dialog { animation: none; } }
</style>
