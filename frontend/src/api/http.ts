// Minimal JSON fetch: same-origin relative URLs, {error} handling, timeouts, abort.

export type ErrorKind = 'http' | 'network' | 'timeout' | 'abort'

export class ApiError extends Error {
  constructor(
    message: string,
    readonly kind: ErrorKind,
    readonly status: number,
    readonly url: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  /** Serialized as JSON unless it is FormData. */
  body?: unknown
  query?: Record<string, string | number | boolean | null | undefined>
  signal?: AbortSignal
  /** ms; 0 disables. */
  timeout?: number
  /** Survive page unload (progress save on pagehide). */
  keepalive?: boolean
}

const DEFAULT_TIMEOUT = 20_000

export function withQuery(url: string, query?: RequestOptions['query']): string {
  if (!query) return url
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && v !== '') p.set(k, String(v))
  }
  const s = p.toString()
  return s ? `${url}${url.includes('?') ? '&' : '?'}${s}` : url
}

export function isAbort(e: unknown): boolean {
  return e instanceof ApiError && e.kind === 'abort'
}

export async function request<T>(url: string, opts: RequestOptions = {}): Promise<T> {
  const full = withQuery(url, opts.query)
  const ctrl = new AbortController()
  const timeout = opts.timeout ?? DEFAULT_TIMEOUT
  let timedOut = false
  const timer = timeout > 0 ? setTimeout(() => ((timedOut = true), ctrl.abort()), timeout) : undefined
  const onAbort = () => ctrl.abort()
  if (opts.signal) {
    if (opts.signal.aborted) ctrl.abort()
    else opts.signal.addEventListener('abort', onAbort, { once: true })
  }

  const headers: Record<string, string> = { Accept: 'application/json' }
  let body: BodyInit | undefined
  if (opts.body instanceof FormData) body = opts.body
  else if (opts.body !== undefined) {
    body = JSON.stringify(opts.body)
    headers['Content-Type'] = 'application/json'
  }

  let res: Response
  try {
    res = await fetch(full, {
      method: opts.method ?? (body ? 'POST' : 'GET'),
      headers,
      body,
      signal: ctrl.signal,
      keepalive: opts.keepalive,
    })
  } catch {
    if (timedOut) throw new ApiError('timeout', 'timeout', 0, full)
    if (ctrl.signal.aborted) throw new ApiError('aborted', 'abort', 0, full)
    throw new ApiError('network', 'network', 0, full)
  } finally {
    clearTimeout(timer)
    opts.signal?.removeEventListener('abort', onAbort)
  }

  const text = await res.text().catch(() => '')
  let data: unknown = undefined
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = text
    }
  }
  if (!res.ok) {
    const msg =
      data && typeof data === 'object' && typeof (data as { error?: unknown }).error === 'string'
        ? (data as { error: string }).error
        : ''
    throw new ApiError(msg, 'http', res.status, full)
  }
  return data as T
}
