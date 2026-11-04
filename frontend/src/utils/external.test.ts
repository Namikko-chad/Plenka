import { describe, expect, it } from 'vitest'
import {
  androidIntentUrl,
  detectPlatform,
  externalLinks,
  m3uPath,
  playlistPath,
  streamPath,
  vlcIosUrl,
} from './external'

const ORIGIN = 'http://192.168.88.2:8095'

describe('stream URLs', () => {
  it('builds /api/stream/{hash}/{fileId}/{encoded name} and the playlist URLs', () => {
    expect(streamPath('Серия 1 [1080p].mkv', 'abc', 3)).toBe(
      '/api/stream/abc/3/%D0%A1%D0%B5%D1%80%D0%B8%D1%8F%201%20%5B1080p%5D.mkv',
    )
    expect(m3uPath('abc', 2)).toBe('/api/torrents/abc/playlist.m3u?file=2')
    expect(playlistPath('abc')).toBe('/api/torrents/abc/playlist.m3u')
  })
  it('encodes slashes in names (nested torrent paths are passed as the base name)', () => {
    expect(streamPath('a/b.mkv', 'h', 1)).toBe('/api/stream/h/1/a%2Fb.mkv')
  })
})

describe('external player links', () => {
  const links = externalLinks(ORIGIN, 'Film.mkv', 'abc', 1, 'Фильм')

  it('builds absolute URLs including the host', () => {
    expect(links.stream).toBe(`${ORIGIN}/api/stream/abc/1/Film.mkv`)
    expect(links.m3u).toBe(`${ORIGIN}/api/torrents/abc/playlist.m3u?file=1`)
    expect(links.playlist).toBe(`${ORIGIN}/api/torrents/abc/playlist.m3u`)
  })
  it('targets VLC on Android, and a generic chooser without package', () => {
    expect(links.vlcAndroid).toBe(
      `intent:${links.stream}#Intent;package=org.videolan.vlc;type=video/*;S.title=%D0%A4%D0%B8%D0%BB%D1%8C%D0%BC;end`,
    )
    expect(links.chooserAndroid).not.toContain('package=')
    expect(links.chooserAndroid.startsWith(`intent:${ORIGIN}/api/stream/`)).toBe(true)
    expect(androidIntentUrl('http://h/x')).toBe('intent:http://h/x#Intent;type=video/*;end')
  })
  it('uses the VLC x-callback scheme on iOS with an encoded URL', () => {
    expect(vlcIosUrl('http://h/s?a=1&b=2')).toBe('vlc-x-callback://x-callback-url/stream?url=http%3A%2F%2Fh%2Fs%3Fa%3D1%26b%3D2')
  })
})

describe('detectPlatform', () => {
  it('detects Android, iOS (incl. iPadOS) and others', () => {
    expect(detectPlatform('Mozilla/5.0 (Linux; Android 14; Pixel 8)')).toBe('android')
    expect(detectPlatform('Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)')).toBe('ios')
    expect(detectPlatform('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 5)).toBe('ios')
    expect(detectPlatform('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 0)).toBe('other')
    expect(detectPlatform('Mozilla/5.0 (X11; Linux x86_64)')).toBe('other')
  })
})
