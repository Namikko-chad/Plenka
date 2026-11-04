<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import IconButton from '@/components/ui/IconButton.vue'
import Spinner from '@/components/ui/Spinner.vue'
import type { TrackOption } from '@/composables/usePlayerTracks'
import { msg } from '@/i18n'

/** Audio / subtitle choice inside the player (not teleported: it must stay visible in fullscreen). */
defineProps<{
  audioOptions: TrackOption[]
  subtitleOptions: TrackOption[]
  audio: string
  subtitle: string
  subLoading: boolean
  note?: string
}>()
defineEmits<{ audio: [key: string]; subtitle: [key: string]; close: [] }>()

const root = ref<HTMLElement | null>(null)
onMounted(async () => {
  await nextTick()
  root.value?.querySelector<HTMLElement>('[aria-checked="true"]')?.focus()
})
</script>

<template>
  <div ref="root" class="tracks" role="dialog" :aria-label="msg.player.tracks" @pointerup.stop>
    <header class="tracks__head">
      <span class="tracks__title">{{ msg.player.tracks }}</span>
      <IconButton icon="close" :label="msg.common.close" class="ctl" @click="$emit('close')" />
    </header>
    <div class="tracks__body">
      <section class="tracks__group" role="radiogroup" :aria-label="msg.player.audio">
        <h3 class="tracks__label">{{ msg.player.audio }}</h3>
        <button
          v-for="o in audioOptions"
          :key="o.key"
          type="button"
          role="radio"
          class="ctl tracks__item"
          :aria-checked="o.key === audio"
          @click="$emit('audio', o.key)"
        >
          <span class="tracks__dot" />
          <span class="tracks__name ellipsis">{{ o.label }}</span>
          <span v-if="o.hint" class="tracks__hint">{{ o.hint }}</span>
        </button>
        <p v-if="note" class="tracks__note">{{ note }}</p>
      </section>
      <section class="tracks__group" role="radiogroup" :aria-label="msg.player.subtitles">
        <h3 class="tracks__label">
          {{ msg.player.subtitles }}
          <Spinner v-if="subLoading" :size="12" :aria-label="msg.player.subsLoading" />
        </h3>
        <button
          v-for="o in subtitleOptions"
          :key="o.key"
          type="button"
          role="radio"
          class="ctl tracks__item"
          :aria-checked="o.key === subtitle"
          @click="$emit('subtitle', o.key)"
        >
          <span class="tracks__dot" />
          <span class="tracks__name ellipsis">{{ o.label }}</span>
          <span v-if="o.hint" class="tracks__hint">{{ o.hint }}</span>
        </button>
      </section>
    </div>
  </div>
</template>

<style scoped>
.tracks {
  position: absolute;
  right: calc(12px + var(--safe-right));
  bottom: calc(96px + var(--safe-bottom));
  width: min(360px, calc(100% - 24px));
  max-height: calc(100% - 140px);
  display: flex;
  flex-direction: column;
  background: rgba(var(--rgb-bg), 0.94);
  border: 1px solid var(--c-border);
  border-radius: var(--r-dialog);
  color: var(--c-text);
  z-index: 5;
}
.tracks__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 6px 6px 16px;
  border-bottom: 1px solid var(--c-border);
}
.tracks__title {
  font-weight: 700;
  font-size: 13px;
}
.tracks__body {
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tracks__group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tracks__label {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 8px 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--c-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.tracks__item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: var(--r-control);
  background: none;
  color: var(--c-text);
  font-size: 13px;
  text-align: left;
}
.tracks__item:hover,
.tracks__item:focus-visible {
  border-color: rgba(var(--rgb-accent), var(--a-line));
  outline: none;
}
.tracks__item[aria-checked='true'] {
  color: var(--c-accent);
}
.tracks__dot {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
  border-radius: 50%;
  border: 2px solid currentColor;
  opacity: 0.6;
}
.tracks__item[aria-checked='true'] .tracks__dot {
  background: currentColor;
  opacity: 1;
}
.tracks__name {
  flex: 1;
  min-width: 0;
}
.tracks__hint {
  font-size: 11px;
  color: var(--c-muted);
}
.tracks__note {
  margin: 4px 8px 0;
  font-size: 11px;
  line-height: 1.5;
  color: var(--c-muted);
}
</style>
