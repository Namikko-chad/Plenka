import type { TorrentFile } from '@/api/types'

const VIDEO_EXT = new Set([
  'mkv', 'mp4', 'avi', 'm4v', 'mov', 'ts', 'm2ts', 'mts', 'webm', 'wmv', 'mpg', 'mpeg', 'vob', 'flv',
])

export function fileName(path: string): string {
  const parts = path.split(/[\\/]/)
  return parts[parts.length - 1] ?? path
}

export function isVideo(path: string): boolean {
  const name = fileName(path)
  const dot = name.lastIndexOf('.')
  return dot > 0 && VIDEO_EXT.has(name.slice(dot + 1).toLowerCase())
}

const CHUNK = /(\d+|\D+)/g

/** Compares strings with numbers by value: "Episode 2" < "Episode 10". */
export function naturalCompare(a: string, b: string): number {
  const pa = a.toLowerCase().match(CHUNK) ?? []
  const pb = b.toLowerCase().match(CHUNK) ?? []
  const n = Math.min(pa.length, pb.length)
  for (let i = 0; i < n; i++) {
    const x = pa[i]!
    const y = pb[i]!
    const nx = /^\d/.test(x) ? Number(x) : NaN
    const ny = /^\d/.test(y) ? Number(y) : NaN
    let c: number
    if (!Number.isNaN(nx) && !Number.isNaN(ny)) c = nx - ny || x.length - y.length
    else c = x < y ? -1 : x > y ? 1 : 0
    if (c !== 0) return c
  }
  return pa.length - pb.length
}

/** Video files naturally sorted; all files when the torrent has no video. */
export function playableFiles(files: TorrentFile[]): TorrentFile[] {
  const videos = files.filter((f) => isVideo(f.path))
  const list = videos.length > 0 ? videos : [...files]
  return list.sort((a, b) => naturalCompare(a.path, b.path))
}

const SUBTITLE_EXT = new Set(['srt', 'vtt', 'ass', 'ssa'])
const AUDIO_EXT = new Set(['mka', 'ac3', 'eac3', 'dts', 'aac', 'flac', 'mp3', 'm4a', 'opus', 'ogg', 'oga', 'wav'])

export function extOf(path: string): string {
  const name = fileName(path)
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : ''
}

/** Text subtitles the built-in player can show (srt, vtt, ass/ssa). */
export function isSubtitle(path: string): boolean {
  return SUBTITLE_EXT.has(extOf(path))
}

/** Separate audio tracks (dubs) that ship next to the video. */
export function isAudio(path: string): boolean {
  return AUDIO_EXT.has(extOf(path))
}

/** File name without its extension, lower-cased: the key that pairs a video with its subtitles. */
function stem(path: string): string {
  const name = fileName(path).toLowerCase()
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(0, dot) : name
}

/**
 * Extra files of a torrent for one video (subtitles or audio): the ones named after
 * the video come first, then the rest in natural order.
 */
export function companionFiles(files: TorrentFile[], video: TorrentFile, kind: 'subtitle' | 'audio'): TorrentFile[] {
  const test = kind === 'subtitle' ? isSubtitle : isAudio
  const base = stem(video.path)
  const rank = (f: TorrentFile) => (stem(f.path).startsWith(base) ? 0 : 1)
  return files
    .filter((f) => test(f.path))
    .sort((a, b) => rank(a) - rank(b) || naturalCompare(a.path, b.path))
}

/** All files naturally sorted (the browse mode of the files screen). */
export function sortedFiles(files: TorrentFile[]): TorrentFile[] {
  return [...files].sort((a, b) => naturalCompare(a.path, b.path))
}
