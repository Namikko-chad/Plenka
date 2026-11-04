import { en } from './en'
import { ru } from './ru'

/** Same shape as the English template, with plain `string` instead of literal types. */
type Widen<T> = T extends string
  ? string
  : T extends (...args: infer A) => infer R
    ? (...args: A) => Widen<R>
    : T extends object
      ? { readonly [K in keyof T]: Widen<T[K]> }
      : T
export type Messages = Widen<typeof en>

export type Locale = 'en' | 'ru'
/** What the user chose in Settings; `system` follows the browser. */
export type LanguagePref = 'system' | Locale

export const LOCALES: readonly Locale[] = ['en', 'ru']
const KEY = 'plenka.language'
const catalogs: Record<Locale, Messages> = { en, ru }

/** The first supported browser language, English otherwise. */
export function resolveLocale(pref: LanguagePref, browser: readonly string[]): Locale {
  if (pref !== 'system') return pref
  for (const tag of browser) {
    const code = tag.toLowerCase().split('-')[0] ?? ''
    if ((LOCALES as readonly string[]).includes(code)) return code as Locale
  }
  return 'en'
}

function readPref(): LanguagePref {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'en' || v === 'ru') return v
  } catch {
    /* storage unavailable */
  }
  return 'system'
}

function browserLanguages(): readonly string[] {
  if (typeof navigator === 'undefined') return []
  return navigator.languages?.length ? navigator.languages : [navigator.language]
}

export const languagePref: LanguagePref = readPref()
export const locale: Locale = resolveLocale(languagePref, browserLanguages())

/** TMDB `language` for the catalog: titles and overviews follow the interface language. */
export const tmdbLanguage: string = { en: 'en-US', ru: 'ru-RU' }[locale]

/** User-facing strings in the current language. */
export const msg: Messages = catalogs[locale]

/** Saves the choice and reloads, so every screen is rebuilt in the new language. */
export function setLanguage(pref: LanguagePref): void {
  try {
    if (pref === 'system') localStorage.removeItem(KEY)
    else localStorage.setItem(KEY, pref)
  } catch {
    /* storage unavailable: the choice lasts until reload only */
  }
  location.reload()
}
