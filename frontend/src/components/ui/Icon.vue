<script setup lang="ts">
import { computed } from 'vue'
import { ICONS, type IconName } from './icons'

const props = withDefaults(defineProps<{ name: IconName; size?: number }>(), { size: 20 })
const icon = computed(() => ICONS[props.name] as { d: readonly string[]; fill?: boolean })
</script>

<template>
  <svg
    class="icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    :fill="icon.fill ? 'currentColor' : 'none'"
    :stroke="icon.fill ? 'none' : 'currentColor'"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path v-for="(d, i) in icon.d" :key="i" :d="d" />
  </svg>
</template>

<style scoped>
.icon {
  flex-shrink: 0;
  display: block;
}
</style>
