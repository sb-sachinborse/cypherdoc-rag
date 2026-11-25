<template>
  <div class="rag" aria-hidden="false">
    <div class="glow g1"></div><div class="glow g2"></div><div class="grid"></div>
    <span class="drift d1">chunk #42 · score 0.91</span>
    <span class="drift d2">page 4 · HR_Policy.pdf</span>
    <span class="drift d3">768-dim vector</span>
    <div class="content">
      <div class="logo"><div class="logo-mark">C</div><span>CypherDoc</span></div>
      <div>
        <h1>Your documents,<br />now answering back.</h1>
        <p>Watch how a question travels: from your files to chunks, vectors and a cited answer.</p>
      </div>
      <svg viewBox="0 0 700 380" role="img" aria-label="Animated diagram: documents are split into chunks, embedded into a vector database, retrieved and answered by an LLM">
        <g v-for="(y, i) in [50, 150, 250]" :key="'d' + y" class="pulse" :style="{ animationDelay: i * 0.5 + 's' }">
          <rect x="20" :y="y" width="64" height="78" rx="9" class="node" />
          <rect x="32" :y="y + 16" width="40" height="4" rx="2" class="bar-accent" />
          <rect x="32" :y="y + 28" width="32" height="4" rx="2" class="bar-dim" />
          <rect x="32" :y="y + 40" width="38" height="4" rx="2" class="bar-dim" />
        </g>
        <rect v-for="(y, i) in [70, 108, 146, 184, 222, 260]" :key="'c' + y" class="pulse node" :class="i % 2 ? 'green' : 'blue'" :style="{ animationDelay: i * 0.35 + 's' }" x="190" :y="y" width="92" height="22" rx="6" />
        <g v-for="(p, i) in paths" :key="'p' + i">
          <path class="dash" :d="p.d" fill="none" />
          <circle r="4" class="travel"><animateMotion :dur="p.dur" :begin="p.begin" repeatCount="indefinite" :path="p.d" /></circle>
        </g>
        <ellipse cx="400" cy="226" rx="52" ry="15" class="node" />
        <rect x="348" y="176" width="104" height="50" class="db-body" />
        <line x1="348" y1="176" x2="348" y2="226" class="db-line" /><line x1="452" y1="176" x2="452" y2="226" class="db-line" />
        <ellipse cx="400" cy="176" rx="52" ry="15" class="node blue" />
        <circle v-for="(d, i) in dots" :key="'v' + i" class="pulse vec" :style="{ animationDelay: i * 0.3 + 's' }" :cx="d[0]" :cy="d[1]" r="3.5" />
        <g class="pulse"><rect x="548" y="160" width="80" height="80" rx="18" class="node blue thick" /><text x="588" y="207" text-anchor="middle" class="llm">LLM</text></g>
        <rect class="pulse node green" x="350" y="20" width="100" height="32" rx="16" /><text x="400" y="41" text-anchor="middle" class="q">your question</text>
        <g text-anchor="middle" class="lbl"><text x="52" y="350">Documents</text><text x="236" y="350">Chunks</text><text x="400" y="350">pgvector</text><text x="588" y="350">Ollama</text></g>
      </svg>
      <div class="answer">
        <div class="src">Based on HR_Policy.pdf, page 4</div>
        <div><span class="type">Full-time employees receive 24 days of paid leave per year.</span><span class="caret"></span></div>
      </div>
    </div>
  </div>
</template>

<script setup>
const paths = [
  { d: 'M84 89 C140 89 150 81 190 81', begin: '0s', dur: '3s' },
  { d: 'M84 189 C140 189 150 157 190 157', begin: '1s', dur: '3s' },
  { d: 'M84 289 C140 289 150 233 190 233', begin: '2s', dur: '3s' },
  { d: 'M282 81 C320 81 320 190 352 195', begin: '0.5s', dur: '3s' },
  { d: 'M282 157 C320 157 330 200 352 200', begin: '1.5s', dur: '3s' },
  { d: 'M282 233 C320 233 330 210 352 205', begin: '2.5s', dur: '3s' },
  { d: 'M400 52 L400 168', begin: '0.8s', dur: '2.4s' },
  { d: 'M450 200 L548 200', begin: '1.2s', dur: '2s' },
]
const dots = [[372, 176], [430, 172], [385, 236], [424, 230], [448, 200]]
</script>

