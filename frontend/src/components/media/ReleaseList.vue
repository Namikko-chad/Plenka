<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Release, SearchResponse } from '@/api/types'
import Chip from '@/components/ui/Chip.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Icon from '@/components/ui/Icon.vue'
import Segmented from '@/components/ui/Segmented.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Spinner from '@/components/ui/Spinner.vue'
import StatusDot from '@/components/ui/StatusDot.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { msg } from '@/i18n'
import { dedupKey, presentQualities, qualityOf, sortReleases, type Quality, type ReleaseSort } from '@/utils/releases'
import ReleaseTile from './ReleaseTile.vue'

const props = defineProps<{
  response: SearchResponse | null
  loading: boolean
  error: unknown
  busyKey?: string | null
}>()
defineEmits<{ select: [Release]; retry: [] }>()

const PAGE = 60
const sort = ref<ReleaseSort>('seeders')
const quality = ref<Quality | null>(null)
const limit = ref(PAGE)
const showIndexers = ref(false)

const results = computed(() => props.response?.results ?? [])
const indexers = computed(() => props.response?.indexers ?? [])
const okCount = computed(() => indexers.value.filter((i) => i.status === 'ok').length)
const qualities = computed(() => presentQualities(results.value))
const filtered = computed(() => {
  const list = quality.value ? results.value.filter((r) => qualityOf(r.title) === quality.value) : results.value
  return sortReleases(list, sort.value)
})
const visible = computed(() => filtered.value.slice(0, limit.value))

watch(
  () => props.response,
  () => {
    limit.value = PAGE
    if (quality.value && !qualities.value.includes(quality.value)) quality.value = null
  },
)
watch([sort, quality], () => (limit.value = PAGE))

const sortOptions = [
  { value: 'seeders' as const, label: msg.releases.bySeeders },
  { value: 'size' as const, label: msg.releases.bySize },
  { value: 'date' as const, label: msg.releases.byDate },
]
</script>

<template>
  <div class="rlist">
    <div v-if="results.length" class="rlist__controls">
      <Segmented v-model="sort" :options="sortOptions" :label="msg.releases.sort" />
      <div v-if="qualities.length > 1" class="rlist__chips" role="group" :aria-label="msg.releases.quality">
        <Chip :active="quality === null" @click="quality = null">{{ msg.releases.any }}</Chip>
        <Chip v-for="q in qualities" :key="q" :active="quality === q" @click="quality = q">{{ q }}</Chip>
      </div>
    </div>

    <div v-if="indexers.length" class="rlist__idx card">
      <button type="button" class="rlist__idx-head" :aria-expanded="showIndexers" @click="showIndexers = !showIndexers">
        <StatusDot :ok="okCount > 0" />
        <span class="spacer">{{ msg.releases.indexersSummary(okCount, indexers.length) }}</span>
        <Spinner v-if="loading" :size="14" />
        <Icon name="chevronDown" :size="16" class="rlist__chev" :class="{ 'rlist__chev--open': showIndexers }" />
      </button>
      <ul v-if="showIndexers" class="rlist__idx-list">
        <li v-for="i in indexers" :key="i.id" class="rlist__idx-item">
          <StatusDot :ok="i.status === 'ok'" />
          <span class="rlist__idx-name">{{ i.name }}</span>
          <span v-if="i.status === 'ok'" class="muted">{{ msg.releases.results(i.results) }} · {{ msg.format.ms(i.elapsedMs) }}</span>
          <span v-else class="rlist__idx-err">{{ i.error || msg.indexers.error }}</span>
        </li>
      </ul>
    </div>

    <ErrorBanner v-if="error" :error="error" retry @retry="$emit('retry')" />

    <div v-if="loading && !results.length" class="stack" :aria-label="msg.releases.searching">
      <p class="muted rlist__searching"><Spinner :size="14" /> {{ msg.releases.searching }}</p>
      <Skeleton v-for="i in 6" :key="i" height="74px" />
    </div>

    <template v-else-if="response && !error">
      <EmptyState v-if="!indexers.length" icon="radar" :text="msg.releases.noIndexers">
        <UiButton to="/prefs/trackers" icon="radar">{{ msg.releases.toIndexers }}</UiButton>
      </EmptyState>
      <EmptyState v-else-if="!results.length" icon="search" :text="msg.releases.empty" />
      <EmptyState v-else-if="!filtered.length" icon="search" :text="msg.releases.filtered" />
      <div v-else class="stack">
        <ReleaseTile
          v-for="r in visible"
          :key="dedupKey(r)"
          :release="r"
          :busy="busyKey === dedupKey(r)"
          :disabled="!!busyKey"
          @select="$emit('select', r)"
        />
        <UiButton v-if="filtered.length > visible.length" variant="neutral" block @click="limit += PAGE">
          {{ msg.releases.more(filtered.length - visible.length) }}
        </UiButton>
      </div>
    </template>
  </div>
</template>

<style scoped>
.rlist {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.rlist__controls {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}
.rlist__chips {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  scrollbar-width: none;
}
.rlist__idx-head {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: var(--touch);
  padding: 0 14px;
  border: none;
  background: none;
  font-size: 12px;
  color: var(--c-text);
  text-align: left;
}
.rlist__chev {
  color: var(--c-muted);
  transition: transform var(--dur);
}
.rlist__chev--open {
  transform: rotate(180deg);
}
.rlist__idx-list {
  list-style: none;
  margin: 0;
  padding: 4px 14px 12px;
  border-top: 1px solid var(--c-border);
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 11px;
}
.rlist__idx-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-top: 6px;
}
.rlist__idx-name {
  min-width: 90px;
  font-weight: 600;
}
.rlist__idx-err {
  color: var(--c-error);
  overflow-wrap: anywhere;
}
.rlist__searching {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}
</style>
