<script setup lang="ts">
import type { Media } from '@/api/types'
import Badge from '@/components/ui/Badge.vue'
import { msg } from '@/i18n'
import { imageUrl, ratingTone } from '@/utils/media'
import NetImage from './NetImage.vue'

defineProps<{ item: Media }>()
</script>

<template>
  <RouterLink :to="`/${item.type}/${item.id}`" class="poster" :aria-label="`${item.title}${item.year ? `, ${item.year}` : ''}`">
    <div class="poster__frame">
      <NetImage :src="imageUrl(item.posterPath, 'w342')" :alt="item.title" />
      <Badge v-if="item.rating > 0" class="poster__rating" :tone="ratingTone(item.rating)" on-image>
        {{ item.rating.toFixed(1) }}
      </Badge>
      <Badge v-if="item.type === 'tv'" class="poster__tv" tone="accent" on-image>{{ msg.common.series }}</Badge>
    </div>
    <span class="poster__title clamp-2">{{ item.title }}</span>
    <span v-if="item.year" class="poster__year">{{ item.year }}</span>
  </RouterLink>
</template>

<style scoped>
.poster {
  display: flex;
  flex-direction: column;
  gap: 2px;
  color: var(--c-text);
  min-width: 0;
  border-radius: var(--r-card);
}
.poster:focus-visible {
  outline: none;
}
.poster__frame {
  position: relative;
  aspect-ratio: 2 / 3;
  border-radius: var(--r-card);
  overflow: hidden;
  border: 1px solid var(--c-border);
  margin-bottom: 6px;
  transition:
    transform 0.18s,
    border-color 0.18s;
}
.poster:hover .poster__frame,
.poster:focus-visible .poster__frame {
  transform: scale(1.04);
  border-color: var(--c-accent);
}
.poster:active .poster__frame {
  transform: scale(0.98);
}
.poster__rating {
  position: absolute;
  top: 6px;
  right: 6px;
}
.poster__tv {
  position: absolute;
  top: 6px;
  left: 6px;
}
.poster__title {
  font-size: 12px;
  line-height: 1.3;
}
.poster__year {
  font-size: 11px;
  color: var(--c-muted);
}
</style>
