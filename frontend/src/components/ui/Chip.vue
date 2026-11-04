<script setup lang="ts">
defineProps<{ active?: boolean }>()
</script>

<template>
  <button type="button" class="chip" :class="{ 'chip--active': active }" :aria-pressed="!!active">
    <slot />
  </button>
</template>

<style scoped>
.chip {
  height: 32px;
  padding: 0 12px;
  border-radius: 16px;
  border: 1px solid var(--c-border);
  background: var(--c-surface);
  color: var(--c-muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
  white-space: nowrap;
  transition: background var(--dur), color var(--dur), border-color var(--dur);
}
.chip:hover {
  color: var(--c-text);
}
.chip--active,
.chip--active:hover {
  background: rgba(var(--rgb-accent), var(--a-segment));
  border-color: rgba(var(--rgb-accent), var(--a-line));
  color: var(--c-accent);
}
</style>
