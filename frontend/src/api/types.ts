// Wire types from docs/en/31-rest-api.md.

export type MediaType = 'movie' | 'tv'

/** Media snapshot stored in history entries and torrent `data`. */
export interface Media {
  id: number
  type: MediaType
  title: string
  originalTitle: string
  year: number | null
  posterPath: string | null
  backdropPath: string | null
  overview: string
  rating: number
}

export interface MediaDetails extends Media {
  genres: string[]
  runtime: number | null
  seasons: number | null
  episodes: number | null
  tagline: string | null
  cast: string[]
  countries: string[]
}

export interface ServerInfo {
  service: string
  version: string
  tmdb: boolean
  torrents: boolean
  indexers: { total: number; enabled: number }
}

export interface AppConfig {
  tmdbApiKey: string
  tmdbApiKeySet: boolean
  searchTimeoutSec: number
  /** FlareSolverr address for trackers behind Cloudflare ("" = off). */
  flareSolverrUrl: string
}

export interface Release {
  title: string
  tracker: string
  trackerId: string
  size: number
  seeders: number
  peers: number
  link: string
  magnet: string
  infoHash: string
  details: string
  publishDate: string | null
  category: number[]
  categoryDesc: string
}

export interface IndexerStatus {
  id: string
  name: string
  status: 'ok' | 'error'
  results: number
  error: string
  elapsedMs: number
}

export interface SearchResponse {
  results: Release[]
  indexers: IndexerStatus[]
}

type IndexerType = 'public' | 'semi-private' | 'private'

export interface Indexer {
  id: string
  name: string
  description: string
  language: string
  type: IndexerType
  links: string[]
  enabled: boolean
  configured: boolean
  lastError: string
  lastCheckedAt: string | null
}

interface IndexerSettingOption {
  value: string
  label: string
}

export interface IndexerSetting {
  name: string
  /** text | password | checkbox | select | multi-select | info* */
  type: string
  label: string
  default?: unknown
  options?: IndexerSettingOption[] | null
}

export type ConfigValue = string | boolean | string[]

export interface IndexerDetail extends Indexer {
  settings: IndexerSetting[]
  config: Record<string, ConfigValue>
  categories: { id: number; name: string }[]
}

export interface IndexerTestResult {
  ok: boolean
  results: number
  error: string
  elapsedMs: number
}

export interface HistoryEntry {
  hash: string
  fileId: number
  fileName: string
  torrentTitle: string
  media: Media | null
  positionMs: number
  durationMs: number
  updatedAt: string
}

export type HistoryInput = Omit<HistoryEntry, 'updatedAt'>

export interface TorrentFile {
  id: number
  path: string
  length: number
}

export interface Torrent {
  hash: string
  title: string
  poster: string
  /** Opaque client JSON (the media card), may be "". */
  data: string
  /** Torrent name from metadata, "" until it arrives. */
  name: string
  /** 0 added, 1 getting info, 2 preload, 3 working, 4 closed, 5 in db (not loaded). */
  stat: number
  statString: string
  /** UTC ISO 8601. */
  addedAt: string
  torrentSize: number
  loadedSize: number
  preloadedBytes: number
  preloadSize: number
  /** The whole torrent is being downloaded and kept in the cache; progress is loadedSize / torrentSize. */
  fullDownload: boolean
  downloadSpeed: number
  uploadSpeed: number
  totalPeers: number
  pendingPeers: number
  activePeers: number
  connectedSeeders: number
  halfOpenPeers: number
  bytesRead: number
  /** [] until metadata arrives; ids start at 1. */
  files: TorrentFile[]
}

export interface AddTorrent {
  link: string
  title?: string
  poster?: string
  data?: string
}

interface CachePiece {
  id: number
  length: number
  size: number
  completed: boolean
  priority: number
}

export interface CacheState {
  hash: string
  capacity: number
  filled: number
  piecesLength: number
  piecesCount: number
  /** Pieces with data (cache or kept on disk) plus queued ones with nothing yet (size 0, priority > 0). */
  pieces: Record<string, CachePiece>
  readers: { start: number; end: number; reader: number }[]
}

export interface EngineSettings {
  cacheSize: number
  readerReadAhead: number
  preloadCache: number
  useDisk: boolean
  torrentsSavePath: string
  removeCacheOnDrop: boolean
  forceEncrypt: boolean
  retrackersMode: number
  torrentDisconnectTimeout: number
  enableDebug: boolean
  enableIPv6: boolean
  disableTCP: boolean
  disableUTP: boolean
  disableUPNP: boolean
  disableDHT: boolean
  disablePEX: boolean
  disableUpload: boolean
  downloadRateLimit: number
  uploadRateLimit: number
  connectionsLimit: number
  peersListenPort: number
  responsiveMode: boolean
}

export interface DefinitionsState {
  source: string
  count: number
  updatedAt: string | null
  updating: boolean
  lastError: string
}
