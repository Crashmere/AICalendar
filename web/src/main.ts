import {createApp} from 'vue'
import App from './App.vue'
import './style.css'
document.addEventListener('keydown',e=>{if(e.key==='Tab'&&!e.ctrlKey&&!e.metaKey&&!e.altKey&&!e.isComposing){e.preventDefault();e.stopImmediatePropagation()}},true)
createApp(App).mount('#app')
