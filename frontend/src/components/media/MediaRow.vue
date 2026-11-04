<script setup lang="ts">
import type { Media } from '@/api/types'
import Skeleton from '@/components/ui/Skeleton.vue'
import PosterCard from './PosterCard.vue'

defineProps<{ title: string; items: Media[]; loading?: boolean }>()
</script>

<template>
  <section class="mrow" :aria-label="title">
    <h2 class="section-title mrow__title">{{ title }}</h2>
    <div class="mrow__scroll">
      <template v-if="loading">
        <div v-for="i in 8" :key="i" class="mrow__item">
          <Skeleton ratio="2 / 3" />
          <Skeleton height="12px" width="80%" radius="4px" style="margin-top: 8px" />
        </div>
      </template>
      <div v-for="m in items" v-else :key="`${m.type}:${m.id}`" class="mrow__item">
        <PosterCard :item="m" />
      </div>
    </div>
  </section>
</template>

<style scoped>
.mrow__title {
  padding: 0 var(--gutter);
  margin-bottom: 4px;
}
.mrow__scroll {
  display: flex;
  gap: 14px;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-snap-type: x proximity;
  scroll-padding: 0 var(--gutter);
  padding: 10px var(--gutter) 6px;
  scrollbar-width: none;
}
.mrow__scroll::-webkit-scrollbar {
  display: none;
}
.mrow__item {
  flex: 0 0 auto;
  width: clamp(112px, 30vw, 160px);
  scroll-snap-align: start;
}
</style>
