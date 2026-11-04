<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { torrentsApi } from '@/api/torrents'
import type { Media, TorrentFile, Torrent } from '@/api/types'
import TracksPanel from '@/components/media/TracksPanel.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Icon from '@/components/ui/Icon.vue'
import IconButton from '@/components/ui/IconButton.vue'
import Spinner from '@/components/ui/Spinner.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { usePlayerTracks } from '@/composables/usePlayerTracks'
import { usePolling } from '@/composables/usePolling'
import { msg } from '@/i18n'
import { useHistory } from '@/stores/history'
import { usePlayback, type PlayTarget } from '@/stores/playback'
import { useToast } from '@/stores/toast'
import { reviveFromHistory, useTorrents } from '@/stores/torrents'
import { streamPath } from '@/utils/external'
import { fileName, isVideo, playableFiles } from '@/utils/files'
import { formatDuration, formatSpeed } from '@/utils/format'
import { resumeSeconds } from '@/utils/history'
import { imageUrl, mediaFromTorrentData } from '@/utils/media'

const props = defineProps<{ hash: string; fileId: number }>()
const router = useRouter()
const history = useHistory()
const playback = usePlayback()
const torrents = useTorrents()
const toast = useToast()

const root = ref<HTMLElement | null>(null)
const video = ref<HTMLVideoElement | null>(null)

const status = ref<Torrent | null>(null)
const files = ref<TorrentFile[]>([])
/** Every file of the torrent: subtitle and audio files next to the video. */
const allFiles = ref<TorrentFile[]>([])
const loadError = ref<unknown>(null)
const mediaError = ref(false)
/** Sound plays but the browser can't decode the picture (HEVC 10-bit, HDR10+, Dolby Vision). */
const noPicture = ref(false)
let audioOnly = false
const tracksOpen = ref(false)

const playing = ref(false)
const buffering = ref(true)
const current = ref(0)
const duration = ref(0)
const bufferedEnd = ref(0)
const scrub = ref<number | null>(null)
const overlay = ref(true)
const fullscreen = ref(false)
const seekHint = ref<{ text: string; side: 'left' | 'right' | 'center' } | null>(null)

const file = computed(() => files.value.find((f) => f.id === props.fileId) ?? null)
const name = computed(() => (file.value ? fileName(file.value.path) : ''))
const next = computed(() => {
  const i = files.value.findIndex((f) => f.id === props.fileId)
  return i >= 0 ? (files.value[i + 1] ?? null) : null
})
const fromHistory = computed(
  () => history.entry(props.hash, props.fileId) ?? history.entries.find((e) => e.hash === props.hash) ?? null,
)
const media = computed<Media | null>(() => mediaFromTorrentData(status.value?.data) ?? fromHistory.value?.media ?? null)
const torrentTitle = computed(() => status.value?.title || fromHistory.value?.torrentTitle || '')
const title = computed(() => media.value?.title || torrentTitle.value || name.value)
const src = computed(() => (file.value ? streamPath(name.value, props.hash, file.value.id) : undefined))
const shown = computed(() => scrub.value ?? current.value)
const pct = (v: number) => (duration.value > 0 ? Math.min(100, (v / duration.value) * 100) : 0)
const busy = computed(() => (buffering.value || !file.value) && !mediaError.value && !loadError.value)
const busyText = computed(() => (file.value ? msg.player.buffering : msg.player.loadingFile))
const statLine = computed(() => {
  const s = status.value
  if (!s) return ''
  return `${formatSpeed(s.downloadSpeed)} · ${msg.player.peers(s.activePeers)}`
})

const target = (): PlayTarget | null =>
  file.value
    ? { hash: props.hash, fileId: file.value.id, fileName: name.value, torrentTitle: torrentTitle.value, media: media.value }
    : null

// ---------- loading ----------
let ctrl: AbortController | null = null

async function load() {
  ctrl?.abort()
  const c = new AbortController()
  ctrl = c
  loadError.value = null
  try {
    await history.ensure()
    const st = await torrents.waitForFiles(props.hash, {
      signal: c.signal,
      revive: reviveFromHistory(props.hash, fromHistory.value),
      onStatus: (s) => (status.value = s),
    })
    if (!st) throw new Error(msg.files.noPeers)
    allFiles.value = st.files
    files.value = playableFiles(st.files)
    if (!file.value) throw new Error(msg.player.notFound)
    setupMediaSession()
  } catch (e) {
    if (!c.signal.aborted) loadError.value = e
  }
}

