<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError } from '@/api/http'
import { torrentsApi } from '@/api/torrents'
import type { CacheState, Torrent } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import PieceMap from '@/components/media/PieceMap.vue'
import TorrentCard from '@/components/media/TorrentCard.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import IconButton from '@/components/ui/IconButton.vue'
import Sheet from '@/components/ui/Sheet.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { useClipboard } from '@/composables/useClipboard'
import { usePolling } from '@/composables/usePolling'
import { msg } from '@/i18n'
import { useDialog } from '@/stores/dialog'
import { useHistory } from '@/stores/history'
import { useToast } from '@/stores/toast'
import { useTorrents } from '@/stores/torrents'
import { errorText } from '@/utils/errors'
import { playableFiles } from '@/utils/files'
import { formatBytes, formatDateTime, formatSpeed } from '@/utils/format'

defineOptions({ name: 'TorrentsView' })

const torrents = useTorrents()
const dialog = useDialog()
const toast = useToast()
const router = useRouter()
const copy = useClipboard()
const history = useHistory()

usePolling(() => torrents.refresh(), 3000)
void history.ensure()

// ---------- add ----------
const addOpen = ref(false)
const addLink = ref('')
const addTitle = ref('')
const addFile = ref<File | null>(null)
const adding = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function openAdd() {
  addLink.value = ''
  addTitle.value = ''
  addFile.value = null
  addOpen.value = true
}

function onFile(e: Event) {
  addFile.value = (e.target as HTMLInputElement).files?.[0] ?? null
}

async function submitAdd() {
  if (adding.value || (!addLink.value.trim() && !addFile.value)) return
  adding.value = true
  try {
    if (addFile.value) await torrents.upload(addFile.value, addTitle.value.trim() || undefined)
    else await torrents.add({ link: addLink.value.trim(), title: addTitle.value.trim() })
    toast.ok(msg.torrents.added)
    addOpen.value = false
  } catch (e) {
    toast.error(errorText(e))
  } finally {
    adding.value = false
  }
}

// ---------- actions ----------
async function drop(t: Torrent) {
  try {
    await torrents.drop(t.hash)
    toast.ok(msg.torrents.dropped)
  } catch (e) {
    toast.error(errorText(e))
  }
}

async function remove(t: Torrent) {
  const ok = await dialog.confirm({
    title: msg.torrents.remove,
    text: msg.torrents.removeConfirm(t.title || t.name || t.hash),
    confirmLabel: msg.common.delete,
    danger: true,
  })
  if (!ok) return
  try {
    await torrents.remove(t.hash)
    if (selectedHash.value === t.hash) selectedHash.value = null
    toast.ok(msg.torrents.removed)
  } catch (e) {
    toast.error(errorText(e))
  }
}

// The Torrents tab only shows the files; nothing starts playing by itself.
function openFiles(t: Torrent) {
  void router.push({ name: 'files', params: { hash: t.hash }, query: { browse: '1' } })
}

/** The file worth preloading: the last watched one, else the first video. */
function preloadTarget(t: Torrent): number | null {
  const watched = history.entries.find((e) => e.hash === t.hash && t.files.some((f) => f.id === e.fileId))
  return watched?.fileId ?? playableFiles(t.files)[0]?.id ?? null
}

const preloading = ref(false)
async function preload(t: Torrent) {
  const id = preloadTarget(t)
  if (id === null || preloading.value) return
  preloading.value = true
  try {
    await torrents.preload(t.hash, id)
    toast.ok(msg.torrents.preloadStarted)
  } catch (e) {
    toast.error(errorText(e))
  } finally {
    preloading.value = false
  }
}

const fullBusy = ref(false)
async function toggleFullDownload(t: Torrent) {
  if (fullBusy.value) return
  fullBusy.value = true
  try {
    if (t.fullDownload) {
      await torrents.cancelDownload(t.hash)
      toast.ok(msg.torrents.downloadAllCancelled)
    } else {
      await torrents.downloadAll(t.hash)
      toast.ok(msg.torrents.downloadAllStarted)
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 507 ? msg.torrents.downloadAllNoRoom : errorText(e))
  } finally {
    fullBusy.value = false
  }
}

