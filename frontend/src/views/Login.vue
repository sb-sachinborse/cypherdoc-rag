<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { auth } from '../store'
import RagAnimation from '../components/RagAnimation.vue'
import ThemeToggle from '../components/ThemeToggle.vue'

const route = useRoute()
const router = useRouter()
const isRegister = computed(() => route.name === 'register')
const name = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    if (isRegister.value) await auth.register(name.value, email.value, password.value)
    else await auth.login(email.value, password.value)
    router.push({ name: 'app' })
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="split">
    <aside class="left"><RagAnimation /></aside>
    <main class="right">
      <div class="theme"><ThemeToggle /></div>
      <form class="form" @submit.prevent="submit">
        <div>
          <h2>{{ isRegister ? 'Create your account' : 'Welcome back' }}</h2>
          <p class="muted">{{ isRegister ? 'Start asking your documents questions.' : 'Log in to your CypherDoc workspace.' }}</p>
        </div>
        <div v-if="isRegister" class="field"><label for="name">Name</label><input id="name" v-model="name" class="input" autocomplete="name" maxlength="60" placeholder="Your name" /></div>
        <div class="field"><label for="email">Email</label><input id="email" v-model="email" class="input" type="email" autocomplete="email" required placeholder="you@company.com" /></div>
        <div class="field">
          <label for="pw">Password</label>
          <input id="pw" v-model="password" class="input" type="password" required :minlength="isRegister ? 10 : 1" maxlength="72" :autocomplete="isRegister ? 'new-password' : 'current-password'" :placeholder="isRegister ? 'At least 10 characters' : 'Enter your password'" />
        </div>
        <p v-if="error" class="error" role="alert">{{ error }}</p>
        <button class="btn primary" type="submit" :disabled="busy">{{ busy ? 'Please wait…' : isRegister ? 'Create account' : 'Log in' }}</button>
        <p class="muted center">
          <template v-if="isRegister">Already have an account? <router-link class="link" to="/login">Log in</router-link></template>
          <template v-else>New to CypherDoc? <router-link class="link" to="/register">Create an account</router-link></template>
        </p>
        <router-link class="muted center" to="/">Back to home</router-link>
      </form>
    </main>
  </div>
</template>

<style scoped>
.split { display: flex; min-height: 100%; }
.left { flex: 1; min-width: 0; }
.right { width: 560px; flex: none; display: grid; place-items: center; padding: 48px; position: relative; }
.theme { position: absolute; top: 20px; right: 20px; }
.form { width: 100%; max-width: 380px; display: flex; flex-direction: column; gap: 18px; }
h2 { font-size: 28px; letter-spacing: -0.02em; margin-bottom: 8px; }
.field { display: flex; flex-direction: column; gap: 7px; }
label { font-size: 13px; color: var(--text-2); }
.error { color: var(--err); background: var(--err-bg); padding: 10px 12px; border-radius: 8px; font-size: 13px; }
.center { text-align: center; font-size: 13px; }
.link { color: var(--accent-text); }
@media (max-width: 960px) { .left { display: none; } .right { width: 100%; } }
</style>
