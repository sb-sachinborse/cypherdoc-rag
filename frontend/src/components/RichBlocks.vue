<script setup>
import { computed } from 'vue'
const props = defineProps({ blocks: { type: Object, required: true } })
const colors = ['var(--bar1)', 'var(--bar2)', 'var(--bar3)']
const pieColors = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#6250d6', '#008300', '#e34948', '#898781', '#14b8a6', '#a855f7', '#f97316']
const W = 600, H = 190, PAD = 28

function fmt(n) {
  if (Number.isInteger(n)) return n.toLocaleString('en-US')
  return n.toLocaleString('en-US', { maximumFractionDigits: 2 })
}

const kind = computed(() => props.blocks.chart?.kind || 'bar')

const bars = computed(() => {
  const c = props.blocks.chart
  if (!c || kind.value !== 'bar') return null
  const max = Math.max(1, ...c.series.flatMap((s) => s.values))
  const groupW = (W - PAD * 2) / c.labels.length
  const barW = Math.min(24, (groupW - 8) / c.series.length)
  const items = []
  c.labels.forEach((label, i) => {
    c.series.forEach((s, j) => {
      const h = (s.values[i] / max) * (H - 40)
      items.push({
        key: i + '-' + j,
        x: PAD + i * groupW + (groupW - barW * c.series.length) / 2 + j * barW,
        y: H - 22 - h, w: barW - 2, h, fill: colors[j], title: `${s.name} · ${label}: ${fmt(s.values[i])}`,
      })
    })
  })
  return { items, ticks: c.labels.map((l, i) => ({ l, x: PAD + i * groupW + groupW / 2 })), max }
})

const lines = computed(() => {
  const c = props.blocks.chart
  if (!c || kind.value !== 'line') return null
  const max = Math.max(1, ...c.series.flatMap((s) => s.values))
  const step = (W - PAD * 2) / Math.max(1, c.labels.length - 1)
  const y = (v) => H - 22 - (v / max) * (H - 40)
  const series = c.series.map((s, j) => ({
    name: s.name, color: colors[j],
    points: s.values.map((v, i) => ({ x: PAD + i * step, y: y(v), title: `${s.name} · ${c.labels[i]}: ${fmt(v)}` })),
  }))
  series.forEach((s) => { s.d = s.points.map((p, i) => (i ? 'L' : 'M') + p.x.toFixed(1) + ' ' + p.y.toFixed(1)).join(' ') })
  return { series, ticks: c.labels.map((l, i) => ({ l, x: PAD + i * step })) }
})

const pie = computed(() => {
  const c = props.blocks.chart
  if (!c || kind.value !== 'pie') return null
  const vals = c.series[0].values
  const total = vals.reduce((a, b) => a + b, 0) || 1
  let a0 = -Math.PI / 2
  const R = 88, CX = 100, CY = 100
  const slices = vals.map((v, i) => {
    const frac = v / total
    const a1 = a0 + frac * Math.PI * 2
    const large = frac > 0.5 ? 1 : 0
    const p = (a) => `${(CX + R * Math.cos(a)).toFixed(2)} ${(CY + R * Math.sin(a)).toFixed(2)}`
    const d = frac >= 0.9999 ? null : `M${CX} ${CY} L${p(a0)} A${R} ${R} 0 ${large} 1 ${p(a1)} Z`
    a0 = a1
    return { d, full: frac >= 0.9999, color: pieColors[i % pieColors.length], label: c.labels[i], value: v, pct: Math.round(frac * 1000) / 10 }
  })
  return { slices, CX, CY, R }
})
</script>

