<script setup lang="ts">
import { msg } from '@/i18n'
withDefaults(defineProps<{ size?: number }>(), { size: 24 })
</script>

<template>
  <span class="spinner" :style="{ width: `${size}px`, height: `${size}px` }" role="status" :aria-label="msg.common.loading" />
</template>

<style scoped>
.spinner {
  display: inline-block;
  flex-shrink: 0;
  border-radius: 50%;
  border: 2px solid rgba(var(--rgb-accent), var(--a-line));
  border-top-color: var(--c-accent);
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
