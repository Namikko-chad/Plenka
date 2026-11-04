<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Sheet from '@/components/ui/Sheet.vue'
import { msg } from '@/i18n'
import { usePlayback } from '@/stores/playback'

/** "Open with?" when the player preference is "ask". */
const playback = usePlayback()
const open = computed({
  get: () => !!playback.asking,
  set: (v) => {
    if (!v) playback.choose(null)
  },
})
</script>

<template>
  <Sheet v-model="open" dialog :title="msg.external.chooseTitle">
    <div class="chooser">
      <button type="button" class="chooser__item" autofocus @click="playback.choose('internal')">
        <Icon name="playCircle" />{{ msg.external.internal }}
      </button>
      <button type="button" class="chooser__item" @click="playback.choose('external')">
        <Icon name="external" />{{ msg.external.title }}
      </button>
    </div>
  </Sheet>
</template>

<style scoped>
.chooser {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.chooser__item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 48px;
  padding: 0 14px;
  border-radius: var(--r-control);
  border: 1px solid var(--c-border);
  background: var(--c-bg);
  font-size: 13px;
}
.chooser__item:hover {
  border-color: rgba(var(--rgb-accent), var(--a-line));
  color: var(--c-accent);
}
</style>
