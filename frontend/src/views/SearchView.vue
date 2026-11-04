<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { searchReleases } from '@/api/search'
import type { Media, SearchResponse } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import MediaGrid from '@/components/media/MediaGrid.vue'
import ReleaseList from '@/components/media/ReleaseList.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Icon from '@/components/ui/Icon.vue'
import IconButton from '@/components/ui/IconButton.vue'
import Segmented from '@/components/ui/Segmented.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { useAddRelease } from '@/composables/useAddRelease'
import { useAsync } from '@/composables/useAsync'
import { msg } from '@/i18n'
import { useCatalog } from '@/stores/catalog'
import { useServer } from '@/stores/server'

defineOptions({ name: 'SearchView' })

type Mode = 'catalog' | 'trackers'
const route = useRoute()
const router = useRouter()
const catalog = useCatalog()
const server = useServer()
const { busyKey, add } = useAddRelease()

const mode = ref<Mode>(route.query.mode === 'trackers' ? 'trackers' : 'catalog')
const query = ref('')
const input = ref<HTMLInputElement | null>(null)
const modes = [
  { value: 'catalog' as const, label: msg.search.catalog },
  { value: 'trackers' as const, label: msg.search.trackers },
]

const media = useAsync<Media[], [string]>((signal, q) => catalog.search(q, signal))
const releases = useAsync<SearchResponse, [string]>((signal, q) =>
  searchReleases(q, { signal, timeoutSec: server.config?.searchTimeoutSec }),
)
const trackersQuery = ref('')

let debounce: ReturnType<typeof setTimeout> | undefined
watch(query, (q) => {
  if (mode.value !== 'catalog') return
  clearTimeout(debounce)
  const text = q.trim()
  if (!text) {
    media.cancel()
    media.data.value = null
    media.error.value = null
    return
  }
  debounce = setTimeout(() => void media.run(text), 500)
})

watch(mode, (m) => {
  void router.replace({ query: m === 'trackers' ? { mode: 'trackers' } : {} })
  if (m === 'catalog' && query.value.trim()) void media.run(query.value.trim())
  input.value?.focus()
})
watch(
  () => route.query.mode,
  (v) => {
    if (route.name === 'search') mode.value = v === 'trackers' ? 'trackers' : 'catalog'
  },
)

function submit() {
  const q = query.value.trim()
  if (!q) return
  if (mode.value === 'trackers') {
    trackersQuery.value = q
    void releases.run(q)
  } else {
    clearTimeout(debounce)
    void media.run(q)
  }
  input.value?.blur()
}

function clear() {
  query.value = ''
  input.value?.focus()
}

const tmdbOff = computed(() => server.info !== null && !server.info.tmdb)
onMounted(() => {
  if (!route.query.mode && tmdbOff.value) mode.value = 'trackers'
  input.value?.focus()
})
</script>

<template>
  <div class="page">
    <AppBar :title="msg.search.title" large />
    <div class="page-body">
      <form class="search" role="search" @submit.prevent="submit">
        <div class="search__field">
          <Icon name="search" :size="18" class="search__icon" />
          <input
            ref="input"
            v-model="query"
            class="input search__input"
            type="search"
            enterkeyhint="search"
            autocomplete="off"
            :aria-label="mode === 'catalog' ? msg.search.placeholderCatalog : msg.search.placeholderTrackers"
            :placeholder="mode === 'catalog' ? msg.search.placeholderCatalog : msg.search.placeholderTrackers"
          />
          <IconButton v-if="query" icon="close" :label="msg.search.clear" :size="16" class="search__clear" @click="clear" />
        </div>
        <div class="row-wrap">
          <Segmented v-model="mode" :options="modes" :label="msg.search.title" />
          <UiButton v-if="mode === 'trackers'" type="submit" variant="primary" icon="search" :disabled="!query.trim()">
            {{ msg.releases.search }}
          </UiButton>
        </div>
      </form>

      <template v-if="mode === 'catalog'">
        <ErrorBanner v-if="media.error.value" :error="media.error.value" retry @retry="submit" />
        <div v-else-if="media.loading.value && !media.data.value" class="skeleton-grid">
          <Skeleton v-for="i in 12" :key="i" ratio="2 / 3" />
        </div>
        <template v-else-if="media.data.value">
          <EmptyState v-if="!media.data.value.length" icon="search" :text="msg.common.nothingFound" />
          <MediaGrid v-else :items="media.data.value" />
        </template>
        <EmptyState v-else icon="film" :text="tmdbOff ? msg.errors.tmdb : msg.search.hintCatalog" />
      </template>

      <template v-else>
        <ReleaseList
          v-if="trackersQuery"
          :response="releases.data.value"
          :loading="releases.loading.value"
          :error="releases.error.value"
          :busy-key="busyKey"
          @select="(r) => add(r, null)"
          @retry="releases.run(trackersQuery)"
        />
        <EmptyState v-else icon="radar" :text="msg.search.hintTrackers" />
      </template>
    </div>
  </div>
</template>

<style scoped>
.search {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.search__field {
  position: relative;
  display: flex;
  align-items: center;
}
.search__icon {
  position: absolute;
  left: 14px;
  color: var(--c-muted);
  pointer-events: none;
}
.search__input {
  height: 46px;
  padding-left: 42px;
  padding-right: 44px;
  font-size: 15px;
  background: var(--c-surface);
}
.search__input::-webkit-search-cancel-button {
  display: none;
}
.search__clear {
  position: absolute;
  right: 1px;
}
.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(clamp(104px, 28vw, 150px), 1fr));
  gap: 18px 14px;
}
</style>
