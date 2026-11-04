<script setup lang="ts">
import { useToast } from '@/stores/toast'

const toast = useToast()
</script>

<template>
  <div class="toasts" aria-live="polite">
    <TransitionGroup name="toast">
      <button
        v-for="t in toast.items"
        :key="t.id"
        type="button"
        class="toast"
        :class="`toast--${t.tone}`"
        :role="t.tone === 'error' ? 'alert' : 'status'"
        @click="toast.dismiss(t.id)"
      >
        {{ t.text }}
      </button>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toasts {
  position: fixed;
  left: 0;
  right: 0;
  bottom: calc(var(--tabbar-h) + var(--safe-bottom) + 12px);
  z-index: var(--z-dialog);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  pointer-events: none;
}
@media (min-width: 840px) {
  .toasts {
    bottom: 24px;
    left: var(--rail-w);
  }
}
.toast {
  pointer-events: auto;
  max-width: 480px;
  padding: 10px 16px;
  border-radius: var(--r-control);
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  color: var(--c-text);
  font-size: 12px;
  text-align: left;
}
.toast--ok {
  border-color: rgba(var(--rgb-ok), var(--a-line));
  color: var(--c-ok);
}
.toast--error {
  border-color: rgba(var(--rgb-error), var(--a-line));
  color: var(--c-error);
}
.toast-enter-active,
.toast-leave-active {
  transition: all 0.2s;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
