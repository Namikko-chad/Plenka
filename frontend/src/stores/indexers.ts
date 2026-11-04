import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { definitionsApi } from '@/api/definitions'
import { indexersApi } from '@/api/indexers'
import type { ConfigValue, DefinitionsState, Indexer, IndexerDetail } from '@/api/types'
import { useServer } from './server'

const summary = (d: IndexerDetail): Indexer => ({
  id: d.id,
  name: d.name,
  description: d.description,
  language: d.language,
  type: d.type,
  links: d.links,
  enabled: d.enabled,
  configured: d.configured,
  lastError: d.lastError,
  lastCheckedAt: d.lastCheckedAt,
})

/** Indexers and tracker definitions (Settings → Trackers tab). */
export const useIndexers = defineStore('indexers', () => {
  const list = ref<Indexer[]>([])
  const loaded = ref(false)
  const error = ref<unknown>(null)

  const enabledCount = computed(() => list.value.filter((i) => i.enabled).length)

  const definitions = ref<DefinitionsState | null>(null)
  const definitionsError = ref<unknown>(null)
  /** This client is waiting on POST /api/definitions/update. */
  const updatingDefinitions = ref(false)
  /** An update is running — ours or a background one on the server. */
  const definitionsBusy = computed(() => updatingDefinitions.value || !!definitions.value?.updating)

  function syncCounts() {
    useServer().setIndexerCounts(list.value.length, enabledCount.value)
  }

  async function load() {
    try {
      list.value = await indexersApi.list()
      loaded.value = true
      error.value = null
      syncCounts()
    } catch (e) {
      error.value = e
    }
  }

  function apply(d: IndexerDetail) {
    const i = list.value.findIndex((x) => x.id === d.id)
    if (i >= 0) list.value.splice(i, 1, summary(d))
    syncCounts()
  }

  async function setEnabled(id: string, enabled: boolean): Promise<IndexerDetail> {
    const d = await indexersApi.update(id, { enabled })
    apply(d)
    return d
  }

  async function save(id: string, body: { enabled: boolean; config: Record<string, ConfigValue> }) {
    const d = await indexersApi.update(id, body)
    apply(d)
    return d
  }

  async function test(id: string) {
    const r = await indexersApi.test(id)
    const i = list.value.findIndex((x) => x.id === id)
    if (i >= 0) {
      list.value.splice(i, 1, { ...list.value[i]!, lastError: r.ok ? '' : r.error, lastCheckedAt: new Date().toISOString() })
    }
    return r
  }

  /** Loads the definitions state; reloads the list once a background update has finished. */
  async function loadDefinitions() {
    const wasUpdating = !!definitions.value?.updating
    try {
      definitions.value = await definitionsApi.get()
      definitionsError.value = null
    } catch (e) {
      definitionsError.value = e
      return
    }
    if (wasUpdating && !definitions.value.updating) await load()
  }

  /** Downloads the definitions now (joins a running update), then reloads the list. */
  async function updateDefinitions() {
    if (updatingDefinitions.value) return
    updatingDefinitions.value = true
    try {
      definitions.value = await definitionsApi.update()
      definitionsError.value = null
    } catch (e) {
      // A failed update keeps the previous set; refresh the state to show its lastError.
      await loadDefinitions()
      throw e
    } finally {
      updatingDefinitions.value = false
    }
    await load()
  }

  return {
    list,
    loaded,
    error,
    enabledCount,
    load,
    setEnabled,
    save,
    test,
    get: indexersApi.get,
    definitions,
    definitionsError,
    updatingDefinitions,
    definitionsBusy,
    loadDefinitions,
    updateDefinitions,
  }
})
