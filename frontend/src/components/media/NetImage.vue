<script setup lang="ts">
import { ref, watch } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import type { IconName } from '@/components/ui/icons'

const props = withDefaults(defineProps<{ src: string | null; alt: string; fallback?: IconName; eager?: boolean }>(), {
  fallback: 'film',
})
const failed = ref(false)
const loaded = ref(false)
watch(
  () => props.src,
  () => {
    failed.value = false
    loaded.value = false
  },
)
</script>

<template>
  <div class="netimg">
    <img
      v-if="src && !failed"
      :src="src"
      :alt="alt"
      :loading="eager ? 'eager' : 'lazy'"
      decoding="async"
      draggable="false"
      :class="{ 'netimg--loaded': loaded }"
      @load="loaded = true"
      @error="failed = true"
    />
    <div v-else class="netimg__fallback" role="img" :aria-label="alt"><Icon :name="fallback" :size="32" /></div>
  </div>
</template>

<style scoped>
.netimg {
  position: relative;
  width: 100%;
  height: 100%;
  background: var(--c-surface);
  overflow: hidden;
}
.netimg img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.25s;
}
.netimg img.netimg--loaded {
  opacity: 1;
}
.netimg__fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--c-border);
}
</style>
