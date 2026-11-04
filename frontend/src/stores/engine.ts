import { defineStore } from 'pinia'
import { ref } from 'vue'
import { engineApi } from '@/api/engine'
import type { EngineSettings } from '@/api/types'

/** Torrent engine settings. */
export const useEngine = defineStore('engine', () => {
  const sets = ref<EngineSettings | null>(null)
  const error = ref<unknown>(null)

  async function load() {
    try {
      sets.value = await engineApi.settings()
      error.value = null
    } catch (e) {
      error.value = e
    }
  }

  async function save(next: EngineSettings) {
    sets.value = await engineApi.save(next)
  }

  async function resetDefaults() {
    sets.value = await engineApi.reset()
  }

  return { sets, error, load, save, resetDefaults }
})