<template>
  <div class="blocks">
    <div v-if="blocks.stats?.length" class="stats">
      <div v-for="s in blocks.stats" :key="s.label" class="stat">
        <span class="muted">{{ s.label }}</span><strong>{{ s.value }}</strong><em v-if="s.delta">{{ s.delta }}</em>
      </div>
    </div>

    <div v-if="blocks.chart" class="card chart">
      <div class="head">
        <span>{{ blocks.chart.title || 'Chart' }}</span>
        <span v-if="kind !== 'pie'" class="legend muted"><span v-for="(s, j) in blocks.chart.series" :key="s.name"><i :style="{ background: colors[j] }"></i>{{ s.name }}</span></span>
      </div>

      <svg v-if="bars" :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="blocks.chart.title || 'Bar chart'">
        <line :x1="PAD" :x2="W - PAD" :y1="H - 22" :y2="H - 22" class="axis" />
        <rect v-for="b in bars.items" :key="b.key" :x="b.x" :y="b.y" :width="b.w" :height="b.h" :fill="b.fill" rx="3"><title>{{ b.title }}</title></rect>
        <text v-for="t in bars.ticks" :key="t.l + t.x" :x="t.x" :y="H - 6" text-anchor="middle" class="tick">{{ t.l }}</text>
      </svg>

      <svg v-else-if="lines" :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="blocks.chart.title || 'Line chart'">
        <line :x1="PAD" :x2="W - PAD" :y1="H - 22" :y2="H - 22" class="axis" />
        <g v-for="s in lines.series" :key="s.name">
          <path :d="s.d" fill="none" :stroke="s.color" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round" />
          <circle v-for="(p, i) in s.points" :key="i" :cx="p.x" :cy="p.y" r="4" :fill="s.color"><title>{{ p.title }}</title></circle>
        </g>
        <text v-for="t in lines.ticks" :key="t.l + t.x" :x="t.x" :y="H - 6" text-anchor="middle" class="tick">{{ t.l }}</text>
      </svg>

      <div v-else-if="pie" class="piewrap">
        <svg viewBox="0 0 200 200" class="pie" role="img" :aria-label="blocks.chart.title || 'Pie chart'">
          <template v-for="(s, i) in pie.slices" :key="i">
            <circle v-if="s.full" :cx="pie.CX" :cy="pie.CY" :r="pie.R" :fill="s.color"><title>{{ s.label }}: {{ fmt(s.value) }}</title></circle>
            <path v-else :d="s.d" :fill="s.color" stroke="var(--surface)" stroke-width="2"><title>{{ s.label }}: {{ fmt(s.value) }} ({{ s.pct }}%)</title></path>
          </template>
        </svg>
        <ul class="plegend">
          <li v-for="(s, i) in pie.slices" :key="i"><i :style="{ background: s.color }"></i><span>{{ s.label }}</span><b>{{ s.pct }}%</b></li>
        </ul>
      </div>
    </div>

    <div v-if="blocks.table" class="card tablewrap">
      <table>
        <thead><tr><th v-for="c in blocks.table.columns" :key="c">{{ c }}</th></tr></thead>
        <tbody><tr v-for="(row, i) in blocks.table.rows" :key="i"><td v-for="(cell, j) in row" :key="j">{{ cell }}</td></tr></tbody>
      </table>
    </div>
    <p v-if="blocks.derived" class="note muted">Some figures were counted by the model, not copied from your documents. Please verify them.</p>
  </div>
</template>

<style scoped>
.note { font-size: 12px; }
.blocks { display: flex; flex-direction: column; gap: 12px; }
.stats { display: flex; gap: 12px; flex-wrap: wrap; }
.stat { flex: 1; min-width: 130px; border: 1px solid var(--border); background: var(--surface); border-radius: 12px; padding: 12px 14px; display: flex; flex-direction: column; }
.stat span { font-size: 12px; } .stat strong { font-size: 20px; } .stat em { font-style: normal; font-size: 12px; color: var(--ok); }
.chart { padding: 14px 16px; } .head { display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.legend { font-weight: 400; font-size: 11px; display: flex; gap: 10px; flex-wrap: wrap; } .legend i, .plegend i { display: inline-block; width: 8px; height: 8px; border-radius: 2px; margin-right: 4px; }
svg { width: 100%; height: auto; } .axis { stroke: var(--border-strong); } .tick { font-size: 11px; fill: var(--muted); }
.piewrap { display: flex; align-items: center; gap: 20px; flex-wrap: wrap; } .pie { width: 190px; flex: none; }
.plegend { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; font-size: 13px; flex: 1; min-width: 140px; }
.plegend li { display: flex; align-items: center; gap: 4px; } .plegend span { flex: 1; overflow-wrap: anywhere; } .plegend b { font-weight: 500; }
.tablewrap { overflow-x: auto; } table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; color: var(--muted); font-weight: 500; white-space: nowrap; } th, td { padding: 8px 12px; } tbody tr { border-top: 1px solid var(--border); }
</style>
