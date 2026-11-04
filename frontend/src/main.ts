import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { locale } from './i18n'
import './styles/fonts.css'
import './styles/tokens.css'
import './styles/base.css'

document.documentElement.lang = locale

createApp(App).use(createPinia()).use(router).mount('#app')
