<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { MediaDetails, MediaType } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import NetImage from '@/components/media/NetImage.vue'
import Badge from '@/components/ui/Badge.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Icon from '@/components/ui/Icon.vue'
import Sheet from '@/components/ui/Sheet.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { useAsync } from '@/composables/useAsync'
import { msg } from '@/i18n'
import { useCatalog } from '@/stores/catalog'
import { useHistory } from '@/stores/history'
import { imageUrl, ratingTone } from '@/utils/media'

const props = defineProps<{ type: MediaType; id: number }>()
const catalog = useCatalog()
const history = useHistory()
const router = useRouter()

// "Watch" asks what to watch, like Lampa: torrents or trailers.
const chooser = ref(false)
function watch(kind: 'releases' | 'trailers') {
  chooser.value = false
  void router.push(`/${kind}/${props.type}/${props.id}`)
}

const details = useAsync<MediaDetails>(() => catalog.getDetails(props.type, props.id))
const d = computed(() => details.data.value)
const last = computed(() => history.lastFor({ type: props.type, id: props.id }))

const facts = computed(() => {
  const m = d.value
  if (!m) return []
  const out: string[] = []
  if (m.year) out.push(String(m.year))
  if (m.countries.length) out.push(m.countries.slice(0, 2).join(', '))
  if (m.type === 'movie' && m.runtime) out.push(msg.details.minutes(m.runtime))
  if (m.type === 'tv' && m.seasons) out.push(msg.details.seasons(m.seasons))
  if (m.type === 'tv' && m.episodes) out.push(msg.details.episodes(m.episodes))
  return out
})

onMounted(() => {
  void details.run()
  void history.ensure()
})
</script>

<template>
  <div class="page">
    <AppBar :title="d?.title ?? ''" back />

    <div v-if="details.error.value" class="page-body">
      <ErrorBanner :error="details.error.value" retry @retry="details.run()" />
    </div>

    <div v-else-if="!d" class="page-body">
      <div class="det__head">
        <Skeleton class="det__poster" ratio="2 / 3" />
        <div class="stack" style="flex: 1">
          <Skeleton height="24px" width="70%" radius="6px" />
          <Skeleton height="14px" width="40%" radius="4px" />
          <Skeleton height="14px" width="55%" radius="4px" />
        </div>
      </div>
      <Skeleton height="96px" />
    </div>

    <template v-else>
      <div class="det__backdrop" aria-hidden="true">
        <img v-if="d.backdropPath" :src="imageUrl(d.backdropPath, 'w1280')!" alt="" />
      </div>
      <div class="page-body det">
        <div class="det__head">
          <div class="det__poster">
            <NetImage :src="imageUrl(d.posterPath, 'w500')" :alt="d.title" eager />
          </div>
          <div class="det__info">
            <h2 class="det__title">{{ d.title }}</h2>
            <p v-if="d.originalTitle && d.originalTitle !== d.title" class="det__orig">{{ d.originalTitle }}</p>
            <div class="row-wrap">
              <Badge v-if="d.rating > 0" :tone="ratingTone(d.rating)">★ {{ d.rating.toFixed(1) }}</Badge>
              <Badge v-if="d.type === 'tv'" tone="accent">{{ msg.common.series }}</Badge>
            </div>
            <p v-if="facts.length" class="det__facts">{{ facts.join(' · ') }}</p>
            <p v-if="d.genres.length" class="det__genres">{{ d.genres.join(', ') }}</p>
          </div>
        </div>

        <div class="det__actions">
          <UiButton variant="primary" size="lg" icon="play" @click="chooser = true">{{ msg.details.watch }}</UiButton>
          <UiButton
            v-if="last"
            size="lg"
            icon="playCircle"
            class="det__continue"
            :to="{ name: 'files', params: { hash: last.hash }, query: { resume: String(last.fileId) } }"
          >
            {{ msg.details.continue(last.fileName) }}
          </UiButton>
        </div>

        <p v-if="d.tagline" class="det__tagline">«{{ d.tagline }}»</p>

        <section v-if="d.overview" class="stack">
          <h3 class="section-title">{{ msg.details.overview }}</h3>
          <p class="det__overview">{{ d.overview }}</p>
        </section>

        <section v-if="d.cast.length" class="stack">
          <h3 class="section-title">{{ msg.details.cast }}</h3>
          <p class="det__cast">{{ d.cast.join(', ') }}</p>
        </section>
      </div>
    </template>

    <Sheet v-model="chooser" dialog :title="msg.details.watchTitle">
      <div class="chooser">
        <button type="button" class="chooser__item" autofocus @click="watch('releases')">
          <Icon name="download" />
          <span class="chooser__text">
            <span>{{ msg.details.torrents }}</span>
            <span class="small">{{ msg.details.torrentsHint }}</span>
          </span>
        </button>
        <button type="button" class="chooser__item" @click="watch('trailers')">
          <Icon name="youtube" />
          <span class="chooser__text">
            <span>{{ msg.details.trailers }}</span>
            <span class="small">{{ msg.details.trailersHint }}</span>
          </span>
        </button>
      </div>
    </Sheet>
  </div>
</template>

<style scoped>
.det__backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 340px;
  overflow: hidden;
  pointer-events: none;
}
.det__backdrop img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.28;
  mask-image: linear-gradient(black 30%, transparent);
  -webkit-mask-image: linear-gradient(black 30%, transparent);
}
.det {
  position: relative;
}
.det__head {
  display: flex;
  gap: 18px;
  align-items: flex-start;
  padding-top: 12px;
}
.det__poster {
  width: clamp(120px, 34vw, 220px);
  flex-shrink: 0;
  aspect-ratio: 2 / 3;
  border-radius: var(--r-card);
  overflow: hidden;
  border: 1px solid var(--c-border);
}
.det__info {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.det__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: clamp(20px, 4.5vw, 30px);
  font-weight: 800;
  letter-spacing: -0.5px;
  line-height: 1.15;
}
.det__orig {
  margin: 0;
  color: var(--c-muted);
}
.det__facts,
.det__genres {
  margin: 0;
  font-size: 12px;
  color: var(--c-muted);
}
.det__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.det__continue {
  max-width: 100%;
}
.det__continue :deep(span) {
  overflow: hidden;
  text-overflow: ellipsis;
}
.det__tagline {
  margin: 0;
  font-style: italic;
  color: var(--c-muted);
}
.det__overview,
.det__cast {
  margin: 0;
  line-height: 1.6;
}
.chooser {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.chooser__item {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 56px;
  padding: 8px 14px;
  text-align: left;
  border-radius: var(--r-control);
  border: 1px solid var(--c-border);
  background: var(--c-bg);
  font-size: 13px;
}
.chooser__item:hover,
.chooser__item:focus-visible {
  border-color: rgba(var(--rgb-accent), var(--a-line));
  color: var(--c-accent);
  outline: none;
}
.chooser__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
@media (max-width: 480px) {
  .det__actions > * {
    flex: 1 1 100%;
  }
}
</style>
