import type { ConfigValue, IndexerSetting } from '@/api/types'

// The contract types indexer config values loosely (strings in the example; Cardigann
// definitions use booleans for checkboxes and lists for multi-selects). The form works
// on normalized values and writes each field back in the shape the server sent.

export type FormValue = string | boolean | string[]

export const isInfo = (type: string) => type.startsWith('info')

const truthy = (v: unknown) => v === true || v === 'true' || v === '1' || v === 'on'

function toList(v: unknown): string[] {
  if (Array.isArray(v)) return v.map(String)
  if (typeof v === 'string') return v.split(',').map((s) => s.trim()).filter(Boolean)
  return []
}

/** Saved value (or default) → form value. Passwords start empty: "" keeps the stored one. */
export function formValueOf(s: IndexerSetting, saved: ConfigValue | undefined): FormValue {
  const raw = saved ?? s.default
  switch (s.type) {
    case 'checkbox':
      return truthy(raw)
    case 'multi-select':
      return toList(raw)
    case 'password':
      return typeof saved === 'string' ? saved : ''
    default:
      return raw == null ? '' : String(raw)
  }
}

/** Form value → wire value, matching the type of what the server sent (or the default). */
export function toWireValue(s: IndexerSetting, v: FormValue, saved: ConfigValue | undefined): ConfigValue {
  const sample = saved ?? s.default
  if (s.type === 'checkbox') {
    const b = v === true
    return typeof sample === 'string' ? String(b) : b
  }
  if (s.type === 'multi-select') {
    const list = Array.isArray(v) ? v : toList(v)
    return typeof sample === 'string' ? list.join(',') : list
  }
  return typeof v === 'string' ? v : String(v)
}