// ---------- audio / subtitles ----------
const tracks = usePlayerTracks({
  video,
  hash: () => props.hash,
  file,
  files: allFiles,
  onError: (text) => toast.error(text),
})
const tracksNote = computed(() => (tracks.audioListSupported.value ? '' : msg.player.noAudioTracks))

function toggleTracks() {
  tracksOpen.value = !tracksOpen.value
  if (tracksOpen.value) {
    tracks.scan()
    overlay.value = true
    clearTimeout(hideTimer)
  } else {
    scheduleHide()
  }
}

/** A video file whose picture is 0×0 once frames arrive: the browser decodes only the sound. */
function checkPicture() {
  const v = video.value
  if (!v || audioOnly || noPicture.value || !file.value || !isVideo(file.value.path)) return
  if (v.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA && v.videoWidth === 0) {
    noPicture.value = true
    v.pause()
  }
}

function listenOnly() {
  audioOnly = true
  noPicture.value = false
  void video.value?.play().catch(() => undefined)
}

// ---------- progress ----------
function save(keepalive = false) {
  const v = video.value
  const f = file.value
  if (!v || !f || mediaError.value) return
  const d = v.duration
  if (!Number.isFinite(d) || d <= 0 || v.currentTime <= 0) return
  void history.save(
    {
      hash: props.hash,
      fileId: f.id,
      fileName: name.value,
      torrentTitle: torrentTitle.value,
      media: media.value,
      positionMs: Math.round(v.currentTime * 1000),
      durationMs: Math.round(d * 1000),
    },
    keepalive,
  )
}

const saveTimer = setInterval(() => {
  if (playing.value) save()
}, 5000)
const onPageHide = () => save(true)

// ---------- video events ----------
let resumed = false
function onLoadedMetadata() {
  const v = video.value
  if (!v) return
  duration.value = Number.isFinite(v.duration) ? v.duration : 0
  if (!resumed) {
    resumed = true
    const at = resumeSeconds(history.entry(props.hash, props.fileId))
    if (at > 0 && (!duration.value || at < duration.value)) v.currentTime = at
  }
  v.play().catch(() => {
    // autoplay blocked: keep the overlay with the play button
    overlay.value = true
  })
}

function onTime() {
  const v = video.value
  if (!v) return
  current.value = v.currentTime
  if (Number.isFinite(v.duration)) duration.value = v.duration
  updateBuffered()
}

function updateBuffered() {
  const v = video.value
  if (!v) return
  const t = v.currentTime
  let end = 0
  for (let i = 0; i < v.buffered.length; i++) {
    if (v.buffered.start(i) <= t + 0.5 && v.buffered.end(i) >= t) end = v.buffered.end(i)
  }
  bufferedEnd.value = end
}

function onPlay() {
  playing.value = true
  scheduleHide()
}
function onPause() {
  playing.value = false
  overlay.value = true
  save()
}
function onEnded() {
  save()
  const n = next.value
  if (n) void router.replace({ name: 'player', params: { hash: props.hash, fileId: String(n.id) } })
  else router.back()
}
function onError() {
  mediaError.value = true
  buffering.value = false
  overlay.value = true
}

function retryMedia() {
  mediaError.value = false
  buffering.value = true
  resumed = false
  video.value?.load()
}

// ---------- controls ----------
function togglePlay() {
  const v = video.value
  if (!v) return
  if (v.paused) void v.play().catch(() => undefined)
  else v.pause()
  showOverlay()
}

let hintTimer: ReturnType<typeof setTimeout> | undefined
function seekBy(delta: number, side: 'left' | 'right' | 'center' = delta < 0 ? 'left' : 'right') {
  const v = video.value
  if (!v) return
  const d = Number.isFinite(v.duration) ? v.duration : Infinity
  v.currentTime = Math.max(0, Math.min(d - 1, v.currentTime + delta))
  current.value = v.currentTime
  seekHint.value = { text: msg.format.seconds(`${delta < 0 ? '−' : '+'}${Math.abs(delta)}`), side }
  clearTimeout(hintTimer)
  hintTimer = setTimeout(() => (seekHint.value = null), 700)
  showOverlay()
}

