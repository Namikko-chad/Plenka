// Text subtitles for the built-in player: SRT, WebVTT and ASS/SSA are parsed into
// plain cues that the player adds to the <video> as a TextTrack. Pure functions —
// see subtitles.test.ts.

export interface Cue {
  start: number // seconds
  end: number
  text: string
}

/** Decodes subtitle bytes: UTF-8 (with or without BOM), else Windows-1251 (typical for Russian SRT). */
export function decodeSubtitles(buf: ArrayBuffer): string {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(buf).replace(/^﻿/, '')
  } catch {
    return new TextDecoder('windows-1251').decode(buf)
  }
}

const TIME = /(?:(\d+):)?(\d{1,2}):(\d{1,2})[.,](\d{1,3})/

function seconds(s: string): number | null {
  const m = TIME.exec(s)
  if (!m) return null
  const [, h, mi, se, frac] = m
  return Number(h ?? 0) * 3600 + Number(mi) * 60 + Number(se) + Number(frac!.padEnd(3, '0')) / 1000
}

/** SRT and WebVTT share the shape «start --> end» + text lines. */
function parseTimed(src: string): Cue[] {
  const out: Cue[] = []
  for (const block of src.replace(/\r\n?/g, '\n').split(/\n{2,}/)) {
    const lines = block.split('\n')
    const i = lines.findIndex((l) => l.includes('-->'))
    if (i < 0) continue
    const [a, b] = lines[i]!.split('-->')
    const start = seconds(a ?? '')
    const end = seconds(b ?? '')
    if (start === null || end === null) continue
    const text = lines
      .slice(i + 1)
      .join('\n')
      .replace(/<(?!\/?[biu]>)[^>]*>/gi, '') // keep only <b>/<i>/<u>
      .replace(/\{\\[^}]*\}/g, '')
      .trim()
    if (text) out.push({ start, end, text })
  }
  return out
}

/** ASS/SSA: the Dialogue lines of [Events], styling overrides dropped. */
function parseAss(src: string): Cue[] {
  const out: Cue[] = []
  let format: string[] = []
  for (const raw of src.replace(/\r\n?/g, '\n').split('\n')) {
    const line = raw.trim()
    if (/^format:/i.test(line)) {
      format = line
        .slice(7)
        .split(',')
        .map((f) => f.trim().toLowerCase())
      continue
    }
    if (!/^dialogue:/i.test(line) || !format.length) continue
    const fields = line.slice(9).split(',')
    const textAt = format.indexOf('text')
    if (textAt < 0) continue
    const get = (k: string) => fields[format.indexOf(k)]?.trim() ?? ''
    const start = seconds(get('start'))
    const end = seconds(get('end'))
    const text = fields
      .slice(textAt)
      .join(',')
      .replace(/\{[^}]*\}/g, '')
      .replace(/\\[Nn]/g, '\n')
      .replace(/\\h/g, ' ')
      .trim()
    if (start !== null && end !== null && text) out.push({ start, end, text })
  }
  return out.sort((a, b) => a.start - b.start)
}

/** Cues of a subtitle file by its extension (srt, vtt, ass, ssa). */
export function parseSubtitles(src: string, ext: string): Cue[] {
  return ext === 'ass' || ext === 'ssa' ? parseAss(src) : parseTimed(src)
}

const LANGS: [RegExp, string][] = [
  [/\b(rus?|russian|рус)/i, 'ru'],
  [/\b(eng?|english|англ)/i, 'en'],
  [/\b(ukr?|ukrainian|укр)/i, 'uk'],
]

/** Best-effort language code from a file name, '' when unknown. */
export function guessLanguage(name: string): string {
  const n = name.replace(/[._\-[\]()]/g, ' ')
  for (const [re, code] of LANGS) if (re.test(n)) return code
  return ''
}
