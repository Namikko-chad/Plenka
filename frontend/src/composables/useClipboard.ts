import { msg } from '@/i18n'
import { useToast } from '@/stores/toast'

/** Copies text with a toast; falls back to execCommand on plain-HTTP LAN origins. */
export function useClipboard() {
  const toast = useToast()
  return async function copy(text: string) {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text)
      } else {
        const ta = document.createElement('textarea')
        ta.value = text
        ta.setAttribute('readonly', '')
        ta.style.position = 'fixed'
        ta.style.opacity = '0'
        document.body.appendChild(ta)
        ta.select()
        const ok = document.execCommand('copy')
        ta.remove()
        if (!ok) throw new Error('copy failed')
      }
      toast.ok(msg.common.copied)
    } catch {
      toast.error(msg.common.copyFailed)
    }
  }
}
