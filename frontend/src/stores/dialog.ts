import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface ConfirmRequest {
  title: string
  text?: string
  confirmLabel?: string
  danger?: boolean
  resolve: (ok: boolean) => void
}

/** One global confirm dialog, rendered by <ConfirmDialog> in App.vue. */
export const useDialog = defineStore('dialog', () => {
  const current = ref<ConfirmRequest | null>(null)

  function confirm(opts: Omit<ConfirmRequest, 'resolve'>): Promise<boolean> {
    current.value?.resolve(false)
    return new Promise((resolve) => {
      current.value = { ...opts, resolve }
    })
  }

  function answer(ok: boolean) {
    const c = current.value
    current.value = null
    c?.resolve(ok)
  }

  return { current, confirm, answer }
})
