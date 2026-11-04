import type { IconName } from '@/components/ui/icons'
import { msg } from '@/i18n'
import type { TabId } from '@/router'

export interface NavItem {
  id: TabId
  to: string
  label: string
  icon: IconName
}

export const NAV: NavItem[] = [
  { id: 'home', to: '/', label: msg.tabs.home, icon: 'home' },
  { id: 'search', to: '/search', label: msg.tabs.search, icon: 'search' },
  { id: 'torrents', to: '/downloads', label: msg.tabs.torrents, icon: 'download' },
  { id: 'prefs', to: '/prefs', label: msg.tabs.prefs, icon: 'settings' },
]
