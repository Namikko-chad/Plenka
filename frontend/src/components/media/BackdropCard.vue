<script setup lang="ts">
import { computed } from 'vue'
import type { HistoryEntry } from '@/api/types'
import ProgressLine from '@/components/ui/ProgressLine.vue'
import Icon from '@/components/ui/Icon.vue'
import { msg } from '@/i18n'
import { progressOf } from '@/utils/history'
import { imageUrl } from '@/utils/media'
import NetImage from './NetImage.vue'

const props = defineProps<{ entry: HistoryEntry }>()
const emit = defineEmits<{ open: []; remove: [] }>()

const title = computed(() => props.entry.media?.title || props.entry.torrentTitle || props.entry.fileName)
const image = computed(() => {
  const m = props.entry.media
  return imageUrl(m?.backdropPath, 'w780') ?? imageUrl(m?.posterPath, 'w780')
})
</script>

<template>
  <div class="bcard">
    <button type="button" class="bcard__main" :aria-label="`${title}: ${entry.fileName}`" @click="emit('open')" @contextmenu.prevent="emit('remove')">
      <div class="bcard__frame">
        <NetImage :src="image" :alt="title" fallback="playCircle" />
        <div class="bcard__shade" />
        <Icon name="playCircle" :size="36" class="bcard__play" />
        <div class="bcard__progress"><ProgressLine :value="progressOf(entry)" /></div>
      </div>
      <span class="bcard__title ellipsis">{{ title }}</span>
      <span class="bcard__file ellipsis">{{ entry.fileName }}</span>
    </button>
    <button type="button" class="bcard__remove" :aria-label="msg.home.removeAria(title)" :title="msg.home.remove" @click="emit('remove')">
      <Icon name="close" :size="14" />
    </button>
  </div>
</template>

<style scoped>
.bcard {
  position: relative;
  width: clamp(220px, 62vw, 300px);
}
.bcard__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  padding: 0;
  border: none;
  background: none;
  text-align: left;
  color: var(--c-text);
  border-radius: var(--r-card);
}
.bcard__main:focus-visible {
  outline: none;
}
.bcard__frame {
  position: relative;
  aspect-ratio: 16 / 9;
  border-radius: var(--r-card);
  overflow: hidden;
  border: 1px solid var(--c-border);
  margin-bottom: 6px;
  transition:
    transform 0.18s,
    border-color 0.18s;
}
.bcard__main:hover .bcard__frame,
.bcard__main:focus-visible .bcard__frame {
  transform: scale(1.03);
  border-color: var(--c-accent);
}
.bcard__shade {
  position: absolute;
  inset: 0;
  background: linear-gradient(transparent 50%, rgba(var(--rgb-bg), 0.85));
}
.bcard__play {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  color: var(--c-text);
  opacity: 0.85;
}
.bcard__progress {
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: 10px;
}
.bcard__title {
  font-size: 12px;
  font-weight: 600;
}
.bcard__file {
  font-size: 11px;
  color: var(--c-muted);
}
.bcard__remove {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: none;
  padding: 0;
  color: var(--c-text);
}
.bcard__remove::before {
  content: '';
  position: absolute;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(var(--rgb-bg), 0.75);
  border: 1px solid var(--c-border);
  z-index: -1;
}
.bcard__remove:hover {
  color: var(--c-error);
}
</style>
