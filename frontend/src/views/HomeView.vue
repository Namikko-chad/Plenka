<script setup lang="ts">
import { computed, onActivated, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { HistoryEntry } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import BackdropCard from '@/components/media/BackdropCard.vue'
import MediaRow from '@/components/media/MediaRow.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import IconButton from '@/components/ui/IconButton.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { msg } from '@/i18n'
import { useCatalog } from '@/stores/catalog'
import { useDialog } from '@/stores/dialog'
import { useHistory } from '@/stores/history'
import { useServer } from '@/stores/server'
import { useToast } from '@/stores/toast'
import { errorText } from '@/utils/errors'

defineOptions({ name: 'HomeView' })

const server = useServer()
const catalog = useCatalog()
const history = useHistory()
const dialog = useDialog()
const toast = useToast()
const router = useRouter()

const tmdbOff = computed(() => server.info !== null && !server.info.tmdb)
const allFailed = computed(() => catalog.homeLoaded && catalog.rows.every((r) => r.state === 'error'))
const visibleRows = computed(() => catalog.rows.filter((r) => r.state === 'loading' || (r.state === 'ready' && r.items.length)))

async function load(force = false) {
  const info = await server.loadInfo(force)
  if (info?.tmdb && (force || !catalog.homeLoaded)) await catalog.loadHome()
}

onMounted(() => void load())
// the key was set (or changed) in Settings: reload the catalog
watch(
  () => server.config?.tmdbApiKeySet,
  (set, prev) => {
    if (set && prev !== undefined && set !== prev) void catalog.loadHome()
  },
)
watch(
  () => server.info?.tmdb,
  (on) => {
    if (on && !catalog.homeLoaded) void catalog.loadHome()
  },
)
// returning to the tab refreshes «continue watching» (the player may have saved progress)
onActivated(() => void history.load())

function refresh() {
  void history.load()
  void load(true)
}

function open(e: HistoryEntry) {
  void router.push({ name: 'files', params: { hash: e.hash }, query: { resume: String(e.fileId) } })
}

async function remove(e: HistoryEntry) {
  const title = e.media?.title || e.torrentTitle || e.fileName
  const ok = await dialog.confirm({ title: msg.home.removeTitle, text: title, confirmLabel: msg.home.remove, danger: true })
  if (!ok) return
  try {
    await history.removeGroup(e)
  } catch (err) {
    toast.error(errorText(err))
  }
}
</script>

<template>
  <div class="page">
    <AppBar :title="msg.app.name" large>
      <IconButton icon="refresh" :label="msg.common.retry" @click="refresh" />
    </AppBar>

    <div class="home">
      <div v-if="server.infoError && !server.info" class="home__pad">
        <ErrorBanner :error="server.infoError" retry @retry="refresh" />
      </div>

      <section v-if="history.continueList.length" class="home__continue" :aria-label="msg.home.continue">
        <h2 class="section-title home__pad">{{ msg.home.continue }}</h2>
        <div class="home__scroll">
          <BackdropCard
            v-for="e in history.continueList"
            :key="`${e.hash}:${e.fileId}`"
            :entry="e"
            @open="open(e)"
            @remove="remove(e)"
          />
        </div>
      </section>

      <EmptyState v-if="tmdbOff" icon="film" :title="msg.home.noTmdbTitle" :text="msg.home.noTmdb">
        <UiButton variant="primary" to="/prefs/catalog" icon="settings">{{ msg.home.toSettings }}</UiButton>
        <UiButton :to="{ path: '/search', query: { mode: 'trackers' } }" icon="radar">{{ msg.home.toTrackers }}</UiButton>
      </EmptyState>

      <template v-else-if="server.info">
        <div v-if="allFailed" class="home__pad">
          <ErrorBanner :error="catalog.lastError ?? msg.home.allFailed" retry @retry="load(true)" />
        </div>
        <MediaRow
          v-for="row in visibleRows"
          :key="row.id"
          :title="row.title"
          :items="row.items"
          :loading="row.state === 'loading'"
        />
      </template>
    </div>
  </div>
</template>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 14px 0 32px;
}
.home__pad {
  padding: 0 var(--gutter);
}
.home__scroll {
  display: flex;
  gap: 14px;
  overflow-x: auto;
  padding: 12px var(--gutter) 6px;
  scrollbar-width: none;
  scroll-snap-type: x proximity;
  scroll-padding: 0 var(--gutter);
}
.home__scroll > * {
  flex: 0 0 auto;
  scroll-snap-align: start;
}
.home__scroll::-webkit-scrollbar {
  display: none;
}
</style>