function onScrub(e: Event) {
  scrub.value = Number((e.target as HTMLInputElement).value)
  showOverlay()
}
function onScrubEnd(e: Event) {
  const v = video.value
  const t = Number((e.target as HTMLInputElement).value)
  if (v && Number.isFinite(t)) v.currentTime = t
  current.value = t
  scrub.value = null
}

function playNext() {
  const n = next.value
  if (!n) return
  save()
  void router.replace({ name: 'player', params: { hash: props.hash, fileId: String(n.id) } })
}

function openExternal() {
  const t = target()
  if (!t) return
  video.value?.pause()
  playback.openExternal(t)
}

type WebkitVideo = HTMLVideoElement & { webkitEnterFullscreen?: () => void }

async function toggleFullscreen() {
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen()
    } else if (root.value?.requestFullscreen) {
      await root.value.requestFullscreen({ navigationUI: 'hide' })
      const o = screen.orientation as ScreenOrientation & { lock?: (o: string) => Promise<void> }
      await o.lock?.('landscape').catch(() => undefined)
    } else {
      // iPhone Safari: only the native video fullscreen exists
      ;(video.value as WebkitVideo | null)?.webkitEnterFullscreen?.()
    }
  } catch {
    /* fullscreen denied */
  }
  showOverlay()
}
const onFullscreenChange = () => (fullscreen.value = !!document.fullscreenElement)

function goBack() {
  save()
  if (window.history.state?.back) router.back()
  else void router.replace('/')
}

// ---------- overlay ----------
let hideTimer: ReturnType<typeof setTimeout> | undefined
function scheduleHide() {
  clearTimeout(hideTimer)
  hideTimer = setTimeout(() => {
    const focusedControl = root.value?.querySelector('.ctl:focus-visible, input:focus-visible')
    if (playing.value && !mediaError.value && scrub.value === null && !focusedControl && !tracksOpen.value) {
      overlay.value = false
    }
  }, 4000)
}
function showOverlay() {
  overlay.value = true
  scheduleHide()
}

// tap: toggle overlay (touch) / play-pause (mouse); double tap on a side: ±10 s
let lastTap = { t: 0, x: 0 }
let tapTimer: ReturnType<typeof setTimeout> | undefined
function onSurfaceTap(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement
  const w = el.clientWidth
  const x = e.offsetX
  const now = performance.now()
  if (tracksOpen.value) {
    toggleTracks()
    return
  }
  const isDouble = now - lastTap.t < 300 && Math.abs(x - lastTap.x) < 80
  lastTap = { t: now, x }
  clearTimeout(tapTimer)
  if (isDouble) {
    lastTap = { t: 0, x: 0 }
    if (x < w / 3) seekBy(-10, 'left')
    else if (x > (w * 2) / 3) seekBy(10, 'right')
    else void toggleFullscreen()
    return
  }
  tapTimer = setTimeout(() => {
    if (e.pointerType === 'mouse') togglePlay()
    else if (overlay.value) overlay.value = false
    else showOverlay()
  }, 260)
}

function onKey(e: KeyboardEvent) {
  if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return
  if (playback.external || playback.asking) return // a sheet owns the keyboard
  if (tracksOpen.value) {
    // the panel owns the arrows; Escape closes it
    if (e.key === 'Escape') {
      e.preventDefault()
      toggleTracks()
    }
    return
  }
  const t = e.target as HTMLElement
  const onControl = t.tagName === 'BUTTON' || t.tagName === 'A'
  const onRange = t.tagName === 'INPUT'
  switch (e.key) {
    case ' ':
    case 'k':
    case 'MediaPlayPause':
      if (onControl && e.key === ' ') return
      e.preventDefault()
      togglePlay()
      break
    case 'ArrowLeft':
      if (onRange || (onControl && overlay.value)) return showOverlay()
      e.preventDefault()
      seekBy(-10)
      break
    case 'ArrowRight':
      if (onRange || (onControl && overlay.value)) return showOverlay()
      e.preventDefault()
      seekBy(10)
      break
    case 'ArrowUp':
    case 'ArrowDown':
      showOverlay()
      break
    case 'f':
    case 'F':
      e.preventDefault()
      void toggleFullscreen()
      break
    case 'n':
    case 'N':
      playNext()
      break
    case 'Escape':
      if (!document.fullscreenElement) goBack()
      break
    default:
      if (e.key === 'Enter' && !onControl) togglePlay()
  }
}