// ---------- details ----------
const selectedHash = ref<string | null>(null)
const cache = ref<CacheState | null>(null)
const detailsOpen = computed({
  get: () => selectedHash.value !== null,
  set: (v) => {
    if (!v) selectedHash.value = null
  },
})
const selected = computed(() => torrents.list.find((t) => t.hash === selectedHash.value) ?? null)

/** Closes the details sheet, then runs a footer action on its torrent. */
function closeDetails(action: () => unknown) {
  action()
  selectedHash.value = null
}

function openDetails(t: Torrent) {
  cache.value = null
  selectedHash.value = t.hash
}

usePolling(
  async () => {
    const h = selectedHash.value
    if (!h) return
    const c = await torrentsApi.cache(h)
    if (selectedHash.value === h) cache.value = c
  },
  2000,
  detailsOpen,
)

const stats = computed(() => {
  const t = selected.value
  if (!t) return []
  const rows: [string, string][] = [
    [msg.torrents.size, formatBytes(t.torrentSize)],
    [msg.torrents.loaded, formatBytes(t.loadedSize || t.bytesRead)],
    [msg.torrents.speedDown, formatSpeed(t.downloadSpeed)],
    [msg.torrents.speedUp, formatSpeed(t.uploadSpeed)],
    [msg.torrents.peers, `${t.activePeers} / ${t.totalPeers}`],
    [msg.torrents.seeders, String(t.connectedSeeders)],
  ]
  if (t.fullDownload && t.torrentSize)
    rows.push([msg.torrents.fullDownload, `${formatBytes(t.loadedSize)} / ${formatBytes(t.torrentSize)} · ${Math.floor((t.loadedSize / t.torrentSize) * 100)}%`])
  if (t.preloadSize) rows.push([msg.torrents.preload, `${formatBytes(t.preloadedBytes)} / ${formatBytes(t.preloadSize)}`])
  if (t.addedAt) rows.push([msg.torrents.addedAt, formatDateTime(t.addedAt)])
  return rows
})
</script>

