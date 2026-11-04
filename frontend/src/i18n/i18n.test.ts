import { describe, expect, it } from 'vitest'
import { en } from './en'
import { resolveLocale } from './index'
import { ru } from './ru'

/** Key paths of a message tree: `prefs.title`, `trailers.types.Trailer`, … */
function keys(o: object, prefix = ''): string[] {
  return Object.entries(o).flatMap(([k, v]) =>
    v && typeof v === 'object' ? keys(v, `${prefix}${k}.`) : [`${prefix}${k}`],
  )
}

describe('i18n', () => {
  it('follows the browser, falls back to English, and honours an explicit choice', () => {
    expect(resolveLocale('system', ['ru-RU', 'en-US'])).toBe('ru')
    expect(resolveLocale('system', ['de-DE', 'en-GB'])).toBe('en')
    expect(resolveLocale('system', ['de-DE'])).toBe('en')
    expect(resolveLocale('system', [])).toBe('en')
    expect(resolveLocale('en', ['ru-RU'])).toBe('en')
    expect(resolveLocale('ru', ['en-US'])).toBe('ru')
  })

  it('Russian has exactly the keys of the English template', () => {
    expect(keys(ru).sort()).toEqual(keys(en).sort())
  })
})
