import { createRouter, createWebHistory, type RouteLocationNormalized } from 'vue-router'
import HomeView from '@/views/HomeView.vue'

export type TabId = 'home' | 'search' | 'torrents' | 'prefs'

declare module 'vue-router' {
  interface RouteMeta {
    tab?: TabId
    /** Tab a pushed screen belongs to, highlighted when it is opened directly by URL. */
    parentTab?: TabId
    fullscreen?: boolean
  }
}

// SPA paths never reuse server paths (/api, /img, /health): a reload on any SPA
// route must reach the SPA fallback, not a server handler.
const router = createRouter({
  history: createWebHistory('/'),
  routes: [
    { path: '/', name: 'home', component: HomeView, meta: { tab: 'home' } },
    { path: '/search', name: 'search', component: () => import('@/views/SearchView.vue'), meta: { tab: 'search' } },
    {
      path: '/downloads',
      name: 'torrents',
      component: () => import('@/views/TorrentsView.vue'),
      meta: { tab: 'torrents' },
    },
    {
      path: '/prefs/:section(catalog|trackers|engine)?',
      name: 'prefs',
      component: () => import('@/views/PrefsView.vue'),
      props: true,
      meta: { tab: 'prefs' },
    },
    {
      path: '/:type(movie|tv)/:id(\\d+)',
      name: 'details',
      component: () => import('@/views/DetailsView.vue'),
      props: (r) => ({ type: r.params.type, id: Number(r.params.id) }),
    },
    {
      path: '/releases/:type(movie|tv)/:id(\\d+)',
      name: 'releases',
      component: () => import('@/views/ReleasesView.vue'),
      props: (r) => ({ type: r.params.type, id: Number(r.params.id) }),
    },
    {
      path: '/trailers/:type(movie|tv)/:id(\\d+)',
      name: 'trailers',
      component: () => import('@/views/TrailersView.vue'),
      props: (r) => ({ type: r.params.type, id: Number(r.params.id) }),
    },
    {
      path: '/files/:hash',
      name: 'files',
      component: () => import('@/views/FilesView.vue'),
      props: (r) => ({
        hash: String(r.params.hash),
        resume: r.query.resume ? Number(r.query.resume) : undefined,
        browse: r.query.browse === '1',
      }),
    },
    {
      path: '/player/:hash/:fileId(\\d+)',
      name: 'player',
      component: () => import('@/views/PlayerView.vue'),
      props: (r) => ({ hash: String(r.params.hash), fileId: Number(r.params.fileId) }),
      meta: { fullscreen: true },
    },
    {
      path: '/prefs/trackers/:id',
      name: 'indexer',
      component: () => import('@/views/IndexerView.vue'),
      props: true,
      meta: { parentTab: 'prefs' },
    },
    // Trackers used to be a tab of their own; old links land in Settings → Trackers.
    { path: '/indexers', redirect: '/prefs/trackers' },
    { path: '/indexers/:id', redirect: (to) => `/prefs/trackers/${encodeURIComponent(String(to.params.id))}` },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

export type PageTransition = 'push' | 'pop' | ''

const statePosition = () =>
  typeof window.history.state?.position === 'number' ? (window.history.state.position as number) : 0
let lastPosition = statePosition()

/** Slide for push/pop; nothing for tab switches and replaces. */
export function transitionFor(to: RouteLocationNormalized, from: RouteLocationNormalized): PageTransition {
  const pos = statePosition()
  const prev = lastPosition
  lastPosition = pos
  if (!from.matched.length) return ''
  if (pos < prev) return 'pop'
  if (pos === prev) return ''
  return to.meta.tab ? '' : 'push'
}

export default router
