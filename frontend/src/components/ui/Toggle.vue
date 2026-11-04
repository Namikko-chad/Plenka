<script setup lang="ts">
import { useId } from 'vue'

defineProps<{ label: string; hint?: string; disabled?: boolean }>()
const model = defineModel<boolean>({ required: true })
const id = useId()
</script>

<template>
  <label class="toggle-row" :class="{ 'toggle-row--disabled': disabled }" :for="id">
    <span class="toggle-row__text">
      <span class="toggle-row__label">{{ label }}</span>
      <span v-if="hint" class="toggle-row__hint">{{ hint }}</span>
    </span>
    <input :id="id" v-model="model" class="sr-only" type="checkbox" role="switch" :disabled="disabled" />
    <span class="toggle" :class="{ 'toggle--on': model }" aria-hidden="true"><span class="toggle__thumb" /></span>
  </label>
</template>

<style scoped>
.toggle-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: var(--touch);
  cursor: pointer;
}
.toggle-row--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.toggle-row__text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.toggle-row__label {
  font-size: 13px;
}
.toggle-row__hint {
  font-size: 11px;
  color: var(--c-muted);
}
.toggle {
  position: relative;
  width: 40px;
  height: 22px;
  border-radius: 11px;
  background: var(--c-border);
  transition: background 0.2s;
  flex-shrink: 0;
}
.toggle--on {
  background: var(--c-accent);
}
.toggle__thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.2s;
}
.toggle--on .toggle__thumb {
  transform: translateX(18px);
}
input:focus-visible + .toggle {
  outline: 2px solid var(--c-accent);
  outline-offset: 2px;
}
</style>
