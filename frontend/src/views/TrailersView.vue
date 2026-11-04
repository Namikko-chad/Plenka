<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { MediaDetails, MediaType } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import Badge from '@/components/ui/Badge.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Icon from '@/components/ui/Icon.vue'
import Sheet from '@/components/ui/Sheet.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { useAsync } from '@/composables/useAsync'
import { msg } from '@/i18n'
import { useCatalog } from '@/stores/catalog'
import { formatDate } from '@/utils/format'
import { type Trailer, youtubeEmbed, youtubeThumb, youtubeWatch } from '@/utils/trailers'

/** Trailers of a title (TMDB videos on YouTube), opened from "Watch" → "Trailers". */
const props = defineProps<{ type: MediaType; id: number }>()
const catalog = useCatalog()

const media = useAsync<MediaDetails>(() => catalog.getDetails(props.type, props.id))
const list = useAsync<Trailer[]>(() => catalog.getTrailers(props.type, props.id))
const subtitle = computed(() => {
  const m = media.data.value
  return m ? [m.title, m.year].filter(Boolean).join(' · ') : ''
})

const playing = ref<Trailer | null>(null)
const playerOpen = computed({
  get: () => playing.value !== null,
  set: (v) => {
    if (!v) playing.value = null
  },
})

const typeLabel = (t: Trailer) => msg.trailers.types[t.type] ?? t.type

onMounted(() => {
  void media.run()
  void list.run()
})
</script>

<template>
  <div class="page">
    <AppBar :title="msg.trailers.title" :subtitle="subtitle" :back="`/${type}/${id}`" />
    <div class="page-body">
      <ErrorBanner v-if="list.error.value" :error="list.error.value" retry @retry="list.run()" />

      <div v-else-if="!list.data.value" class="trailers">
        <Skeleton v-for="i in 6" :key="i" ratio="16 / 9" />
      </div>

      <EmptyState v-else-if="!list.data.value.length" icon="film" :text="msg.trailers.empty" />

      <ul v-else class="trailers">
        <li v-for="(t, i) in list.data.value" :key="t.key">
          <button type="button" class="trailer" :autofocus="i === 0" @click="playing = t">
            <span class="trailer__thumb">
              <img :src="youtubeThumb(t.key)" :alt="t.name" loading="lazy" />
              <Icon name="play" :size="28" class="trailer__play" />
            </span>
            <span class="trailer__body">
              <span class="trailer__name clamp-2">{{ t.name }}</span>
              <span class="trailer__meta">
                <Badge tone="accent">{{ typeLabel(t) }}</Badge>
                <span v-if="t.lang">{{ t.lang.toUpperCase() }}</span>
                <span v-if="t.publishedAt">{{ formatDate(t.publishedAt) }}</span>
              </span>
            </span>
          </button>
        </li>
      </ul>
    </div>

    <Sheet v-model="playerOpen" dialog wide :title="playing?.name">
      <div v-if="playing" class="trailer-player">
        <iframe
          :src="youtubeEmbed(playing.key)"
          :title="playing.name"
          allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
          allowfullscreen
          referrerpolicy="strict-origin-when-cross-origin"
        />
      </div>
      <template #footer>
        <UiButton v-if="playing" icon="external" :href="youtubeWatch(playing.key)">
          {{ msg.trailers.youtube }}
        </UiButton>
        <UiButton variant="neutral" autofocus @click="playing = null">{{ msg.common.close }}</UiButton>
      </template>
    </Sheet>
  </div>
</template>

<style scoped>
.trailers {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
}
.trailer {
  display: flex;
  flex-direction: column;
  width: 100%;
  padding: 0;
  text-align: left;
  color: var(--c-text);
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-radius: var(--r-card);
  overflow: hidden;
  transition: border-color var(--dur);
}
.trailer:hover,
.trailer:focus-visible {
  border-color: var(--c-accent);
  outline: none;
}
.trailer__thumb {
  position: relative;
  display: block;
  aspect-ratio: 16 / 9;
  background: var(--c-bg);
}
.trailer__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.trailer__play {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  padding: 12px;
  box-sizing: content-box;
  border-radius: 50%;
  color: var(--c-accent);
  background: rgba(var(--rgb-bg), 0.7);
}
.trailer__body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px 12px;
}
.trailer__name {
  font-size: 13px;
  line-height: 1.35;
}
.trailer__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 10px;
  font-size: 11px;
  color: var(--c-muted);
}
.trailer-player {
  width: 100%;
  aspect-ratio: 16 / 9;
  background: #000;
  border-radius: var(--r-control);
  overflow: hidden;
}
.trailer-player iframe {
  width: 100%;
  height: 100%;
  border: 0;
  display: block;
}
</style>
