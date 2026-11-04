<script setup lang="ts">
import { msg } from '@/i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/ui/Icon.vue'
import type { TabId } from '@/router'
import { NAV, type NavItem } from './nav'

/** Bottom tab bar (phones) or left navigation rail (≥ 840px). */
const props = defineProps<{ active: TabId | null; rail?: boolean }>()
const router = useRouter()

function go(item: NavItem) {
  if (props.active === item.id && router.currentRoute.value.meta.tab === item.id) {
    // re-tap on the current tab scrolls it to the top, like native apps
    document.querySelector('.page')?.scrollTo({ top: 0, behavior: 'smooth' })
    return
  }
  void router.push(item.to)
}
</script>

<template>
  <nav class="nav" :class="rail ? 'nav--rail' : 'nav--bar'" :aria-label="msg.common.sections">
    <div v-if="rail" class="nav__logo" aria-hidden="true">
      <svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="5" width="18" height="14" rx="2" />
        <path d="M10 9v6l5-3z" fill="currentColor" stroke="none" />
      </svg>
    </div>
    <button
      v-for="item in NAV"
      :key="item.id"
      type="button"
      class="nav__item"
      :class="{ 'nav__item--active': active === item.id }"
      :aria-current="active === item.id ? 'page' : undefined"
      @click="go(item)"
    >
      <span class="nav__pill"><Icon :name="item.icon" :size="22" /></span>
      <span class="nav__label">{{ item.label }}</span>
    </button>
  </nav>
</template>

<style scoped>
.nav {
  z-index: var(--z-nav);
  background: var(--c-surface);
  display: flex;
}
.nav--bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  height: calc(var(--tabbar-h) + var(--safe-bottom));
  padding: 0 var(--safe-right) var(--safe-bottom) var(--safe-left);
  border-top: 1px solid var(--c-border);
  justify-content: space-around;
}
.nav--rail {
  position: relative;
  width: calc(var(--rail-w) + var(--safe-left));
  height: 100%;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: calc(16px + var(--safe-top)) 0 16px var(--safe-left);
  border-right: 1px solid var(--c-border);
  flex-shrink: 0;
}
.nav__logo {
  color: var(--c-accent);
  margin-bottom: 18px;
}
.nav__item {
  flex: 1;
  max-width: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  min-height: var(--touch);
  border: none;
  background: transparent;
  color: var(--c-muted);
  padding: 0;
  transition: color var(--dur);
}
.nav--rail .nav__item {
  flex: 0 0 auto;
  width: 100%;
  padding: 6px 0;
  gap: 4px;
}
.nav__item:hover {
  color: var(--c-text);
}
.nav__item--active,
.nav__item--active:hover {
  color: var(--c-accent);
}
.nav__pill {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 30px;
  border-radius: 15px;
  transition: background var(--dur);
}
.nav__item--active .nav__pill {
  background: rgba(var(--rgb-accent), var(--a-segment));
}
.nav__label {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.3px;
  white-space: nowrap;
}
.nav__item:focus-visible {
  outline: none;
}
.nav__item:focus-visible .nav__pill {
  outline: 2px solid var(--c-accent);
}
</style>
