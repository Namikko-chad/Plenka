<script setup lang="ts">
import type { Tone } from '@/utils/media'

withDefaults(defineProps<{ tone?: Tone; onImage?: boolean }>(), { tone: 'neutral' })
</script>

<template>
  <span class="badge" :class="[`badge--${tone}`, { 'badge--img': onImage }]"><slot /></span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 7px;
  border-radius: var(--r-badge);
  font-size: 10px;
  font-weight: 600;
  line-height: 14px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
  border: 1px solid transparent;
}
.badge--img {
  backdrop-filter: blur(4px);
  background-color: rgba(var(--rgb-bg), 0.8);
}
.badge--ok {
  background-image: linear-gradient(rgba(var(--rgb-ok), var(--a-badge)), rgba(var(--rgb-ok), var(--a-badge)));
  border-color: rgba(var(--rgb-ok), var(--a-line));
  color: var(--c-ok);
}
.badge--error {
  background-image: linear-gradient(rgba(var(--rgb-error), var(--a-badge)), rgba(var(--rgb-error), var(--a-badge)));
  border-color: rgba(var(--rgb-error), var(--a-line));
  color: var(--c-error);
}
.badge--warning {
  background-image: linear-gradient(rgba(var(--rgb-warning), var(--a-badge)), rgba(var(--rgb-warning), var(--a-badge)));
  border-color: rgba(var(--rgb-warning), var(--a-line));
  color: var(--c-warning);
}
.badge--accent {
  background-image: linear-gradient(rgba(var(--rgb-accent), var(--a-badge)), rgba(var(--rgb-accent), var(--a-badge)));
  border-color: rgba(var(--rgb-accent), var(--a-line));
  color: var(--c-accent);
}
.badge--active {
  background-image: linear-gradient(rgba(var(--rgb-active), var(--a-badge)), rgba(var(--rgb-active), var(--a-badge)));
  border-color: rgba(var(--rgb-active), var(--a-line));
  color: var(--c-active);
}
.badge--purple {
  background-image: linear-gradient(rgba(var(--rgb-purple), var(--a-badge)), rgba(var(--rgb-purple), var(--a-badge)));
  border-color: rgba(var(--rgb-purple), var(--a-line));
  color: var(--c-purple-l);
}
.badge--neutral {
  background-image: linear-gradient(rgba(var(--rgb-neutral), var(--a-badge)), rgba(var(--rgb-neutral), var(--a-badge)));
  color: var(--c-muted);
}
</style>
