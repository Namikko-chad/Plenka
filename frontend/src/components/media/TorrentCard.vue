<script setup lang="ts">
import { computed } from 'vue'
import type { Torrent } from '@/api/types'
import Badge from '@/components/ui/Badge.vue'
import IconButton from '@/components/ui/IconButton.vue'
import ProgressLine from '@/components/ui/ProgressLine.vue'
import { msg } from '@/i18n'
import { formatBytes, formatSpeed } from '@/utils/format'
import type { Tone } from '@/utils/media'
import NetImage from './NetImage.vue'

const props = defineProps<{ torrent: Torrent }>()
defineEmits<{ details: []; files: []; drop: []; remove: [] }>()

const t = computed(() => props.torrent)
const title = computed(() => t.value.title || t.value.name || t.value.hash)
const tone = computed<Tone>(() => {
  switch (t.value.stat) {
    case 3:
      return 'ok'
    case 2:
      return 'warning'
    case 1:
    case 0:
      return 'active'
    default:
      return 'neutral'
  }
})
const active = computed(() => t.value.stat >= 1 && t.value.stat <= 3)
const full = computed(() => {
  const size = t.value.torrentSize
  return t.value.fullDownload && size > 0 ? Math.min(1, t.value.loadedSize / size) : null
})
const preload = computed(() => {
  const size = t.value.preloadSize
  return t.value.stat === 2 && size > 0 ? t.value.preloadedBytes / size : null
})
</script>

<template>
  <article class="tcard card">
    <button type="button" class="tcard__main" :aria-label="`${msg.torrents.details}: ${title}`" @click="$emit('details')">
      <div class="tcard__poster"><NetImage :src="t.poster || null" :alt="title" /></div>
      <div class="tcard__body">
        <span class="tcard__title clamp-2">{{ title }}</span>
        <span class="tcard__meta">
          <Badge :tone="tone">{{ t.statString || '—' }}</Badge>
          <span v-if="t.torrentSize">{{ formatBytes(t.torrentSize) }}</span>
          <Badge v-if="full !== null" :tone="full >= 1 ? 'ok' : 'active'">
            {{ full >= 1 ? msg.torrents.fullDownloadDone : `${msg.torrents.fullDownload} ${Math.floor(full * 100)}%` }}
          </Badge>
        </span>
        <span v-if="active" class="tcard__meta small">
          <span>↓ {{ formatSpeed(t.downloadSpeed) }}</span>
          <span>{{ msg.torrents.peers.toLowerCase() }} {{ t.activePeers }}/{{ t.totalPeers }}</span>
          <span v-if="t.connectedSeeders">{{ msg.torrents.seeders.toLowerCase() }} {{ t.connectedSeeders }}</span>
        </span>
        <ProgressLine v-if="preload !== null" :value="preload" tone="warning" />
        <ProgressLine v-else-if="full !== null && full < 1" :value="full" />
      </div>
    </button>
    <div class="tcard__actions">
      <IconButton icon="folder" :label="msg.torrents.files" @click="$emit('files')" />
      <IconButton v-if="active" icon="eject" :label="msg.torrents.dropHint" @click="$emit('drop')" />
      <IconButton icon="trash" :label="msg.torrents.remove" tone="danger" @click="$emit('remove')" />
    </div>
  </article>
</template>

<style scoped>
.tcard {
  display: flex;
  align-items: stretch;
}
.tcard__main {
  flex: 1;
  min-width: 0;
  display: flex;
  gap: 12px;
  padding: 10px;
  border: none;
  background: none;
  color: var(--c-text);
  text-align: left;
  border-radius: var(--r-card) 0 0 var(--r-card);
}
.tcard__main:hover {
  background: rgba(var(--rgb-accent), 0.03);
}
.tcard__poster {
  width: 54px;
  flex-shrink: 0;
  aspect-ratio: 2 / 3;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--c-border);
  align-self: flex-start;
}
.tcard__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.tcard__title {
  font-size: 13px;
  line-height: 1.35;
  overflow-wrap: anywhere;
}
.tcard__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 10px;
  font-size: 11px;
}
.tcard__actions {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 4px;
  border-left: 1px solid var(--c-border);
}
@media (min-width: 600px) {
  .tcard__actions {
    flex-direction: row;
    align-items: center;
  }
}
</style>
