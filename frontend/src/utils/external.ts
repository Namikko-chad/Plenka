// External-player links. Browsers can't decode many MKV/AC3/DTS streams, so the
// player offers VLC / any player via these URLs. Pure functions — see external.test.ts.

export type Platform = 'android' | 'ios' | 'other'

export function detectPlatform(ua: string, maxTouchPoints = 0): Platform {
  if (/android/i.test(ua)) return 'android'
  if (/iphone|ipad|ipod/i.test(ua)) return 'ios'
  // iPadOS reports itself as Macintosh
  if (/macintosh/i.test(ua) && maxTouchPoints > 1) return 'ios'
  return 'other'
}

const enc = encodeURIComponent

/** Relative stream URL for the browser `<video>` element; the name is only for players. */
export function streamPath(name: string, hash: string, fileId: number): string {
  return `/api/stream/${enc(hash)}/${fileId}/${enc(name)}`
}

/** One-file m3u playlist (players that take playlists, e.g. iOS/desktop VLC). */
export function m3uPath(hash: string, fileId: number): string {
  return `/api/torrents/${enc(hash)}/playlist.m3u?file=${fileId}`
}

/** Whole-torrent playlist. */
export function playlistPath(hash: string): string {
  return `/api/torrents/${enc(hash)}/playlist.m3u`
}

function absoluteUrl(path: string, origin: string): string {
  return new URL(path, origin).toString()
}

/**
 * Android intent URL. With `pkg` it targets that app (VLC); without it Android shows
 * the "open with" chooser for any video player.
 */
export function androidIntentUrl(absUrl: string, opts: { pkg?: string; title?: string } = {}): string {
  const parts = ['Intent']
  if (opts.pkg) parts.push(`package=${opts.pkg}`)
  parts.push('type=video/*')
  if (opts.title) parts.push(`S.title=${encodeURIComponent(opts.title)}`)
  parts.push('end')
  return `intent:${absUrl}#${parts.join(';')}`
}

const VLC_ANDROID = 'org.videolan.vlc'

export function vlcIosUrl(absUrl: string): string {
  return `vlc-x-callback://x-callback-url/stream?url=${encodeURIComponent(absUrl)}`
}

export interface ExternalLinks {
  stream: string
  m3u: string
  playlist: string
  vlcAndroid: string
  chooserAndroid: string
  vlcIos: string
}

export function externalLinks(origin: string, name: string, hash: string, fileId: number, title?: string): ExternalLinks {
  const stream = absoluteUrl(streamPath(name, hash, fileId), origin)
  return {
    stream,
    m3u: absoluteUrl(m3uPath(hash, fileId), origin),
    playlist: absoluteUrl(playlistPath(hash), origin),
    vlcAndroid: androidIntentUrl(stream, { pkg: VLC_ANDROID, title }),
    chooserAndroid: androidIntentUrl(stream, { title }),
    vlcIos: vlcIosUrl(stream),
  }
}
