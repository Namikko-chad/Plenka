import { ref, shallowRef } from 'vue'
import { isAbort } from '@/api/http'

/** Loading / error / data triple for one async loader; stale calls are aborted. */
export function useAsync<T, A extends unknown[] = []>(loader: (signal: AbortSignal, ...args: A) => Promise<T>) {
  const data = shallowRef<T | null>(null)
  const loading = ref(false)
  const error = ref<unknown>(null)
  let ctrl: AbortController | null = null

  async function run(...args: A): Promise<T | null> {
    ctrl?.abort()
    const c = new AbortController()
    ctrl = c
    loading.value = true
    error.value = null
    try {
      const r = await loader(c.signal, ...args)
      if (ctrl === c) data.value = r
      return r
    } catch (e) {
      if (ctrl === c && !isAbort(e)) error.value = e
      return null
    } finally {
      if (ctrl === c) loading.value = false
    }
  }

  function cancel() {
    ctrl?.abort()
    ctrl = null
    loading.value = false
  }

  return { data, loading, error, run, cancel }
}
