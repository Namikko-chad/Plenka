<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import Icon from './Icon.vue'
import Spinner from './Spinner.vue'
import type { IconName } from './icons'

withDefaults(
  defineProps<{
    variant?: 'primary' | 'cyan' | 'green' | 'purple' | 'red' | 'neutral'
    size?: 'sm' | 'md' | 'lg'
    icon?: IconName
    loading?: boolean
    disabled?: boolean
    block?: boolean
    type?: 'button' | 'submit'
    to?: RouteLocationRaw
    /** External link, opened in a new tab. */
    href?: string
  }>(),
  { variant: 'cyan', size: 'md', type: 'button' },
)
</script>

<template>
  <RouterLink v-if="to" :to="to" class="btn" :class="[`btn--${variant}`, `btn--${size}`, { 'btn--block': block }]">
    <Icon v-if="icon" :name="icon" :size="size === 'lg' ? 18 : 15" />
    <span><slot /></span>
  </RouterLink>
  <a
    v-else-if="href"
    :href="href"
    target="_blank"
    rel="noopener noreferrer"
    class="btn"
    :class="[`btn--${variant}`, `btn--${size}`, { 'btn--block': block }]"
  >
    <Icon v-if="icon" :name="icon" :size="size === 'lg' ? 18 : 15" />
    <span><slot /></span>
  </a>
  <button
    v-else
    :type="type"
    class="btn"
    :class="[`btn--${variant}`, `btn--${size}`, { 'btn--block': block }]"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
  >
    <Spinner v-if="loading" :size="14" />
    <Icon v-else-if="icon" :name="icon" :size="size === 'lg' ? 18 : 15" />
    <span v-if="$slots.default"><slot /></span>
  </button>
</template>

<style scoped>
.btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: var(--control-h);
  padding: 0 14px;
  border-radius: var(--r-control);
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
  border: 1px solid transparent;
  background: transparent;
  white-space: nowrap;
  transition:
    background var(--dur),
    opacity var(--dur),
    border-color var(--dur);
  user-select: none;
  text-decoration: none;
}
/* invisible touch-target extension to ≥ 44px on touch screens */
@media (pointer: coarse) {
  .btn::after {
    content: '';
    position: absolute;
    inset: -7px 0;
  }
}
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn--sm {
  height: 24px;
  padding: 0 10px;
}
.btn--lg {
  height: 44px;
  padding: 0 20px;
  font-size: 13px;
}
.btn--block {
  width: 100%;
}
.btn--primary {
  background: var(--c-accent);
  color: var(--c-on-accent);
}
.btn--primary:hover:not(:disabled) {
  opacity: 0.9;
}
.btn--cyan {
  background: rgba(var(--rgb-accent), var(--a-ghost));
  border-color: rgba(var(--rgb-accent), var(--a-line));
  color: var(--c-accent);
}
.btn--cyan:hover:not(:disabled) {
  background: rgba(var(--rgb-accent), var(--a-ghost-hover));
}
.btn--green {
  background: rgba(var(--rgb-ok), var(--a-ghost));
  border-color: rgba(var(--rgb-ok), var(--a-line));
  color: var(--c-ok);
}
.btn--green:hover:not(:disabled) {
  background: rgba(var(--rgb-ok), var(--a-ghost-hover));
}
.btn--purple {
  background: rgba(var(--rgb-purple), var(--a-ghost));
  border-color: rgba(var(--rgb-purple), var(--a-line));
  color: var(--c-purple-l);
}
.btn--purple:hover:not(:disabled) {
  background: rgba(var(--rgb-purple), var(--a-ghost-hover));
}
.btn--red {
  background: rgba(var(--rgb-error), var(--a-ghost));
  border-color: rgba(var(--rgb-error), var(--a-line));
  color: var(--c-error);
}
.btn--red:hover:not(:disabled) {
  background: rgba(var(--rgb-error), var(--a-ghost-hover));
}
.btn--neutral {
  border-color: var(--c-border);
  color: var(--c-muted);
}
.btn--neutral:hover:not(:disabled) {
  background: rgba(var(--rgb-neutral), var(--a-ghost));
  color: var(--c-text);
}
</style>
