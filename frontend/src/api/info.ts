import { request } from './http'
import type { AppConfig, ServerInfo } from './types'

export const infoApi = {
  info: () => request<ServerInfo>('/api/info', { timeout: 8000 }),
  config: () => request<AppConfig>('/api/config'),
  saveConfig: (patch: Partial<AppConfig>) => request<AppConfig>('/api/config', { method: 'PUT', body: patch }),
}
