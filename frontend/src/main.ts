import {createApp} from 'vue'
import App from './App.vue'
import './style.css';
import { initializeAppServices } from './services/appInitializationService';

// Initialize all core app services at startup
initializeAppServices(); // Run immediately when the app starts

createApp(App).mount('#app')
