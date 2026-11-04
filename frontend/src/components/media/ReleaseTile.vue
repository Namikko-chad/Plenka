<script setup lang="ts">
import { computed } from 'vue'
import type { Release } from '@/api/types'
import Badge from '@/components/ui/Badge.vue'
import Spinner from '@/components/ui/Spinner.vue'
import { msg } from '@/i18n'
import { formatBytes, formatDate } from '@/utils/format'
import { isHdr, qualityOf, seedTone } from '@/utils/releases'

const props = defineProps<{ release: Release; busy?: boolean; disabled?: boolean }>()
defineEmits<{ select: [] }>()

const quality = computed(() => qualityOf(props.release.title))
const hdr = computed(() => isHdr(props.release.title))
const qualityTone = computed(() => (quality.value === '4K' ? 'purple' : quality.value === 'SD' ? 'neutral' : 'accent'))
</script>

<template>
  <button type="button" class="rtile" :disabled="disabled" :aria-busy="busy || undefined" @click="$emit('select')">
    <span class="rtile__title clamp-2">{{ release.title }}</span>
    <span class="rtile__meta">
      <Badge :tone="qualityTone">{{ quality }}</Badge>
      <Badge v-if="hdr" tone="warning">{{ msg.releases.hdr }}</Badge>
      <span class="rtile__tracker">{{ release.tracker }}</span>
      <span>{{ formatBytes(release.size) }}</span>
      <span v-if="release.publishDate" class="muted">{{ formatDate(release.publishDate) }}</span>
      <span class="spacer" />
      <span class="rtile__seeds" :class="`rtile__seeds--${seedTone(release.seeders)}`" :title="msg.releases.seeders">
        ▲ {{ release.seeders }}
      </span>
      <span class="muted" :title="msg.releases.peers">▼ {{ release.peers }}</span>
      <Spinner v-if="busy" :size="14" />
    </span>
  </button>
</template>

<style scoped>
.rtile {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  padding: 12px 14px;
  text-align: left;
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-radius: var(--r-card);
  color: var(--c-text);
  transition:
    border-color var(--dur),
    background var(--dur);
}
.rtile:hover:not(:disabled),
.rtile:focus-visible {
  border-color: rgba(var(--rgb-accent), var(--a-line));
  background: rgba(var(--rgb-accent), 0.03);
}
.rtile:focus-visible {
  outline: none;
  border-color: var(--c-accent);
}
.rtile:disabled {
  opacity: 0.6;
  cursor: default;
}
.rtile__title {
  font-size: 13px;
  line-height: 1.4;
}
.rtile__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  font-size: 11px;
}
.rtile__tracker {
  color: var(--c-accent);
}
.rtile__seeds {
  font-weight: 700;
}
.rtile__seeds--ok {
  color: var(--c-ok);
}
.rtile__seeds--warning {
  color: var(--c-warning);
}
.rtile__seeds--error {
  color: var(--c-error);
}
</style>