<style scoped>
.rag { position: relative; overflow: hidden; height: 100%; background: var(--hero-bg); border-right: 1px solid var(--hero-border); color: var(--text); }
.content { position: relative; height: 100%; padding: 48px; display: flex; flex-direction: column; justify-content: space-between; gap: 16px; }
h1 { font-size: 40px; line-height: 1.15; letter-spacing: -0.02em; }
p { margin-top: 14px; font-size: 15px; color: var(--hero-text); max-width: 520px; }
svg { width: 100%; max-height: 380px; }
.grid { position: absolute; inset: 0; background-image: linear-gradient(var(--grid) 1px, transparent 1px), linear-gradient(90deg, var(--grid) 1px, transparent 1px); background-size: 40px 40px; }
.glow { position: absolute; border-radius: 50%; animation: glow 12s ease-in-out infinite alternate; }
.g1 { left: -120px; top: -100px; width: 520px; height: 520px; background: radial-gradient(circle, rgba(37, 99, 235, 0.4), transparent 65%); }
.g2 { right: -140px; bottom: -60px; width: 480px; height: 480px; background: radial-gradient(circle, rgba(52, 201, 139, 0.2), transparent 65%); animation-delay: -6s; }
.drift { position: absolute; font-size: 11px; padding: 5px 10px; border: 1px solid var(--hero-line); border-radius: 99px; color: var(--hero-text); background: var(--hero-card); animation: drift 7s ease-in-out infinite; }
.d1 { right: 60px; top: 60px; } .d2 { right: 210px; top: 150px; animation-delay: -2s; } .d3 { left: 58%; top: 230px; animation-delay: -4s; }
.node { fill: var(--hero-card); stroke: var(--hero-line); }
.node.blue { stroke: #4f8dff; } .node.green { stroke: #34c98b; } .thick { stroke-width: 2; }
.bar-accent { fill: #4f8dff; } .bar-dim { fill: var(--hero-line); }
.db-body { fill: var(--hero-card); } .db-line { stroke: var(--hero-line); }
.vec { fill: #34c98b; }
.dash { stroke: #4f8dff; stroke-width: 1.6; opacity: 0.8; stroke-dasharray: 5 6; animation: flow 1.2s linear infinite; }
.travel { fill: var(--accent-text); }
.llm { font-size: 18px; font-weight: 600; fill: var(--accent-text); }
.q { font-size: 12px; fill: var(--ok); }
.lbl { font-size: 12px; fill: var(--accent-text); }
.pulse { transform-box: fill-box; transform-origin: center; animation: pulse 2.4s ease-in-out infinite; }
.answer { position: relative; background: var(--hero-card); border: 1px solid var(--hero-line); border-radius: 14px; padding: 16px 18px; max-width: 520px; }
.src { font-size: 12px; color: var(--accent-text); margin-bottom: 6px; }
.type { display: inline-block; animation: type 7s steps(40, end) infinite; }
.caret { display: inline-block; width: 2px; height: 15px; background: #4f8dff; vertical-align: -2px; margin-left: 3px; animation: blink 1s steps(1) infinite; }
@keyframes flow { to { stroke-dashoffset: -22; } }
@keyframes pulse { 0%, 100% { opacity: 0.55; transform: scale(1); } 50% { opacity: 1; transform: scale(1.07); } }
@keyframes drift { 0%, 100% { transform: translateY(0); opacity: 0.35; } 50% { transform: translateY(-18px); opacity: 0.9; } }
@keyframes glow { from { transform: translate(0, 0) scale(1); } to { transform: translate(60px, 40px) scale(1.25); } }
@keyframes type { 0% { clip-path: inset(0 100% 0 0); } 55%, 100% { clip-path: inset(0 0 0 0); } }
@keyframes blink { 50% { opacity: 0; } }
@media (prefers-reduced-motion: reduce) { .dash, .pulse, .drift, .glow, .type, .caret { animation: none; } }
</style>
