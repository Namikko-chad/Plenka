<script setup lang="ts">
import { useRouter, type RouteLocationRaw } from 'vue-router'
import IconButton from '@/components/ui/IconButton.vue'
import { msg } from '@/i18n'

/** Per-screen top bar; `back` shows the back button (pushed screens). */
const props = defineProps<{ title?: string; subtitle?: string; back?: RouteLocationRaw | true; large?: boolean }>()
const router = useRouter()

function goBack() {
  if (window.history.state?.back) router.back()
  else void router.replace(props.back === true || !props.back ? '/' : props.back)
}
</script>

<template>
  <header class="appbar" :class="{ 'appbar--large': large && !back }">
    <IconButton v-if="back" icon="back" :label="msg.common.back" class="appbar__back" @click="goBack" />
    <div class="appbar__titles">
      <h1 v-if="title" class="appbar__title" :class="{ 'page-title': large && !back }">{{ title }}</h1>
      <p v-if="subtitle" class="appbar__subtitle ellipsis">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.default" class="appbar__actions"><slot /></div>
  </header>
</template>

<style scoped>
.appbar {
  position: sticky;
  top: 0;
  z-index: var(--z-appbar);
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: calc(var(--appbar-h) + var(--safe-top));
  padding: var(--safe-top) calc(var(--gutter) - 8px) 0 calc(var(--gutter) - 8px);
  background: rgba(var(--rgb-bg), 0.86);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--c-border);
}
.appbar--large {
  padding-left: var(--gutter);
}
.appbar__titles {
  flex: 1;
  min-width: 0;
  padding: 0 6px;
}
.appbar--large .appbar__titles {
  padding: 0;
}
.appbar__title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.appbar__subtitle {
  margin: 0;
  font-size: 11px;
  color: var(--c-muted);
}
.appbar__actions {
  display: flex;
  align-items: center;
  gap: 2px;
}
</style>
