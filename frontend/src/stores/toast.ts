import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastTone = 'info' | 'ok' | 'error'
export interface Toast {
  id: number
  text: string
  tone: ToastTone
}

let seq = 0

export const useToast = defineStore('toast', () => {
  const items = ref<Toast[]>([])

  function dismiss(id: number) {
    items.value = items.value.filter((t) => t.id !== id)
  }

  function show(text: string, tone: ToastTone = 'info', ms = tone === 'error' ? 5000 : 2500) {
    const id = ++seq
    items.value = [...items.value.slice(-2), { id, text, tone }]
    setTimeout(() => dismiss(id), ms)
  }

  return { items, dismiss, ok: (t: string) => show(t, 'ok'), error: (t: string) => show(t, 'error') }
})
