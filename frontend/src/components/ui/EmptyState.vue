<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from './icons'

withDefaults(defineProps<{ icon?: IconName; title?: string; text?: string }>(), { icon: 'info' })
</script>

<template>
  <div class="empty">
    <Icon :name="icon" :size="32" class="empty__icon" />
    <h2 v-if="title" class="empty__title">{{ title }}</h2>
    <p v-if="text" class="empty__text">{{ text }}</p>
    <div v-if="$slots.default" class="empty__actions"><slot /></div>
  </div>
</template>

<style scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 10px;
  padding: 48px 24px;
  max-width: 420px;
  margin: 0 auto;
}
.empty__icon {
  color: var(--c-muted);
}
.empty__title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
}
.empty__text {
  margin: 0;
  color: var(--c-muted);
  white-space: pre-line;
}
.empty__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
  margin-top: 8px;
}
</style>
