<script setup lang="ts">
import { computed, onActivated, onMounted, ref } from 'vue'
import type { Indexer } from '@/api/types'
import Badge from '@/components/ui/Badge.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Icon from '@/components/ui/Icon.vue'
import Segmented from '@/components/ui/Segmented.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Spinner from '@/components/ui/Spinner.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { usePolling } from '@/composables/usePolling'
import { msg } from '@/i18n'
import { useIndexers } from '@/stores/indexers'
import { useServer } from '@/stores/server'
import { useToast } from '@/stores/toast'
import { errorText } from '@/utils/errors'
import { formatDateTime } from '@/utils/format'

/** Settings → Trackers: tracker search options, tracker definitions and the tracker list. */
defineOptions({ name: 'PrefsTrackers' })

type Filter = 'all' | 'enabled' | 'public' | 'private'
const indexers = useIndexers()
const server = useServer()
const toast = useToast()

// ---------- tracker search (part of /api/config) ----------
const configError = ref<unknown>(null)
const searchTimeout = ref(30)
const flareSolverr = ref('')
const savingConfig = ref(false)

async function loadConfig() {
  try {
    const c = await server.loadConfig(true)
    searchTimeout.value = c.searchTimeoutSec || 30
    flareSolverr.value = c.flareSolverrUrl ?? ''
    configError.value = null
  } catch (e) {
    configError.value = e
  }
}

async function saveConfig() {
  savingConfig.value = true
  try {
    const c = await server.saveConfig({
      searchTimeoutSec: Math.max(1, Math.round(searchTimeout.value || 30)),
      flareSolverrUrl: flareSolverr.value.trim(),
    })
    searchTimeout.value = c.searchTimeoutSec
    flareSolverr.value = c.flareSolverrUrl
    toast.ok(msg.common.saved)
  } catch (e) {
    toast.error(errorText(e))
  } finally {
    savingConfig.value = false
  }
}

const q = ref('')
const filter = ref<Filter>('all')
const toggling = ref<Set<string>>(new Set())
const filters = [
  { value: 'all' as const, label: msg.indexers.all },
  { value: 'enabled' as const, label: msg.indexers.enabled },
  { value: 'public' as const, label: msg.indexers.public },
  { value: 'private' as const, label: msg.indexers.private },
]

const visible = computed(() => {
  const text = q.value.trim().toLowerCase()
  return indexers.list
    .filter((i) => {
      if (filter.value === 'enabled' && !i.enabled) return false
      if (filter.value === 'public' && i.type !== 'public') return false
      if (filter.value === 'private' && i.type === 'public') return false
      if (!text) return true
      return (
        i.name.toLowerCase().includes(text) ||
        i.id.toLowerCase().includes(text) ||
        i.description.toLowerCase().includes(text)
      )
    })
    .sort((a, b) => Number(b.enabled) - Number(a.enabled) || a.name.localeCompare(b.name))
})

// ---------- tracker definitions ----------
/** Built-in indexer that does not come from the downloaded definitions. */
const BUILT_IN = new Set(['knaben', 'rutracker'])
const defs = computed(() => indexers.definitions)
const noDefinitions = computed(
  () => !!defs.value && defs.value.count === 0 && indexers.loaded && indexers.list.every((i) => BUILT_IN.has(i.id)),
)
// Follow a background update (on server start) until it finishes.
usePolling(
  () => indexers.loadDefinitions(),
  3000,
  computed(() => !!defs.value?.updating && !indexers.updatingDefinitions),
)

async function updateDefinitions() {
  try {
    await indexers.updateDefinitions()
    toast.ok(msg.indexers.definitionsUpdated(indexers.definitions?.count ?? 0))
  } catch (e) {
    toast.error(errorText(e))
  }
}

const typeLabel = (t: Indexer['type']) =>
  t === 'public' ? msg.indexers.typePublic : t === 'private' ? msg.indexers.typePrivate : msg.indexers.typeSemi

