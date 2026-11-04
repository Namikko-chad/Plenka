import { describe, expect, it } from 'vitest'
import { decodeSubtitles, guessLanguage, parseSubtitles } from './subtitles'

describe('parseSubtitles', () => {
  it('parses SRT with CRLF, tags and multi-line text', () => {
    const srt = '1\r\n00:00:01,500 --> 00:00:03,000\r\n<font color="red">Привет</font>\r\n<i>мир</i>\r\n\r\n2\r\n01:02:03,04 --> 01:02:04,000\r\nДа\r\n'
    expect(parseSubtitles(srt, 'srt')).toEqual([
      { start: 1.5, end: 3, text: 'Привет\n<i>мир</i>' },
      { start: 3723.04, end: 3724, text: 'Да' },
    ])
  })
  it('parses WebVTT with short timestamps and settings', () => {
    const vtt = 'WEBVTT\n\nintro\n00:05.000 --> 00:06.250 line:90%\nHello\n'
    expect(parseSubtitles(vtt, 'vtt')).toEqual([{ start: 5, end: 6.25, text: 'Hello' }])
  })
  it('parses ASS dialogue with commas and overrides', () => {
    const ass = [
      '[Events]',
      'Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text',
      'Dialogue: 0,0:00:10.00,0:00:12.50,Default,,0,0,0,,{\\i1}Раз, два{\\i0}\\Nтри',
      'Dialogue: 0,0:00:02.00,0:00:03.00,Default,,0,0,0,,Первая',
    ].join('\n')
    expect(parseSubtitles(ass, 'ass')).toEqual([
      { start: 2, end: 3, text: 'Первая' },
      { start: 10, end: 12.5, text: 'Раз, два\nтри' },
    ])
  })
})

describe('decodeSubtitles', () => {
  it('reads UTF-8 and strips the BOM', () => {
    const buf = new TextEncoder().encode('﻿Ёж').buffer
    expect(decodeSubtitles(buf as ArrayBuffer)).toBe('Ёж')
  })
  it('falls back to Windows-1251', () => {
    const buf = new Uint8Array([0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2]).buffer // "Привет" in cp1251
    expect(decodeSubtitles(buf)).toBe('Привет')
  })
})

describe('guessLanguage', () => {
  it('finds common markers', () => {
    expect(guessLanguage('Movie.RUS.Forced.srt')).toBe('ru')
    expect(guessLanguage('Movie_eng.srt')).toBe('en')
    expect(guessLanguage('Movie.srt')).toBe('')
  })
})
