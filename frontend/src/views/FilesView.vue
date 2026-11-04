<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { torrentsApi } from '@/api/torrents'
import type { Media, TorrentFile, Torrent } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Icon from '@/components/ui/Icon.vue'
import IconButton from '@/components/ui/IconButton.vue'
import ProgressLine from '@/components/ui/ProgressLine.vue'
import Spinner from '@/components/ui/Spinner.vue'
import { usePolling } from '@/composables/usePolling'
import { msg } from '@/i18n'
import { useHistory } from '@/stores/history'
import { usePlayback, type PlayTarget } from '@/stores/playback'
import { useToast } from '@/stores/toast'
import { reviveFromHistory, useTorrents } from '@/stores/torrents'
import { errorText } from '@/utils/errors'
import { fileName, playableFiles, sortedFiles } from '@/utils/files'
import { formatBytes, formatSpeed } from '@/utils/format'
import { isFinished, progressOf } from '@/utils/history'
import { mediaFromTorrentData } from '@/utils/media'

/**
 * Files of a torrent. Opened to play (from a release or "Continue watching") it starts the
 * only video by itself; with `browse` (the Torrents tab) it lists every file and
 * starts nothing.
 */
const props = defineProps<{ hash: string; resume?: number; browse?: boolean }>()
const history = useHistory()
const playback = usePlayback()
const torrents = useTorrents()
const toast = useToast()

const stage = ref<string>(msg.files.connecting)
const status = ref<Torrent | null>(null)
const files = ref<TorrentFile[] | null>(null)
/** Ids of the files the player can start. */
const playable = ref<Set<number>>(new Set())
const error = ref<unknown>(null)
const list = ref<HTMLElement | null>(null)
let ctrl: AbortController | null = null

const fromHistory = computed(() => history.entries.find((e) => e.hash === props.hash) ?? null)
const media = computed<Media | null>(() => mediaFromTorrentData(status.value?.data) ?? fromHistory.value?.media ?? null)
const torrentTitle = computed(() => status.value?.title || fromHistory.value?.torrentTitle || status.value?.name || '')
const title = computed(() => media.value?.title || torrentTitle.value || msg.files.title)

const statusLine = computed(() => {
  const s = status.value
  if (!s) return ''
  const parts = [s.statString, msg.files.peers(s.activePeers, s.totalPeers)]
  if (s.downloadSpeed) parts.push(formatSpeed(s.downloadSpeed))
  if (s.stat === 2 && s.preloadSize > 0) parts.push(msg.files.preloading(formatBytes(s.preloadedBytes), formatBytes(s.preloadSize)))
  return parts.filter(Boolean).join(' · ')
})

const preloadProgress = computed(() => {
  const s = status.value
  return s && s.stat === 2 && s.preloadSize > 0 ? Math.min(1, s.preloadedBytes / s.preloadSize) : null
})

const lastWatched = computed(() => {
  let best: { id: number; t: number } | null = null
  for (const f of files.value ?? []) {
    const e = history.entry(props.hash, f.id)
    const t = e ? Date.parse(e.updatedAt) : 0
    if (e && (!best || t > best.t)) best = { id: f.id, t }
  }
  return best?.id ?? null
})

const rows = computed(() => {
  const focusId = lastWatched.value ?? files.value?.find((f) => playable.value.has(f.id))?.id ?? files.value?.[0]?.id
  return (files.value ?? []).map((f) => {
    const e = history.entry(props.hash, f.id)
    return {
      file: f,
      name: fileName(f.path),
      playable: playable.value.has(f.id),
      done: !!e && isFinished(e),
      progress: e ? progressOf(e) : 0,
      focus: f.id === focusId,
    }
  })
})

async function load() {
  ctrl?.abort()
  const c = new AbortController()
  ctrl = c
  error.value = null
  files.value = null
  stage.value = msg.files.connecting
  try {
    await history.ensure()
    const st = await torrents.waitForFiles(props.hash, {
      signal: c.signal,
      revive: reviveFromHistory(props.hash, fromHistory.value),
      onStatus: (s) => {
        status.value = s
        stage.value = msg.files.waitingFiles
      },
    })
    if (!st) throw new Error(msg.files.noPeers)
    const videos = playableFiles(st.files)
    playable.value = new Set(videos.map((f) => f.id))
    files.value = props.browse ? sortedFiles(st.files) : videos
    if (!props.browse) autoplay(videos)
    await nextTick()
    list.value?.querySelector<HTMLElement>('[data-focus="1"]')?.focus({ preventScroll: false })
  } catch (e) {
    if (c.signal.aborted) return
    error.value = e
  }
}

