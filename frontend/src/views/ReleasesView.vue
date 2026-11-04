<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { searchReleases } from '@/api/search'
import type { MediaDetails, MediaType, SearchResponse } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import ReleaseList from '@/components/media/ReleaseList.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { useAddRelease } from '@/composables/useAddRelease'
import { useAsync } from '@/composables/useAsync'
import { msg } from '@/i18n'
import { useCatalog } from '@/stores/catalog'
import { useServer } from '@/stores/server'
import { buildQueries, mergeSearch } from '@/utils/releases'

const props = defineProps<{ type: MediaType; id: number }>()
const catalog = useCatalog()
const server = useServer()
const { busyKey, add } = useAddRelease()

const media = useAsync<MediaDetails>(() => catalog.getDetails(props.type, props.id))
const query = ref('')

/** Several queries in parallel; a failure of one doesn't hide the others. */
const search = useAsync<SearchResponse, [string[]]>(async (signal, queries) => {
  const timeoutSec = server.config?.searchTimeoutSec
  const settled = await Promise.allSettled(queries.map((q) => searchReleases(q, { signal, timeoutSec })))
  const ok = settled.flatMap((s) => (s.status === 'fulfilled' ? [s.value] : []))
  if (!ok.length) throw (settled[0] as PromiseRejectedResult).reason
  return mergeSearch(ok)
})

const subtitle = computed(() => {
  const m = media.data.value
  return m ? [m.title, m.year].filter(Boolean).join(' · ') : ''
})

function runDefault() {
  const m = media.data.value
  if (!m) return
  const qs = buildQueries(m)
  query.value = qs[0] ?? ''
  void search.run(qs)
}

function runCustom() {
  const q = query.value.trim()
  if (q) void search.run([q])
}

onMounted(async () => {
  await media.run()
  runDefault()
})
</script>

<template>
  <div class="page">
    <AppBar :title="msg.releases.title" :subtitle="subtitle" :back="`/${type}/${id}`" />
    <div class="page-body">
      <ErrorBanner v-if="media.error.value" :error="media.error.value" retry @retry="media.run().then(runDefault)" />

      <form class="rq" @submit.prevent="runCustom">
        <label class="sr-only" for="rq-input">{{ msg.releases.query }}</label>
        <input
          id="rq-input"
          v-model="query"
          class="input"
          type="search"
          enterkeyhint="search"
          :placeholder="msg.releases.query"
        />
        <UiButton type="submit" icon="search" :disabled="!query.trim() || search.loading.value">{{ msg.releases.search }}</UiButton>
      </form>

      <ReleaseList
        :response="search.data.value"
        :loading="search.loading.value || media.loading.value"
        :error="search.error.value"
        :busy-key="busyKey"
        @select="(r) => add(r, media.data.value)"
        @retry="runDefault"
      />
    </div>
  </div>
</template>

<style scoped>
.rq {
  display: flex;
  gap: 8px;
  align-items: center;
}
.rq .input {
  flex: 1;
  min-width: 0;
  height: var(--control-h);
  min-height: 0;
  padding: 0 12px;
}
</style>
