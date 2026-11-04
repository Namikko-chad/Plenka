import { computed, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { torrentsApi } from '@/api/torrents'
import type { TorrentFile } from '@/api/types'
import { msg } from '@/i18n'
import { streamPath } from '@/utils/external'
import { companionFiles, extOf, fileName } from '@/utils/files'
import { decodeSubtitles, guessLanguage, parseSubtitles } from '@/utils/subtitles'

export interface TrackOption {
  key: string
  label: string
  hint?: string
}

// HTMLMediaElement.audioTracks exists in Safari (and Chromium behind a flag); TS has no types for it.
interface MediaAudioTrack {
  enabled: boolean
  label: string
  language: string
}
type AudioTrackList = ArrayLike<MediaAudioTrack>
const audioTracksOf = (v: HTMLVideoElement) => (v as HTMLVideoElement & { audioTracks?: AudioTrackList }).audioTracks

/** Max drift between the video and a separate audio file before the audio is re-synced, s. */
const MAX_DRIFT = 0.4

/**
 * Audio and subtitle choice of the built-in player: tracks the browser exposes inside
 * the file, plus subtitle (srt/vtt/ass) and audio (mka/ac3/…) files of the same
 * torrent. A separate audio file plays in a hidden <audio> kept in sync with the muted video.
 */
export function usePlayerTracks(opts: {
  video: Ref<HTMLVideoElement | null>
  hash: () => string
  file: Ref<TorrentFile | null>
  files: Ref<TorrentFile[]>
  onError: (text: string) => void
}) {
  const embeddedAudio = ref<TrackOption[]>([])
  const embeddedSubs = ref<TrackOption[]>([])
  const audioListSupported = ref(true)
  const audio = ref('orig')
  const subtitle = ref('off')
  const subLoading = ref(false)

  let embeddedSubRefs: TextTrack[] = []
  const loadedSubs = new Map<number, TextTrack>()
  const ext = new Audio()
  ext.preload = 'auto'
  let external = false

  const subFiles = computed(() => (opts.file.value ? companionFiles(opts.files.value, opts.file.value, 'subtitle') : []))
  const audioFiles = computed(() => (opts.file.value ? companionFiles(opts.files.value, opts.file.value, 'audio') : []))

  const fileOption = (f: TorrentFile, hint: string): TrackOption => ({ key: `file:${f.id}`, label: fileName(f.path), hint })

  const audioOptions = computed<TrackOption[]>(() => [
    ...(embeddedAudio.value.length ? embeddedAudio.value : [{ key: 'orig', label: msg.player.original }]),
    ...audioFiles.value.map((f) => fileOption(f, msg.player.externalAudio)),
  ])
  const subtitleOptions = computed<TrackOption[]>(() => [
    { key: 'off', label: msg.player.off },
    ...embeddedSubs.value,
    ...subFiles.value.map((f) => fileOption(f, guessLanguage(fileName(f.path)).toUpperCase())),
  ])

  /** Reads the tracks the browser found in the file. */
  function scan() {
    const v = opts.video.value
    if (!v) return
    const at = audioTracksOf(v)
    audioListSupported.value = !!at
    if (at && at.length > 0) {
      const list: TrackOption[] = []
      for (let i = 0; i < at.length; i++) {
        const t = at[i]!
        list.push({ key: `emb:${i}`, label: t.label || msg.player.track(i + 1), hint: t.language.toUpperCase() })
        if (t.enabled && !external) audio.value = `emb:${i}`
      }
      embeddedAudio.value = list
    } else {
      embeddedAudio.value = []
    }
    const ours = new Set(loadedSubs.values())
    embeddedSubRefs = Array.from(v.textTracks).filter(
      (t) => !ours.has(t) && (t.kind === 'subtitles' || t.kind === 'captions'),
    )
    embeddedSubs.value = embeddedSubRefs.map((t, i) => ({
      key: `emb:${i}`,
      label: t.label || msg.player.track(i + 1),
      hint: t.language.toUpperCase(),
    }))
  }

  async function loadSubtitle(f: TorrentFile, v: HTMLVideoElement): Promise<TextTrack> {
    const cached = loadedSubs.get(f.id)
    if (cached) return cached
    const text = decodeSubtitles(await torrentsApi.fileBytes(opts.hash(), f.id))
    const name = fileName(f.path)
    const track = v.addTextTrack('subtitles', name, guessLanguage(name))
    for (const c of parseSubtitles(text, extOf(f.path))) track.addCue(new VTTCue(c.start, c.end, c.text))
    loadedSubs.set(f.id, track)
    return track
  }

  async function selectSubtitle(key: string) {
    const v = opts.video.value
    if (!v) return
    subtitle.value = key
    for (const t of Array.from(v.textTracks)) t.mode = 'disabled'
    if (key.startsWith('emb:')) {
      const t = embeddedSubRefs[Number(key.slice(4))]
      if (t) t.mode = 'showing'
      return
    }
    if (!key.startsWith('file:')) return
    const f = subFiles.value.find((x) => x.id === Number(key.slice(5)))
    if (!f) return
    subLoading.value = true
    try {
      const track = await loadSubtitle(f, v)
      if (subtitle.value === key) track.mode = 'showing'
    } catch {
      if (subtitle.value === key) subtitle.value = 'off'
      opts.onError(msg.player.subsError)
    } finally {
      subLoading.value = false
    }
  }

  function stopExternal() {
    if (!external) return
    external = false
    ext.pause()
    ext.removeAttribute('src')
    ext.load()
    if (opts.video.value) opts.video.value.muted = false
  }

  function selectAudio(key: string) {
    const v = opts.video.value
    if (!v) return
    if (key.startsWith('file:')) {
      const f = audioFiles.value.find((x) => x.id === Number(key.slice(5)))
      if (!f) return
      audio.value = key
      external = true
      ext.src = streamPath(fileName(f.path), opts.hash(), f.id)
      ext.currentTime = v.currentTime
      ext.playbackRate = v.playbackRate
      v.muted = true
      if (!v.paused) void ext.play().catch(() => undefined)
      return
    }
    stopExternal()
    audio.value = key
    const at = audioTracksOf(v)
    if (key.startsWith('emb:') && at) {
      const idx = Number(key.slice(4))
      for (let i = 0; i < at.length; i++) at[i]!.enabled = i === idx
    }
  }

  // ---------- keeping a separate audio file in sync with the video ----------
  const sync: Record<string, (this: HTMLVideoElement) => void> = {
    loadedmetadata: scan,
    play() {
      if (external) void ext.play().catch(() => undefined)
    },
    playing() {
      if (external) void ext.play().catch(() => undefined)
    },
    pause() {
      if (external) ext.pause()
    },
    waiting() {
      if (external) ext.pause()
    },
    seeking() {
      if (external) ext.currentTime = this.currentTime
    },
    ratechange() {
      if (external) ext.playbackRate = this.playbackRate
    },
    timeupdate() {
      if (external && !this.paused && Math.abs(ext.currentTime - this.currentTime) > MAX_DRIFT) {
        ext.currentTime = this.currentTime
      }
    },
  }
  ext.addEventListener('error', () => {
    if (!external || !ext.getAttribute('src')) return
    opts.onError(msg.player.audioError)
    selectAudio(embeddedAudio.value[0]?.key ?? 'orig')
  })

  watch(
    opts.video,
    (v, old) => {
      for (const [ev, fn] of Object.entries(sync)) {
        old?.removeEventListener(ev, fn)
        v?.addEventListener(ev, fn)
      }
    },
    { immediate: true },
  )

  onBeforeUnmount(() => {
    stopExternal()
    for (const [ev, fn] of Object.entries(sync)) opts.video.value?.removeEventListener(ev, fn)
  })

  return {
    audioOptions,
    subtitleOptions,
    audio,
    subtitle,
    subLoading,
    audioListSupported,
    scan,
    selectAudio,
    selectSubtitle,
  }
}
