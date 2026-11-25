import { createRouter, createWebHistory } from 'vue-router'
import { auth } from './store'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: () => import('./views/Landing.vue') },
    { path: '/login', name: 'login', component: () => import('./views/Login.vue'), meta: { guest: true } },
    { path: '/register', name: 'register', component: () => import('./views/Login.vue'), meta: { guest: true } },
    { path: '/app', name: 'app', component: () => import('./views/Workspace.vue'), meta: { auth: true } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach(async (to) => {
  if (!auth.loaded) await auth.load()
  if (to.meta.auth && !auth.user) return { name: 'login' }
  if (to.meta.guest && auth.user) return { name: 'app' }
})

export default router
