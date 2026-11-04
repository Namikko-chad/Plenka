<script setup lang="ts">
import { useClipboard } from '@/composables/useClipboard'
import { msg } from '@/i18n'
import IconButton from './IconButton.vue'

defineProps<{ label: string; value: string }>()
const copy = useClipboard()
</script>

<template>
  <div class="copy-field">
    <span class="label">{{ label }}</span>
    <div class="copy-field__row">
      <code class="copy-field__value">{{ value || '—' }}</code>
      <IconButton icon="copy" :label="`${msg.common.copy}: ${label}`" :size="18" :disabled="!value" @click="copy(value)" />
    </div>
  </div>
</template>

<style scoped>
.copy-field__row {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--c-bg);
  border: 1px solid var(--c-border);
  border-radius: var(--r-control);
  padding-left: 12px;
}
.copy-field__value {
  flex: 1;
  min-width: 0;
  font-family: var(--font-mono);
  font-size: 12px;
  overflow-wrap: anywhere;
  padding: 6px 0;
  user-select: all;
}
</style>