<template>
  <div class="page">
    <AppBar :title="msg.torrents.title" large>
      <IconButton icon="plus" :label="msg.torrents.addTitle" tone="accent" @click="openAdd" />
    </AppBar>

    <div class="page-body">
      <ErrorBanner v-if="torrents.error" :error="torrents.error" retry @retry="torrents.refresh()" />

      <div v-if="!torrents.loaded && !torrents.error" class="stack">
        <Skeleton v-for="i in 4" :key="i" height="100px" />
      </div>

      <EmptyState v-else-if="torrents.loaded && !torrents.list.length" icon="download" :text="msg.torrents.empty">
        <UiButton variant="primary" icon="plus" @click="openAdd">{{ msg.torrents.add }}</UiButton>
      </EmptyState>

      <div v-else class="stack">
        <TorrentCard
          v-for="t in torrents.list"
          :key="t.hash"
          :torrent="t"
          @details="openDetails(t)"
          @files="openFiles(t)"
          @drop="drop(t)"
          @remove="remove(t)"
        />
      </div>
    </div>

    <Sheet v-model="addOpen" :title="msg.torrents.addTitle">
      <form id="add-form" class="stack" @submit.prevent="submitAdd">
        <div class="field">
          <label class="label" for="add-link">{{ msg.torrents.link }}</label>
          <input
            id="add-link"
            v-model="addLink"
            class="input"
            :placeholder="msg.torrents.linkPh"
            autocomplete="off"
            :disabled="!!addFile"
            autofocus
          />
        </div>
        <div class="field">
          <span class="label">{{ msg.torrents.file }}</span>
          <div class="row">
            <UiButton icon="upload" @click="fileInput?.click()">{{ msg.torrents.chooseFile }}</UiButton>
            <span class="small ellipsis">{{ addFile?.name }}</span>
            <IconButton v-if="addFile" icon="close" :label="msg.common.cancel" :size="16" @click="addFile = null" />
          </div>
          <input ref="fileInput" class="sr-only" type="file" accept=".torrent,application/x-bittorrent" tabindex="-1" @change="onFile" />
        </div>
        <div class="field">
          <label class="label" for="add-title">{{ msg.torrents.titleField }}</label>
          <input id="add-title" v-model="addTitle" class="input" autocomplete="off" />
        </div>
      </form>
      <template #footer>
        <UiButton variant="neutral" @click="addOpen = false">{{ msg.common.cancel }}</UiButton>
        <UiButton
          variant="primary"
          type="submit"
          form="add-form"
          :loading="adding"
          :disabled="!addLink.trim() && !addFile"
        >
          {{ msg.torrents.addSubmit }}
        </UiButton>
      </template>
    </Sheet>

    <Sheet v-model="detailsOpen" :title="selected?.title || selected?.name || msg.torrents.details">
      <div v-if="selected" class="stack">
        <dl class="kv">
          <template v-for="[k, v] in stats" :key="k">
            <dt>{{ k }}</dt>
            <dd>{{ v }}</dd>
          </template>
          <dt>{{ msg.torrents.hash }}</dt>
          <dd>
            <button type="button" class="kv__hash" @click="copy(selected.hash)">{{ selected.hash }}</button>
          </dd>
        </dl>

        <h3 class="section-title">{{ msg.torrents.cache }}</h3>
        <template v-if="cache && cache.piecesCount > 0">
          <p class="small">
            {{ msg.torrents.filled }} {{ formatBytes(cache.filled) }} / {{ formatBytes(cache.capacity) }} ·
            {{ msg.torrents.pieces(cache.piecesCount) }} · {{ msg.torrents.pieceSize }} {{ formatBytes(cache.piecesLength) }}
          </p>
          <PieceMap :cache="cache" />
          <div class="legend small">
            <span><i class="legend__sw legend__sw--done" />{{ msg.torrents.legendDone }}</span>
            <span><i class="legend__sw legend__sw--partial" />{{ msg.torrents.legendPartial }}</span>
            <span><i class="legend__sw legend__sw--prio" />{{ msg.torrents.legendPriority }}</span>
            <span><i class="legend__sw legend__sw--reader" />{{ msg.torrents.legendReader }}</span>
          </div>
        </template>
        <p v-else-if="cache" class="small">{{ msg.torrents.noCache }}</p>
        <Skeleton v-else height="60px" />
      </div>
      <template #footer>
        <UiButton
          v-if="selected && preloadTarget(selected) !== null"
          variant="neutral"
          icon="preload"
          :loading="preloading || selected.stat === 2"
          @click="closeDetails(() => preload(selected!))"
        >
          {{ msg.torrents.preloadAction }}
        </UiButton>
        <UiButton
          v-if="selected"
          :variant="selected.fullDownload ? 'red' : 'neutral'"
          :icon="selected.fullDownload ? 'close' : 'download'"
          :title="selected.fullDownload ? undefined : msg.torrents.downloadAllHint"
          :loading="fullBusy"
          @click="closeDetails(() => toggleFullDownload(selected!))"
        >
          {{ selected.fullDownload ? msg.torrents.downloadAllCancel : msg.torrents.downloadAll }}
        </UiButton>
        <UiButton v-if="selected" icon="folder" @click="closeDetails(() => openFiles(selected!))">{{ msg.torrents.files }}</UiButton>
        <UiButton v-if="selected" variant="red" icon="trash" @click="remove(selected)">{{ msg.torrents.remove }}</UiButton>
      </template>
    </Sheet>
  </div>
</template>

<style scoped>
.kv {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px 16px;
  margin: 0;
  font-size: 12px;
}
.kv dt {
  color: var(--c-muted);
}
.kv dd {
  margin: 0;
  text-align: right;
  overflow-wrap: anywhere;
}
.kv__hash {
  border: none;
  background: none;
  padding: 0;
  color: var(--c-accent);
  font-size: 11px;
  text-align: right;
  overflow-wrap: anywhere;
}
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
}
.legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.legend__sw {
  width: 10px;
  height: 10px;
  border-radius: 2px;
  display: inline-block;
}
.legend__sw--done {
  background: var(--c-accent);
}
.legend__sw--partial {
  background: rgba(var(--rgb-accent), 0.45);
}
.legend__sw--prio {
  background: var(--c-warning);
}
.legend__sw--reader {
  border: 1px solid var(--c-purple-l);
}
</style>
