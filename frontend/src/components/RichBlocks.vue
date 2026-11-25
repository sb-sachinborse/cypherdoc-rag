<script setup>
import { computed } from 'vue'
const props = defineProps({ blocks: { type: Object, required: true } })
const colors = ['var(--bar1)', 'var(--bar2)', 'var(--bar3)']
const W = 600, H = 170, PAD = 24
const chart = computed(() => {
  const c = props.blocks.chart
  if (!c) return null
  const max = Math.max(1, ...c.series.flatMap((s) => s.values))
  const groupW = (W - PAD * 2) / c.labels.length
  const barW = Math.min(22, (groupW - 8) / c.series.length)
  const bars = []
  c.labels.forEach((label, i) => {
    c.series.forEach((s, j) => {
      const h = (s.values[i] / max) * (H - 30)
      bars.push({
        key: i + '-' + j,
        x: PAD + i * groupW + (groupW - barW * c.series.length) / 2 + j * barW,
        y: H - 20 - h, w: barW - 2, h, fill: colors[j], title: `${s.name} · ${label}: ${s.values[i]}`,
      })
    })
  })
  const labels = c.labels.map((l, i) => ({ l, x: PAD + i * groupW + groupW / 2 }))
  return { bars, labels, c }
})
</script>

<template>
  <div class="blocks">
    <div v-if="blocks.stats?.length" class="stats">
      <div v-for="s in blocks.stats" :key="s.label" class="stat">
        <span class="muted">{{ s.label }}</span><strong>{{ s.value }}</strong><em v-if="s.delta">{{ s.delta }}</em>
      </div>
    </div>
    <div v-if="chart" class="card chart">
      <div class="head"><span>{{ chart.c.title || 'Chart' }}</span>
        <span class="legend muted"><span v-for="(s, j) in chart.c.series" :key="s.name"><i :style="{ background: colors[j] }"></i>{{ s.name }}</span></span>
      </div>
      <svg :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="chart.c.title || 'Chart'">
        <line :x1="PAD" :x2="W - PAD" :y1="H - 20" :y2="H - 20" class="axis" />
        <rect v-for="b in chart.bars" :key="b.key" :x="b.x" :y="b.y" :width="b.w" :height="b.h" :fill="b.fill" rx="3"><title>{{ b.title }}</title></rect>
        <text v-for="l in chart.labels" :key="l.l + l.x" :x="l.x" :y="H - 4" text-anchor="middle" class="tick">{{ l.l }}</text>
      </svg>
    </div>
    <div v-if="blocks.table" class="card tablewrap">
      <table>
        <thead><tr><th v-for="c in blocks.table.columns" :key="c">{{ c }}</th></tr></thead>
        <tbody><tr v-for="(row, i) in blocks.table.rows" :key="i"><td v-for="(cell, j) in row" :key="j">{{ cell }}</td></tr></tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.blocks { display: flex; flex-direction: column; gap: 12px; }
.stats { display: flex; gap: 12px; flex-wrap: wrap; }
.stat { flex: 1; min-width: 130px; border: 1px solid var(--border); background: var(--surface); border-radius: 12px; padding: 12px 14px; display: flex; flex-direction: column; }
.stat span { font-size: 12px; } .stat strong { font-size: 20px; } .stat em { font-style: normal; font-size: 12px; color: var(--ok); }
.chart { padding: 14px 16px; } .head { display: flex; justify-content: space-between; font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.legend { font-weight: 400; font-size: 11px; display: flex; gap: 10px; } .legend i { display: inline-block; width: 8px; height: 8px; border-radius: 2px; margin-right: 4px; }
svg { width: 100%; height: auto; } .axis { stroke: var(--border-strong); } .tick { font-size: 11px; fill: var(--muted); }
.tablewrap { overflow-x: auto; } table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; color: var(--muted); font-weight: 500; } th, td { padding: 8px 12px; } tbody tr { border-top: 1px solid var(--border); }
</style>
