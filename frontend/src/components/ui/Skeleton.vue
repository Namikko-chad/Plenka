<script setup lang="ts">
defineProps<{ width?: string; height?: string; radius?: string; ratio?: string }>()
</script>

<template>
  <div
    class="skeleton"
    aria-hidden="true"
    :style="{ width, height, borderRadius: radius ?? 'var(--r-card)', aspectRatio: ratio }"
  />
</template>

<style scoped>
.skeleton {
  background: linear-gradient(
    100deg,
    var(--c-surface) 30%,
    rgba(var(--rgb-accent), 0.05) 50%,
    var(--c-surface) 70%
  );
  background-size: 300% 100%;
  border: 1px solid var(--c-border);
  animation: shimmer 1.4s ease-in-out infinite;
}
@keyframes shimmer {
  from {
    background-position: 100% 0;
  }
  to {
    background-position: 0 0;
  }
}
</style>
