import { describe, expect, it } from 'vitest'
import type { IndexerSetting } from '@/api/types'
import { formValueOf, isInfo, toWireValue } from './indexerForm'

const s = (type: string, def?: unknown): IndexerSetting => ({ name: 'x', type, label: 'X', default: def })

describe('indexer form values', () => {
  it('detects info variants', () => {
    expect(isInfo('info')).toBe(true)
    expect(isInfo('info_cookie')).toBe(true)
    expect(isInfo('text')).toBe(false)
  })
  it('normalizes checkboxes from bool or string and writes back the same shape', () => {
    expect(formValueOf(s('checkbox', false), 'true')).toBe(true)
    expect(formValueOf(s('checkbox', true), undefined)).toBe(true)
    expect(toWireValue(s('checkbox', false), true, 'false')).toBe('true')
    expect(toWireValue(s('checkbox', false), true, undefined)).toBe(true)
  })
  it('normalizes multi-selects from arrays or comma strings', () => {
    expect(formValueOf(s('multi-select', ['1']), '1, 2')).toEqual(['1', '2'])
    expect(toWireValue(s('multi-select', []), ['1', '2'], '1')).toBe('1,2')
    expect(toWireValue(s('multi-select', []), ['3'], ['1'])).toEqual(['3'])
  })
  it('starts passwords empty so an untouched field keeps the stored secret', () => {
    expect(formValueOf(s('password'), '')).toBe('')
    expect(toWireValue(s('password'), '', '')).toBe('')
  })
  it('uses defaults for text/select', () => {
    expect(formValueOf(s('select', '0'), undefined)).toBe('0')
    expect(formValueOf(s('text'), undefined)).toBe('')
  })
})
