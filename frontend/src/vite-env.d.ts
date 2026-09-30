/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_WAILS_MOCK?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare module '*.vue' {
    import type {DefineComponent} from 'vue'
    const component: DefineComponent<object, object, object>
    export default component
}
