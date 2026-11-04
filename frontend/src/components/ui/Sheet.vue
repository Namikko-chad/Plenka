<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { msg } from '@/i18n'
import IconButton from './IconButton.vue'

/** Bottom sheet on phones, centered dialog on wide screens. */
const props = withDefaults(defineProps<{ title?: string; dialog?: boolean; wide?: boolean }>(), {
  dialog: false,
  wide: false,
})
const open = defineModel<boolean>({ required: true })
const panel = ref<HTMLElement | null>(null)
let restore: HTMLElement | null = null

function close() {
  open.value = false
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
    return
  }
  if (e.key !== 'Tab' || !panel.value) return
  const nodes = panel.value.querySelectorAll<HTMLElement>(
    'button:not([disabled]), [href], input:not([disabled]), select, textarea, [tabindex]:not([tabindex="-1"])',
  )
  if (!nodes.length) return
  const first = nodes[0]!
  const last = nodes[nodes.length - 1]!
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}

watch(
  open,
  async (v) => {
    if (v) {
      restore = document.activeElement as HTMLElement | null
      await nextTick()
      const target = panel.value?.querySelector<HTMLElement>('[autofocus]') ?? panel.value
      target?.focus()
    } else {
      restore?.focus?.()
      restore = null
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => restore?.focus?.())
defineExpose({ close, props })
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet">
      <div v-if="open" class="sheet-overlay" :class="{ 'sheet-overlay--dialog': dialog, 'sheet-overlay--wide': wide }" @click.self="close" @keydown="onKey">
        <div ref="panel" class="sheet" role="dialog" aria-modal="true" :aria-label="title" tabindex="-1">
          <div v-if="!dialog" class="sheet__grip" aria-hidden="true" />
          <header v-if="title" class="sheet__head">
            <h2 class="sheet__title">{{ title }}</h2>
            <IconButton icon="close" :label="msg.common.close" @click="close" />
          </header>
          <div class="sheet__body"><slot /></div>
          <footer v-if="$slots.footer" class="sheet__foot"><slot name="footer" /></footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.sheet-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-overlay);
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.sheet {
  width: 100%;
  max-height: 88dvh;
  display: flex;
  flex-direction: column;
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-bottom: none;
  border-radius: var(--r-dialog) var(--r-dialog) 0 0;
  padding-bottom: var(--safe-bottom);
  outline: none;
}
.sheet__grip {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--c-border);
  margin: 8px auto 0;
  flex-shrink: 0;
}
.sheet__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 6px 0 18px;
  flex-shrink: 0;
}
.sheet__title {
  flex: 1;
  margin: 0;
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 700;
}
.sheet__body {
  padding: 12px 18px 18px;
  overflow-y: auto;
  overscroll-behavior: contain;
}
.sheet__foot {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 12px;
  padding: 12px 18px 16px;
  border-top: 1px solid var(--c-border);
  flex-shrink: 0;
}
.sheet-overlay--dialog {
  z-index: var(--z-dialog);
  align-items: center;
  padding: 16px;
}
.sheet-overlay--dialog .sheet {
  max-width: 420px;
  border-radius: var(--r-dialog);
  border-bottom: 1px solid var(--c-border);
  padding-bottom: 0;
}
/* wide: video and other 16:9 content */
.sheet-overlay--wide .sheet {
  max-width: 960px;
}
@media (min-width: 840px) {
  .sheet-overlay {
    align-items: center;
    padding: 24px;
  }
  .sheet {
    max-width: 560px;
    max-height: 84vh;
    border-radius: var(--r-dialog);
    border-bottom: 1px solid var(--c-border);
    padding-bottom: 0;
  }
  .sheet__grip {
    display: none;
  }
  .sheet-overlay--wide .sheet {
    max-width: 960px;
  }
}

.sheet-enter-active,
.sheet-leave-active {
  transition: opacity 0.2s;
}
.sheet-enter-active .sheet,
.sheet-leave-active .sheet {
  transition: transform 0.24s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.sheet-enter-from,
.sheet-leave-to {
  opacity: 0;
}
.sheet-enter-from .sheet,
.sheet-leave-to .sheet {
  transform: translateY(40px);
}
</style>
