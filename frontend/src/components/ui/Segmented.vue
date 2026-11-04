<script setup lang="ts" generic="T extends string">
import { nextTick, onMounted, ref, watch } from 'vue'

/** `scroll`: labels that do not fit scroll inside the frame instead of overflowing it. */
const props = defineProps<{ options: { value: T; label: string }[]; label: string; block?: boolean; scroll?: boolean }>()
const model = defineModel<T>({ required: true })
const root = ref<HTMLElement | null>(null)

function revealActive() {
  if (!props.scroll) return
  root.value?.querySelector('.seg__item--active')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}
onMounted(revealActive)
watch(model, () => void nextTick(revealActive))

function onKey(e: KeyboardEvent, options: { value: T }[]) {
  const i = options.findIndex((o) => o.value === model.value)
  const d = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
  if (!d) return
  e.preventDefault()
  const next = options[(i + d + options.length) % options.length]
  if (next) model.value = next.value
  const root = (e.currentTarget as HTMLElement).querySelectorAll<HTMLButtonElement>('button')
  root[(i + d + options.length) % options.length]?.focus()
}
</script>

<template>
  <div
    ref="root"
    class="seg"
    :class="{ 'seg--block': block, 'seg--scroll': scroll }"
    role="radiogroup" :aria-label="label" @keydown="onKey($event, options)">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      role="radio"
      class="seg__item"
      :class="{ 'seg__item--active': o.value === model }"
      :aria-checked="o.value === model"
      :tabindex="o.value === model ? 0 : -1"
      @click="model = o.value"
    >
      {{ o.label }}
    </button>
  </div>
</template>

<style scoped>
.seg {
  display: inline-flex;
  align-items: stretch;
  background: var(--c-bg);
  border: 1px solid var(--c-border);
  border-radius: var(--r-control);
  padding: 2px;
  gap: 2px;
  height: 36px;
  max-width: 100%;
}
.seg--block {
  display: flex;
  width: 100%;
}
.seg--block .seg__item {
  flex: 1;
}
.seg--scroll {
  overflow-x: auto;
  scrollbar-width: none;
}
.seg--scroll::-webkit-scrollbar {
  display: none;
}
.seg--scroll .seg__item {
  flex: 1 0 auto;
}
.seg__item {
  background: transparent;
  border: none;
  border-radius: var(--r-segment);
  padding: 0 12px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
  color: var(--c-muted);
  white-space: nowrap;
  transition: background var(--dur), color var(--dur);
}
.seg__item:hover {
  color: var(--c-text);
}
.seg__item--active,
.seg__item--active:hover {
  background: rgba(var(--rgb-accent), var(--a-segment));
  color: var(--c-accent);
}
</style>