// ---------- torrent stats while buffering / overlay ----------
const statsEnabled = computed(() => buffering.value || overlay.value)
usePolling(
  async () => {
    if (!file.value) return
    status.value = await torrentsApi.get(props.hash)
  },
  2000,
  statsEnabled,
)

// ---------- media session (lock screen / notification controls) ----------
function setupMediaSession() {
  if (!('mediaSession' in navigator)) return
  const poster = imageUrl(media.value?.posterPath, 'w500')
  navigator.mediaSession.metadata = new MediaMetadata({
    title: media.value ? `${media.value.title}` : name.value,
    artist: name.value,
    artwork: poster ? [{ src: poster, sizes: '500x750', type: 'image/jpeg' }] : [],
  })
  const set = (a: MediaSessionAction, h: MediaSessionActionHandler | null) => {
    try {
      navigator.mediaSession.setActionHandler(a, h)
    } catch {
      /* unsupported action */
    }
  }
  set('play', () => void video.value?.play())
  set('pause', () => video.value?.pause())
  set('seekbackward', () => seekBy(-10))
  set('seekforward', () => seekBy(30))
  set('nexttrack', next.value ? playNext : null)
}

onMounted(() => {
  void load()
  window.addEventListener('keydown', onKey)
  window.addEventListener('pagehide', onPageHide)
  document.addEventListener('fullscreenchange', onFullscreenChange)
  root.value?.focus()
})
onBeforeUnmount(() => {
  save()
  ctrl?.abort()
  clearInterval(saveTimer)
  clearTimeout(hideTimer)
  clearTimeout(hintTimer)
  clearTimeout(tapTimer)
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('pagehide', onPageHide)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  if (document.fullscreenElement) void document.exitFullscreen().catch(() => undefined)
  if ('mediaSession' in navigator) navigator.mediaSession.metadata = null
  const v = video.value
  if (v) {
    // stop the download immediately instead of waiting for GC
    v.pause()
    v.removeAttribute('src')
    v.load()
  }
})
</script>

