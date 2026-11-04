import { describe, expect, it } from 'vitest'
import { companionFiles, fileName, isAudio, isSubtitle, isVideo, naturalCompare, playableFiles, sortedFiles } from './files'

describe('naturalCompare', () => {
  it('orders numbers by value', () => {
    const list = ['Серия 10.mkv', 'Серия 2.mkv', 'Серия 1.mkv']
    expect(list.sort(naturalCompare)).toEqual(['Серия 1.mkv', 'Серия 2.mkv', 'Серия 10.mkv'])
  })
  it('handles season/episode paths and case', () => {
    const list = ['Show/S01E10.mkv', 'show/s01e09.mkv', 'Show/S02E01.mkv', 'Show/S01E01.mkv']
    expect(list.sort(naturalCompare)).toEqual(['Show/S01E01.mkv', 'show/s01e09.mkv', 'Show/S01E10.mkv', 'Show/S02E01.mkv'])
  })
  it('puts shorter prefix first and keeps leading zeros stable', () => {
    expect(naturalCompare('a', 'a1')).toBeLessThan(0)
    expect(naturalCompare('ep01', 'ep1')).toBeGreaterThan(0)
    expect(naturalCompare('x', 'x')).toBe(0)
  })
})

describe('video detection', () => {
  it('recognises video extensions case-insensitively', () => {
    for (const n of ['a.mkv', 'b.MP4', 'dir/c.m2ts', 'd.webm', 'e.vob']) expect(isVideo(n)).toBe(true)
    for (const n of ['a.srt', 'b.ac3', 'mkv', 'c.mkv.part', 'readme.txt']) expect(isVideo(n)).toBe(false)
  })
  it('takes the last path segment', () => {
    expect(fileName('Show/Season 1/E01.mkv')).toBe('E01.mkv')
    expect(fileName('E01.mkv')).toBe('E01.mkv')
  })
})

describe('playableFiles', () => {
  const f = (id: number, path: string) => ({ id, path, length: 1 })
  it('keeps only videos, naturally sorted', () => {
    const r = playableFiles([f(1, 'S/E10.mkv'), f(2, 'S/E2.mkv'), f(3, 'S/sub.srt'), f(4, 'S/E1.mkv')])
    expect(r.map((x) => x.id)).toEqual([4, 2, 1])
  })
  it('falls back to all files when there is no video', () => {
    const r = playableFiles([f(2, 'b.flac'), f(1, 'a.flac')])
    expect(r.map((x) => x.id)).toEqual([1, 2])
  })
})

describe('companion files', () => {
  const f = (id: number, path: string) => ({ id, path, length: 1 })
  const video = f(1, 'Movie/Movie.2020.mkv')
  const files = [video, f(2, 'Movie/Subs/Other.srt'), f(3, 'Movie/Movie.2020.rus.srt'), f(4, 'Movie/Movie.2020.ac3'), f(5, 'Movie/x.txt')]
  it('puts the files named after the video first', () => {
    expect(companionFiles(files, video, 'subtitle').map((x) => x.id)).toEqual([3, 2])
    expect(companionFiles(files, video, 'audio').map((x) => x.id)).toEqual([4])
  })
  it('detects kinds and sorts everything', () => {
    expect(isSubtitle('a.ASS')).toBe(true)
    expect(isAudio('a.mka')).toBe(true)
    expect(isAudio('a.mkv')).toBe(false)
    expect(sortedFiles(files).map((x) => x.id)).toEqual([4, 1, 3, 2, 5])
  })
})
