import { defineStore } from 'pinia'
import { ref } from 'vue'
import { infoApi } from '@/api/info'
import type { AppConfig, ServerInfo } from '@/api/types'

/** /api/info and /api/config. */
export const useServer = defineStore('server', () => {
  const info = ref<ServerInfo | null>(null)
  const infoError = ref<unknown>(null)
  const config = ref<AppConfig | null>(null)
  let infoPromise: Promise<ServerInfo | null> | null = null

  async function loadInfo(force = false): Promise<ServerInfo | null> {
    if (info.value && !force) return info.value
    if (infoPromise && !force) return infoPromise
    infoPromise = infoApi
      .info()
      .then((r) => {
        info.value = r
        infoError.value = null
        return r
      })
      .catch((e) => {
        infoError.value = e
        return null
      })
      .finally(() => (infoPromise = null))
    return infoPromise
  }

  async function loadConfig(force = false): Promise<AppConfig> {
    if (config.value && !force) return config.value
    config.value = await infoApi.config()
    return config.value
  }

  async function saveConfig(patch: Partial<AppConfig>): Promise<AppConfig> {
    config.value = await infoApi.saveConfig(patch)
    if (info.value) info.value = { ...info.value, tmdb: config.value.tmdbApiKeySet }
    return config.value
  }

  function setIndexerCounts(total: number, enabled: number) {
    if (info.value) info.value = { ...info.value, indexers: { total, enabled } }
  }

  return { info, infoError, config, loadInfo, loadConfig, saveConfig, setIndexerCounts }
})
