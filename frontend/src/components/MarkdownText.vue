<script setup>
import { computed } from 'vue'
import { parseMarkdown } from '../markdown'
import InlineText from './InlineText.vue'
const props = defineProps({ text: { type: String, default: '' } })
const blocks = computed(() => parseMarkdown(props.text))
</script>

<template>
  <div class="md">
    <template v-for="(b, i) in blocks" :key="i">
      <p v-if="b.type === 'h'" class="h" :class="'h' + b.level"><InlineText :text="b.text" /></p>
      <div v-else-if="b.type === 'table'" class="card tablewrap">
        <table>
          <thead><tr><th v-for="(c, j) in b.head" :key="j"><InlineText :text="c" /></th></tr></thead>
          <tbody><tr v-for="(r, k) in b.rows" :key="k"><td v-for="(c, j) in r" :key="j"><InlineText :text="c" /></td></tr></tbody>
        </table>
      </div>
      <ul v-else-if="b.type === 'list'" class="list">
        <li v-for="(it, k) in b.items" :key="k" :style="{ marginLeft: it.depth * 18 + 'px' }">
          <span class="mark" aria-hidden="true">{{ it.ordered ? it.n + '.' : '•' }}</span>
          <span><InlineText :text="it.text" /></span>
        </li>
      </ul>
      <p v-else class="para"><InlineText :text="b.text" /></p>
    </template>
  </div>
</template>

<style scoped>
.md { display: flex; flex-direction: column; gap: 10px; line-height: 1.65; }
.para { white-space: pre-line; overflow-wrap: anywhere; margin: 0; }
.h { font-weight: 600; color: var(--text); margin: 6px 0 0; } .h1 { font-size: 18px; } .h2 { font-size: 16px; }
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
.list li { display: flex; gap: 8px; overflow-wrap: anywhere; } .mark { color: var(--muted); min-width: 14px; }
.tablewrap { overflow-x: auto; } table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; color: var(--muted); font-weight: 500; white-space: nowrap; } th, td { padding: 8px 12px; } tbody tr { border-top: 1px solid var(--border); }
code { background: var(--chip); color: var(--accent-text); padding: 1px 5px; border-radius: 5px; font-size: 0.92em; }
</style>
