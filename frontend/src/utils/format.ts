import { msg } from '@/i18n'

/** Bytes → «1.4 GB» in the UI language. One decimal below 100, none above or for bytes. */
export function formatBytes(bytes: number | null | undefined): string {
  let v = Number(bytes) || 0
  if (v < 0) v = 0
  let i = 0
  const units = msg.format.units.split(' ')
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  const digits = v >= 100 || i === 0 ? 0 : 1
  return `${v.toFixed(digits)} ${units[i]}`
}

export function formatSpeed(bytesPerSec: number | null | undefined): string {
  return msg.format.perSecond(formatBytes(bytesPerSec))
}

/** Seconds → «1:02:03» or «02:03». */
export function formatDuration(totalSec: number): string {
  const s = Math.max(0, Math.floor(Number.isFinite(totalSec) ? totalSec : 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  const mm = String(m).padStart(2, '0')
  const ss = String(sec).padStart(2, '0')
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`
}

const pad = (n: number) => String(n).padStart(2, '0')

function toDate(v: string | number | Date | null | undefined): Date | null {
  if (v == null || v === '') return null
  const d = v instanceof Date ? v : new Date(v)
  return Number.isNaN(d.getTime()) || d.getTime() <= 0 ? null : d
}

/** Server UTC timestamp → browser-local «YYYY-MM-DD HH:MM:SS». */
export function formatDateTime(v: string | number | Date | null | undefined): string {
  const d = toDate(v)
  if (!d) return ''
  return `${formatDate(d)} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** Browser-local «YYYY-MM-DD». */
export function formatDate(v: string | number | Date | null | undefined): string {
  const d = toDate(v)
  if (!d) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function plural(n: number, one: string, few: string, many: string): string {
  const m10 = n % 10
  const m100 = n % 100
  if (m10 === 1 && m100 !== 11) return one
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few
  return many
}
