<script setup lang="ts">
withDefaults(defineProps<{ value: number; tone?: 'accent' | 'ok' | 'warning' }>(), { tone: 'accent' })
</script>

<template>
  <div class="progress" role="progressbar" :aria-valuenow="Math.round(value * 100)" aria-valuemin="0" aria-valuemax="100">
    <div class="progress__bar" :class="`progress__bar--${tone}`" :style="{ width: `${Math.min(1, Math.max(0, value)) * 100}%` }" />
  </div>
</template>

<style scoped>
.progress {
  height: 3px;
  background: var(--c-border);
  border-radius: 2px;
  overflow: hidden;
}
.progress__bar {
  height: 100%;
  background: var(--c-accent);
  transition: width 0.3s;
}
.progress__bar--ok {
  background: var(--c-ok);
}
.progress__bar--warning {
  background: var(--c-warning);
}
</style>
