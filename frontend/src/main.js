import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { theme } from './store'
import './styles.css'

theme.apply()
createApp(App).use(router).mount('#app')