function target(f: TorrentFile): PlayTarget {
  return { hash: props.hash, fileId: f.id, fileName: fileName(f.path), torrentTitle: torrentTitle.value, media: media.value }
}

function autoplay(list: TorrentFile[]) {
  const resumed = props.resume != null ? list.find((f) => f.id === props.resume) : undefined
  const f = resumed ?? (list.length === 1 ? list[0] : undefined)
  if (f) void playback.start(target(f), { replace: true })
}

function play(f: TorrentFile) {
  if (playable.value.has(f.id)) void playback.start(target(f))
}

async function preload(f: TorrentFile) {
  try {
    await torrents.preload(props.hash, f.id)
    toast.ok(msg.files.preloadStarted)
  } catch (e) {
    toast.error(errorText(e))
  }
}

// Keeps the peers / preload line fresh once the list is shown.
const listed = computed(() => files.value !== null)
usePolling(
  async () => {
    status.value = await torrentsApi.get(props.hash)
  },
  2000,
  listed,
)

onMounted(load)
onBeforeUnmount(() => ctrl?.abort())
</script>

<template>
  <div class="page">
    <AppBar :title="title" :subtitle="files ? torrentTitle : ''" back />
    <div class="page-body">
      <ErrorBanner v-if="error" :error="error" retry @retry="load" />

      <div v-else-if="!files" class="files__wait">
        <Spinner :size="40" />
        <p class="files__stage">{{ stage }}</p>
        <p v-if="torrentTitle" class="muted files__name">{{ torrentTitle }}</p>
        <p v-if="statusLine" class="small">{{ statusLine }}</p>
      </div>

      <template v-else>
        <p v-if="statusLine" class="small">{{ statusLine }}</p>
        <ProgressLine v-if="preloadProgress !== null" :value="preloadProgress" tone="warning" />
        <EmptyState v-if="!files.length" icon="folder" :text="msg.files.empty" />
        <ul v-else ref="list" class="files">
          <li v-for="r in rows" :key="r.file.id" class="files__row">
            <button
              type="button"
              class="file"
              :class="{ 'file--other': !r.playable }"
              :data-focus="r.focus ? '1' : undefined"
              :aria-disabled="!r.playable || undefined"
              :title="r.playable ? undefined : msg.files.notPlayable"
              @click="play(r.file)"
            >
              <Icon
                :name="!r.playable ? 'file' : r.done ? 'checkCircle' : 'playCircle'"
                :size="20"
                class="file__icon"
                :class="{ 'file__icon--done': r.done }"
              />
              <span class="file__main">
                <span class="file__name clamp-2">{{ r.name }}</span>
                <ProgressLine v-if="!r.done && r.progress > 0" :value="r.progress" />
              </span>
              <span class="file__size">{{ formatBytes(r.file.length) }}</span>
            </button>
            <IconButton
              v-if="r.playable"
              icon="preload"
              class="file__preload"
              :label="`${msg.files.preload}: ${r.name}`"
              @click="preload(r.file)"
            />
          </li>
        </ul>
      </template>
    </div>
  </div>
</template>

<style scoped>
.files__wait {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 12px;
  padding: 64px 16px;
}
.files__stage {
  margin: 0;
  font-size: 14px;
}
.files__name {
  margin: 0;
  max-width: 520px;
  overflow-wrap: anywhere;
}
.files {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.file {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-height: 56px;
  padding: 10px 14px;
  text-align: left;
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-radius: var(--r-card);
  color: var(--c-text);
  transition: border-color var(--dur);
}
.files__row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.file--other {
  color: var(--c-muted);
  cursor: default;
}
.file:hover,
.file:focus-visible {
  border-color: var(--c-accent);
  outline: none;
}
.file__icon {
  color: var(--c-muted);
}
.file--other:hover {
  border-color: var(--c-border);
}
.file__preload {
  flex-shrink: 0;
}
.file__icon--done {
  color: var(--c-ok);
}
.file__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.file__name {
  overflow-wrap: anywhere;
}
.file__size {
  font-size: 11px;
  color: var(--c-muted);
  white-space: nowrap;
}
</style>
