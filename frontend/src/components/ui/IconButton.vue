<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from './icons'

withDefaults(defineProps<{ icon: IconName; label: string; size?: number; tone?: 'default' | 'accent' | 'danger' }>(), {
  size: 20,
  tone: 'default',
})
</script>

<template>
  <button type="button" class="icon-btn" :class="`icon-btn--${tone}`" :aria-label="label" :title="label">
    <Icon :name="icon" :size="size" />
  </button>
</template>

<style scoped>
.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--touch);
  height: var(--touch);
  flex-shrink: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--c-text);
  transition: background var(--dur), color var(--dur);
}
.icon-btn:hover {
  background: rgba(var(--rgb-accent), var(--a-ghost));
  color: var(--c-accent);
}
.icon-btn--accent {
  color: var(--c-accent);
}
.icon-btn--danger:hover {
  background: rgba(var(--rgb-error), var(--a-ghost));
  color: var(--c-error);
}
</style>
