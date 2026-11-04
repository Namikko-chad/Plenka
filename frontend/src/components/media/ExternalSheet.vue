<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Sheet from '@/components/ui/Sheet.vue'
import { useClipboard } from '@/composables/useClipboard'
import { msg } from '@/i18n'
import { usePlayback } from '@/stores/playback'
import { detectPlatform, externalLinks } from '@/utils/external'

/** "External player": VLC / chooser intents, m3u, copy link. Driven by the playback store. */
const playback = usePlayback()
const copy = useClipboard()
const platform = detectPlatform(navigator.userAgent, navigator.maxTouchPoints)

const open = computed({
  get: () => !!playback.external,
  set: (v) => {
    if (!v) playback.external = null
  },
})
const links = computed(() => {
  const t = playback.external
  if (!t) return null
  return externalLinks(location.origin, t.fileName, t.hash, t.fileId, t.media?.title ?? t.fileName)
})
</script>

<template>
  <Sheet v-model="open" :title="msg.external.title">
    <div v-if="links && playback.external" class="ext">
      <p class="ext__file ellipsis">{{ playback.external.fileName }}</p>
      <template v-if="platform === 'android'">
        <a class="ext__item" :href="links.vlcAndroid" autofocus><Icon name="playCircle" />{{ msg.external.vlc }}</a>
        <a class="ext__item" :href="links.chooserAndroid"><Icon name="external" />{{ msg.external.chooser }}</a>
      </template>
      <a v-else-if="platform === 'ios'" class="ext__item" :href="links.vlcIos" autofocus>
        <Icon name="playCircle" />{{ msg.external.vlc }}
      </a>
      <a class="ext__item" :href="links.m3u" download><Icon name="file" />{{ msg.external.m3u }}</a>
      <a class="ext__item" :href="links.playlist" download><Icon name="folder" />{{ msg.external.playlist }}</a>
      <button type="button" class="ext__item" @click="copy(links.stream)"><Icon name="copy" />{{ msg.external.copy }}</button>
      <p class="hint">{{ msg.external.hint }}</p>
    </div>
  </Sheet>
</template>

<style scoped>
.ext {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ext__file {
  margin: 0 0 4px;
  color: var(--c-muted);
  font-size: 12px;
}
.ext__item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 48px;
  padding: 0 14px;
  border-radius: var(--r-control);
  border: 1px solid var(--c-border);
  background: var(--c-bg);
  color: var(--c-text);
  font-size: 13px;
  text-align: left;
  width: 100%;
}
.ext__item:hover {
  border-color: rgba(var(--rgb-accent), var(--a-line));
  color: var(--c-accent);
}
</style>