<template>
  <div
    ref="root"
    class="player page"
    :class="{ 'player--idle': !overlay }"
    tabindex="-1"
    @pointermove="(e) => e.pointerType === 'mouse' && showOverlay()"
  >
    <video
      v-if="src && !loadError"
      ref="video"
      class="player__video"
      :src="src"
      preload="auto"
      playsinline
      autoplay
      @loadedmetadata="onLoadedMetadata"
      @timeupdate="onTime"
      @progress="updateBuffered"
      @durationchange="onTime"
      @play="onPlay"
      @pause="onPause"
      @loadeddata="checkPicture"
      @playing="(buffering = false), checkPicture()"
      @canplay="buffering = false"
      @waiting="buffering = true"
      @seeking="buffering = true"
      @seeked="buffering = false"
      @ended="onEnded"
      @error="onError"
    />

    <div class="player__surface" @pointerup="onSurfaceTap" />

    <div v-if="seekHint" class="player__hint" :class="`player__hint--${seekHint.side}`" aria-live="polite">
      {{ seekHint.text }}
    </div>

    <div v-if="busy && !overlay" class="player__buffering">
      <Spinner :size="44" />
      <div class="player__caption">
        <span>{{ busyText }}</span>
        <span v-if="statLine" class="small">{{ statLine }}</span>
      </div>
    </div>

    <div v-if="loadError" class="player__panel">
      <ErrorBanner :error="loadError" retry @retry="load" />
      <UiButton variant="neutral" icon="back" @click="goBack">{{ msg.common.back }}</UiButton>
    </div>

    <div v-else-if="mediaError" class="player__panel" role="alert">
      <Icon name="alert" :size="32" class="player__panel-icon" />
      <h2 class="player__panel-title">{{ msg.player.errorTitle }}</h2>
      <p class="player__panel-text">{{ msg.player.errorText }}</p>
      <div class="row-wrap player__panel-actions">
        <UiButton variant="primary" size="lg" icon="external" autofocus @click="openExternal">{{ msg.player.external }}</UiButton>
        <UiButton size="lg" icon="refresh" @click="retryMedia">{{ msg.player.tryAgain }}</UiButton>
      </div>
    </div>

    <div v-else-if="noPicture" class="player__panel" role="alert">
      <Icon name="alert" :size="32" class="player__panel-icon" />
      <h2 class="player__panel-title">{{ msg.player.noPictureTitle }}</h2>
      <p class="player__panel-text">{{ msg.player.noPictureText }}</p>
      <div class="row-wrap player__panel-actions">
        <UiButton variant="primary" size="lg" icon="external" autofocus @click="openExternal">{{ msg.player.external }}</UiButton>
        <UiButton size="lg" icon="audio" @click="listenOnly">{{ msg.player.audioOnly }}</UiButton>
      </div>
    </div>

    <TracksPanel
      v-if="tracksOpen && !loadError"
      :audio-options="tracks.audioOptions.value"
      :subtitle-options="tracks.subtitleOptions.value"
      :audio="tracks.audio.value"
      :subtitle="tracks.subtitle.value"
      :sub-loading="tracks.subLoading.value"
      :note="tracksNote"
      @audio="tracks.selectAudio"
      @subtitle="tracks.selectSubtitle"
      @close="toggleTracks"
    />

    <Transition name="fade">
      <div v-show="overlay && !loadError" class="player__overlay">
        <header class="player__top">
          <IconButton icon="back" :label="msg.common.back" class="ctl" @click="goBack" />
          <div class="player__titles">
            <div class="player__title ellipsis">{{ title }}</div>
            <div v-if="name && name !== title" class="player__file ellipsis">{{ name }}</div>
          </div>
          <span v-if="statLine" class="player__stat small">{{ statLine }}</span>
          <IconButton icon="external" :label="msg.player.external" class="ctl" @click="openExternal" />
        </header>

        <div v-if="!mediaError" class="player__center">
          <button type="button" class="ctl player__round" :aria-label="msg.player.back10" @click="seekBy(-10)">
            <Icon name="replay" :size="26" /><span class="player__round-label">10</span>
          </button>
          <button type="button" class="ctl player__round player__round--main" :aria-label="playing ? msg.player.pause : msg.player.play" @click="togglePlay">
            <Icon :name="playing ? 'pause' : 'play'" :size="34" />
            <Spinner v-if="busy" :size="86" class="player__ring" />
          </button>
          <button type="button" class="ctl player__round" :aria-label="msg.player.fwd30" @click="seekBy(30)">
            <Icon name="forward" :size="26" /><span class="player__round-label">30</span>
          </button>
          <div v-if="busy" class="player__caption">{{ busyText }}</div>
        </div>

        <footer class="player__bottom">
          <div class="player__seek">
            <span class="player__time">{{ formatDuration(shown) }}</span>
            <input
              class="ctl player__range"
              type="range"
              min="0"
              :max="duration || 0"
              step="1"
              :value="shown"
              :disabled="!duration"
              :aria-label="msg.player.seek"
              :aria-valuetext="`${formatDuration(shown)} / ${formatDuration(duration)}`"
              :style="{ '--played': `${pct(shown)}%`, '--buffered': `${pct(bufferedEnd)}%` }"
              @input="onScrub"
              @change="onScrubEnd"
            />
            <span class="player__time">{{ formatDuration(duration) }}</span>
          </div>
          <div class="player__buttons">
            <IconButton
              v-if="next"
              icon="next"
              :label="`${msg.player.next}: ${fileName(next.path)}`"
              class="ctl"
              @click="playNext"
            />
            <span class="spacer" />
            <IconButton
              icon="subtitles"
              :label="msg.player.tracks"
              class="ctl"
              :class="{ 'player__btn--on': tracksOpen }"
              :aria-expanded="tracksOpen"
              @click="toggleTracks"
            />
            <IconButton
              :icon="fullscreen ? 'fullscreenExit' : 'fullscreen'"
              :label="fullscreen ? msg.player.exitFullscreen : msg.player.fullscreen"
              class="ctl"
              @click="toggleFullscreen"
            />
          </div>
        </footer>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.player {
  background: #000;
  color: #fff;
  overflow: hidden;
  user-select: none;
  padding: 0;
}
.player:focus {
  outline: none;
}
.player--idle {
  cursor: none;
}
.player__video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
}
/* subtitles from the tracks panel */
.player__video::cue {
  font-family: system-ui, sans-serif;
  font-size: clamp(16px, 2.6vw, 30px);
  line-height: 1.3;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
}
.player__buttons :deep(.player__btn--on) {
  color: var(--c-accent);
}
.player__surface {
  position: absolute;
  inset: 0;
  touch-action: manipulation;
}
/* the spinner sits exactly at the centre; the caption hangs below it */
.player__buffering {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  pointer-events: none;
}
.player__caption {
  position: absolute;
  top: calc(100% + 18px);
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  font-size: 13px;
  pointer-events: none;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.8);
}
.player__ring {
  position: absolute;
  top: 50%;
  left: 50%;
  margin: -43px 0 0 -43px;
  pointer-events: none;
}
.player__hint {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  padding: 10px 16px;
  border-radius: var(--r-control);
  background: rgba(var(--rgb-bg), 0.75);
  font-weight: 700;
  pointer-events: none;
  z-index: 3;
}
.player__hint--left {
  left: 12%;
}
.player__hint--right {
  right: 12%;
}
.player__hint--center {
  left: 50%;
  transform: translate(-50%, -50%);
}
.player__overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  pointer-events: none;
  background: linear-gradient(rgba(0, 0, 0, 0.65), transparent 22%, transparent 70%, rgba(0, 0, 0, 0.75));
  z-index: 2;
}
.player__overlay > * {
  pointer-events: auto;
}
.player__top {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: calc(6px + var(--safe-top)) calc(8px + var(--safe-right)) 6px calc(8px + var(--safe-left));
}
.player__titles {
  flex: 1;
  min-width: 0;
}
.player__title {
  font-weight: 700;
  font-size: 14px;
}
.player__file {
  font-size: 11px;
  color: var(--c-muted);
}
.player__stat {
  white-space: nowrap;
}
.player__center {
  position: relative;
  align-self: center;
  display: flex;
  align-items: center;
  gap: clamp(20px, 8vw, 56px);
}
.player__round {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  border: none;
  background: rgba(var(--rgb-bg), 0.45);
  color: #fff;
}
.player__round:hover {
  background: rgba(var(--rgb-accent), var(--a-ghost-hover));
  color: var(--c-accent);
}
.player__round--main {
  width: 72px;
  height: 72px;
  background: rgba(var(--rgb-accent), 0.18);
  border: 1px solid rgba(var(--rgb-accent), var(--a-line));
  color: var(--c-accent);
}
.player__round-label {
  position: absolute;
  font-size: 8px;
  font-weight: 700;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -38%);
}
.player__bottom {
  padding: 0 calc(12px + var(--safe-right)) calc(4px + var(--safe-bottom)) calc(12px + var(--safe-left));
}
.player__seek {
  display: flex;
  align-items: center;
  gap: 10px;
}
.player__time {
  font-size: 12px;
  min-width: 52px;
  text-align: center;
}
.player__buttons {
  display: flex;
  align-items: center;
}
.player__buttons :deep(.icon-btn),
.player__top :deep(.icon-btn) {
  color: #fff;
}
.player__range {
  flex: 1;
  height: 28px;
  appearance: none;
  background: transparent;
  cursor: pointer;
}
.player__range::-webkit-slider-runnable-track {
  height: 4px;
  border-radius: 2px;
  background: linear-gradient(
    to right,
    var(--c-accent) var(--played),
    rgba(255, 255, 255, 0.4) var(--played),
    rgba(255, 255, 255, 0.4) var(--buffered),
    rgba(255, 255, 255, 0.15) var(--buffered)
  );
}
.player__range::-moz-range-track {
  height: 4px;
  border-radius: 2px;
  background: linear-gradient(
    to right,
    var(--c-accent) var(--played),
    rgba(255, 255, 255, 0.4) var(--played),
    rgba(255, 255, 255, 0.4) var(--buffered),
    rgba(255, 255, 255, 0.15) var(--buffered)
  );
}
.player__range::-webkit-slider-thumb {
  appearance: none;
  width: 16px;
  height: 16px;
  margin-top: -6px;
  border-radius: 50%;
  background: var(--c-accent);
  border: none;
}
.player__range::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--c-accent);
  border: none;
}
.player__range:disabled {
  opacity: 0.5;
}
.player__panel {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: min(520px, calc(100% - 32px));
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 24px;
  text-align: center;
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-radius: var(--r-dialog);
  z-index: 4;
}
.player__panel-icon {
  color: var(--c-warning);
}
.player__panel-title {
  margin: 0;
  font-size: 16px;
}
.player__panel-text {
  margin: 0;
  color: var(--c-muted);
  line-height: 1.5;
}
.player__panel-actions {
  justify-content: center;
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
