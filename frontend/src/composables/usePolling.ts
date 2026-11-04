import { onActivated, onBeforeUnmount, onDeactivated, onMounted, watch, type Ref } from 'vue'

/**
 * Runs `fn` every `ms` while the component is active (mounted / KeepAlive-activated),
 * the document is visible and `enabled` (optional) is true. Calls never overlap.
 */
export function usePolling(fn: () => Promise<unknown> | void, ms: number, enabled?: Ref<boolean>) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let active = false
  let running = false

  const canRun = () => active && document.visibilityState === 'visible' && (enabled ? enabled.value : true)

  async function tick() {
    timer = undefined
    if (!canRun() || running) return
    running = true
    try {
      await fn()
    } catch {
      /* the caller tracks its own errors */
    } finally {
      running = false
      if (canRun()) timer = setTimeout(tick, ms)
    }
  }

  function start() {
    stop()
    if (canRun()) void tick()
  }

  function stop() {
    if (timer) clearTimeout(timer)
    timer = undefined
  }

  const onVisibility = () => (document.visibilityState === 'visible' ? start() : stop())

  onMounted(() => {
    active = true
    document.addEventListener('visibilitychange', onVisibility)
    start()
  })
  onActivated(() => {
    if (!active) {
      active = true
      start()
    }
  })
  onDeactivated(() => {
    active = false
    stop()
  })
  onBeforeUnmount(() => {
    active = false
    stop()
    document.removeEventListener('visibilitychange', onVisibility)
  })
  if (enabled) watch(enabled, (v) => (v ? start() : stop()))
}