async function toggle(i: Indexer) {
  if (toggling.value.has(i.id)) return
  toggling.value = new Set(toggling.value).add(i.id)
  try {
    await indexers.setEnabled(i.id, !i.enabled)
  } catch (e) {
    toast.error(errorText(e))
  } finally {
    const s = new Set(toggling.value)
    s.delete(i.id)
    toggling.value = s
  }
}

onMounted(() => {
  void loadConfig()
  if (!indexers.loaded) void indexers.load()
  void indexers.loadDefinitions()
})
onActivated(() => {
  if (indexers.loaded) void indexers.load()
  if (indexers.definitions) void indexers.loadDefinitions()
})
</script>

<template>
  <div class="panel">
    <form class="card card-pad" @submit.prevent="saveConfig">
      <h2 class="section-title search__title">{{ msg.prefs.trackerSearch }}</h2>
      <ErrorBanner v-if="configError" :error="configError" retry @retry="loadConfig" />
      <template v-else-if="server.config">
        <div class="field">
          <label class="label" for="search-timeout">{{ msg.prefs.searchTimeout }}</label>
          <input id="search-timeout" v-model.number="searchTimeout" class="input" type="number" min="1" max="300" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="flaresolverr">{{ msg.prefs.flareSolverr }}</label>
          <input
            id="flaresolverr"
            v-model="flareSolverr"
            class="input"
            type="url"
            inputmode="url"
            autocomplete="off"
            spellcheck="false"
            placeholder="http://localhost:8191"
            aria-describedby="flaresolverr-hint"
          />
          <span id="flaresolverr-hint" class="hint">{{ msg.prefs.flareSolverrHint }}</span>
        </div>
        <div class="search__actions">
          <UiButton variant="primary" type="submit" icon="check" :loading="savingConfig">{{ msg.common.save }}</UiButton>
        </div>
      </template>
      <Skeleton v-else height="150px" />
    </form>

    <section class="card card-pad defs" :aria-busy="indexers.definitionsBusy || undefined">
      <div class="defs__head">
        <h2 class="section-title">{{ msg.indexers.definitions }}</h2>
        <UiButton icon="download" :loading="indexers.definitionsBusy" @click="updateDefinitions">
          {{ msg.indexers.definitionsUpdate }}
        </UiButton>
      </div>
      <ErrorBanner
        v-if="indexers.definitionsError && !defs"
        :error="indexers.definitionsError"
        retry
        @retry="indexers.loadDefinitions()"
      />
      <template v-else-if="defs">
        <dl class="defs__kv">
          <div>
            <dt>{{ msg.indexers.definitionsCount }}</dt>
            <dd>{{ defs.count }}</dd>
          </div>
          <div>
            <dt>{{ msg.indexers.definitionsUpdatedAt }}</dt>
            <dd>{{ defs.updatedAt ? formatDateTime(defs.updatedAt) : msg.indexers.definitionsNever }}</dd>
          </div>
          <div class="defs__source">
            <dt>{{ msg.indexers.definitionsSource }}</dt>
            <dd class="ellipsis" :title="defs.source">{{ defs.source || '—' }}</dd>
          </div>
        </dl>
        <p v-if="indexers.definitionsBusy" class="defs__busy small"><Spinner :size="14" />{{ msg.indexers.definitionsUpdating }}</p>
        <ErrorBanner v-if="defs.lastError" :error="msg.indexers.definitionsLastError(defs.lastError)" />
      </template>
      <Skeleton v-else height="44px" />
    </section>

    <EmptyState
      v-if="noDefinitions"
      icon="download"
      :title="msg.indexers.definitionsEmptyTitle"
      :text="indexers.definitionsBusy ? msg.indexers.definitionsEmptyBusy : msg.indexers.definitionsEmptyIdle"
    >
      <UiButton v-if="!indexers.definitionsBusy" variant="primary" icon="download" @click="updateDefinitions">
        {{ msg.indexers.definitionsUpdate }}
      </UiButton>
      <Spinner v-else />
    </EmptyState>

    <div class="ix-head">
      <h2 class="section-title">{{ msg.indexers.title }}</h2>
      <span v-if="indexers.loaded" class="small">{{ msg.indexers.counter(indexers.enabledCount, indexers.list.length) }}</span>
    </div>

    <div class="ix-controls">
      <div class="ix-search">
        <Icon name="search" :size="16" class="ix-search__icon" />
        <input v-model="q" class="input ix-search__input" type="search" :placeholder="msg.indexers.search" :aria-label="msg.indexers.search" />
      </div>
      <Segmented v-model="filter" :options="filters" :label="msg.indexers.title" scroll />
    </div>

    <ErrorBanner v-if="indexers.error" :error="indexers.error" retry @retry="indexers.load()" />

    <div v-if="!indexers.loaded && !indexers.error" class="stack">
      <Skeleton v-for="i in 6" :key="i" height="64px" />
    </div>

    <EmptyState v-else-if="indexers.loaded && !visible.length" icon="radar" :text="msg.indexers.empty" />

    <ul v-else class="ix-list">
      <li v-for="i in visible" :key="i.id" class="ix card">
        <RouterLink :to="`/prefs/trackers/${encodeURIComponent(i.id)}`" class="ix__main">
          <span class="ix__name">{{ i.name }}</span>
          <span class="ix__badges">
            <Badge :tone="i.type === 'public' ? 'accent' : 'purple'">{{ typeLabel(i.type) }}</Badge>
            <Badge v-if="i.language" tone="neutral">{{ i.language }}</Badge>
            <Badge v-if="!i.configured" tone="warning">{{ msg.indexers.notConfigured }}</Badge>
            <Badge v-if="i.lastError" tone="error" :title="i.lastError">{{ msg.indexers.error }}</Badge>
          </span>
          <span v-if="i.lastError" class="ix__err ellipsis">{{ i.lastError }}</span>
        </RouterLink>
        <label class="ix__toggle" :title="i.enabled ? msg.indexers.on : msg.indexers.off">
          <input
            type="checkbox"
            role="switch"
            class="sr-only"
            :checked="i.enabled"
            :disabled="toggling.has(i.id)"
            :aria-label="msg.indexers.toggle(i.name)"
            @change="toggle(i)"
          />
          <span class="sw" :class="{ 'sw--on': i.enabled, 'sw--busy': toggling.has(i.id) }" aria-hidden="true"><span class="sw__thumb" /></span>
        </label>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.search__title {
  margin-bottom: 14px;
}
.search__actions {
  display: flex;
  margin-top: 18px;
}
.ix-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: -8px;
}
.defs {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.defs__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.defs__head .section-title {
  flex: 1 1 200px;
}
.defs__kv {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 10px 16px;
  margin: 0;
}
.defs__kv div {
  min-width: 0;
}
.defs__source {
  grid-column: span 2;
}
.defs__kv dt {
  font-size: 11px;
  color: var(--c-muted);
}
.defs__kv dd {
  margin: 2px 0 0;
  font-variant-numeric: tabular-nums;
}
.defs__busy {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}
.ix-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.ix-search {
  position: relative;
  flex: 1 1 240px;
  display: flex;
  align-items: center;
}
.ix-search__icon {
  position: absolute;
  left: 12px;
  color: var(--c-muted);
}
.ix-search__input {
  padding-left: 36px;
  background: var(--c-surface);
}
.ix-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ix {
  display: flex;
  align-items: center;
}
.ix__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  color: var(--c-text);
  border-radius: var(--r-card) 0 0 var(--r-card);
}
.ix__main:hover {
  background: rgba(var(--rgb-accent), 0.03);
}
.ix__name {
  font-weight: 600;
}
.ix__badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.ix__err {
  font-size: 11px;
  color: var(--c-error);
}
.ix__toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  align-self: stretch;
  cursor: pointer;
  border-left: 1px solid var(--c-border);
}
.sw {
  position: relative;
  width: 40px;
  height: 22px;
  border-radius: 11px;
  background: var(--c-border);
  transition: background 0.2s;
}
.sw--on {
  background: var(--c-accent);
}
.sw--busy {
  opacity: 0.5;
}
.sw__thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.2s;
}
.sw--on .sw__thumb {
  transform: translateX(18px);
}
input:focus-visible + .sw {
  outline: 2px solid var(--c-accent);
  outline-offset: 2px;
}
</style>
