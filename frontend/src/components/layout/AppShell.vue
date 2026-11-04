<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { transitionFor, type PageTransition, type TabId } from '@/router'
import { useMediaQuery } from '@/composables/useMediaQuery'
import NavBar from './NavBar.vue'

const route = useRoute()
const router = useRouter()
const wide = useMediaQuery('(min-width: 840px)')
const transition = ref<PageTransition>('')
const lastTab = ref<TabId>('home')

// afterEach: history.state.position is already the target entry's here,
// and the new view renders on the next flush with the right transition.
router.afterEach((to, from) => {
  transition.value = transitionFor(to, from)
  const tab = to.meta.tab ?? to.meta.parentTab
  if (tab) lastTab.value = tab
})

const fullscreen = computed(() => !!route.meta.fullscreen)
const activeTab = computed(() => route.meta.tab ?? route.meta.parentTab ?? lastTab.value)
const cached = ['HomeView', 'SearchView', 'TorrentsView', 'PrefsView']
</script>

<template>
  <div class="shell" :class="{ 'shell--wide': wide, 'shell--full': fullscreen }">
    <div class="grid-bg" aria-hidden="true" />
    <NavBar v-if="wide && !fullscreen" :active="activeTab" rail />
    <main class="shell__main">
      <RouterView v-slot="{ Component, route: r }">
        <Transition :name="`page-${transition}`" :css="!!transition">
          <KeepAlive :include="cached" :max="8">
            <component :is="Component" :key="r.meta.tab ? String(r.name) : r.fullPath" />
          </KeepAlive>
        </Transition>
      </RouterView>
    </main>
    <NavBar v-if="!wide && !fullscreen" :active="activeTab" />
  </div>
</template>

<style scoped>
.shell {
  position: relative;
  height: 100%;
  display: flex;
  background: var(--c-bg);
  overflow: hidden;
}
.shell__main {
  position: relative;
  z-index: var(--z-page);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  margin-bottom: calc(var(--tabbar-h) + var(--safe-bottom));
}
.shell--wide .shell__main,
.shell--full .shell__main {
  margin-bottom: 0;
}
</style>

<style>
/* Each screen is its own scroll container, so KeepAlive tabs keep their scroll. */
.page {
  position: absolute;
  inset: 0;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  padding-right: var(--safe-right);
}
.shell:not(.shell--wide) .page {
  padding-left: var(--safe-left);
}

.page-push-enter-active,
.page-push-leave-active,
.page-pop-enter-active,
.page-pop-leave-active {
  transition:
    transform var(--dur-page) cubic-bezier(0.2, 0.8, 0.2, 1),
    opacity var(--dur-page);
  background-color: var(--c-bg);
  will-change: transform;
}
.page-push-enter-active,
.page-pop-leave-active {
  z-index: 2;
}
.page-push-enter-from {
  transform: translateX(100%);
}
.page-push-leave-to {
  transform: translateX(-25%);
  opacity: 0.4;
}
.page-pop-enter-from {
  transform: translateX(-25%);
  opacity: 0.4;
}
.page-pop-leave-to {
  transform: translateX(100%);
}
</style>
