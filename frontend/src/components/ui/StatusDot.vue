<script setup lang="ts">
defineProps<{ ok: boolean; label?: string }>()
</script>

<template>
  <span class="dot" :class="ok ? 'dot--ok' : 'dot--error'" :role="label ? 'img' : undefined" :aria-label="label" />
</template>

<style scoped>
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  flex-shrink: 0;
}
/* The only glow in the system. */
.dot--ok {
  background: var(--c-ok);
  box-shadow: 0 0 8px var(--c-ok);
  animation: pulse 2s infinite;
}
.dot--error {
  background: var(--c-error);
  box-shadow: 0 0 8px var(--c-error);
  animation: pulse 1.5s infinite;
}
@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}
</style>
