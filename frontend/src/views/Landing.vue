<script setup>
import { auth } from '../store'
import Logo from '../components/Logo.vue'
import ThemeToggle from '../components/ThemeToggle.vue'

const features = [
  { t: 'Cited answers', d: 'Every claim links to the file and page it came from. If your documents do not contain the answer, CypherDoc says so.' },
  { t: 'Rich responses', d: 'Stat cards, tables and charts appear when the data calls for them, and only with numbers found in your documents.' },
  { t: 'Fully self-hosted', d: 'Models run locally through Ollama and data stays in your own Postgres. No API keys, no third-party servers.' },
]
const stack = ['Go', 'Vue 3', 'Postgres + pgvector', 'Ollama']
</script>

<template>
  <div class="page">
    <nav class="nav">
      <Logo />
      <div class="actions">
        <ThemeToggle />
        <router-link v-if="auth.user" class="btn primary" to="/app">Open workspace</router-link>
        <template v-else>
          <router-link class="btn" to="/login">Log in</router-link>
          <router-link class="btn primary" to="/register">Get started</router-link>
        </template>
      </div>
    </nav>
    <header class="hero">
      <span class="badge">Private AI for your documents. Runs on Ollama.</span>
      <h1>Ask your documents anything.<br /><span class="accent">Get answers you can verify.</span></h1>
      <p>Upload PDFs, Word files and notes. CypherDoc answers from your own content with page-level citations, tables and charts, and nothing leaves your servers.</p>
      <div class="cta"><router-link class="btn primary big" :to="auth.user ? '/app' : '/register'">Start for free</router-link></div>
    </header>
    <section class="preview card">
      <div class="bubble">How did Q2 revenue compare with Q1?</div>
      <p class="answer">Revenue grew <b>16.5%</b> quarter over quarter, driven by new enterprise contracts.</p>
      <div class="stats">
        <div class="stat"><span class="muted">Revenue</span><strong>$1.62M</strong><em class="up">+16.5% vs Q1</em></div>
        <div class="stat"><span class="muted">Expenses</span><strong>$1.02M</strong><em class="warn">+8.5% vs Q1</em></div>
        <div class="stat"><span class="muted">Net profit</span><strong>$0.56M</strong><em class="up">+24.4% vs Q1</em></div>
      </div>
      <span class="chip">Q1_Q2_Financial_Report.pdf · p.4</span>
    </section>
    <section class="features">
      <h2>Built for answers, not guesses</h2>
      <div class="grid"><div v-for="f in features" :key="f.t" class="card feature"><h3>{{ f.t }}</h3><p class="muted">{{ f.d }}</p></div></div>
    </section>
    <section class="stack"><span class="muted label">BUILT WITH</span><div><span v-for="s in stack" :key="s" class="pill">{{ s }}</span></div></section>
    <footer class="footer"><span>CypherDoc</span><span class="muted">Self-hosted document intelligence</span></footer>
  </div>
</template>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; padding: 0 24px 24px; }
.nav { display: flex; justify-content: space-between; align-items: center; padding: 22px 0; }
.actions { display: flex; gap: 10px; align-items: center; }
.hero { max-width: 860px; margin: 0 auto; padding: 64px 0 32px; text-align: center; }
.badge { display: inline-block; font-size: 12px; padding: 6px 14px; border: 1px solid var(--border-strong); border-radius: 99px; color: var(--accent-text); margin-bottom: 24px; }
h1 { font-size: clamp(36px, 6vw, 58px); line-height: 1.08; letter-spacing: -0.025em; }
.accent { color: var(--accent-text); }
.hero p { margin: 24px auto 0; max-width: 620px; font-size: 18px; color: var(--text-2); }
.cta { margin-top: 32px; }
.big { min-height: 48px; padding: 0 28px; }
.preview { max-width: 900px; margin: 24px auto 0; padding: 22px; display: flex; flex-direction: column; gap: 14px; align-items: flex-start; }
.bubble { align-self: flex-end; background: var(--accent); color: #fff; padding: 10px 16px; border-radius: 14px 14px 4px 14px; }
.stats { display: flex; gap: 12px; width: 100%; flex-wrap: wrap; }
.stat { flex: 1; min-width: 160px; border: 1px solid var(--border); border-radius: 12px; padding: 14px 16px; display: flex; flex-direction: column; background: var(--surface-2); }
.stat strong { font-size: 22px; } .stat em { font-style: normal; font-size: 12px; }
.up { color: var(--ok); } .warn { color: var(--warn); }
.features { margin-top: 88px; } .features h2 { text-align: center; font-size: 32px; margin-bottom: 28px; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 16px; }
.feature { padding: 24px; } .feature h3 { margin-bottom: 8px; }
.stack { margin-top: 64px; text-align: center; } .label { display: block; font-size: 12px; letter-spacing: 0.08em; margin-bottom: 14px; }
.pill { display: inline-block; margin: 4px; padding: 8px 18px; border: 1px solid var(--border); border-radius: 99px; }
.footer { margin-top: 64px; padding-top: 24px; border-top: 1px solid var(--border); display: flex; justify-content: space-between; font-size: 13px; }
@media (max-width: 520px) {
  .page { padding: 0 16px 16px; }
  .nav { padding: 14px 0; }
  .actions { gap: 8px; }
  .actions .btn:not(.primary) { display: none; }
  .actions .btn { padding: 0 14px; }
  .hero { padding: 36px 0 24px; }
  .hero p { font-size: 16px; }
  .preview { padding: 16px; }
  .features { margin-top: 56px; } .features h2 { font-size: 26px; }
  .footer { flex-direction: column; gap: 4px; }
}
</style>
