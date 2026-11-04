<script setup lang="ts">
import type { Media } from '@/api/types'
import PosterCard from './PosterCard.vue'

defineProps<{ items: Media[] }>()
</script>

<template>
  <div class="mgrid">
    <PosterCard v-for="m in items" :key="`${m.type}:${m.id}`" :item="m" />
  </div>
</template>

<style scoped>
.mgrid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(clamp(104px, 28vw, 150px), 1fr));
  gap: 18px 14px;
}
</style>
