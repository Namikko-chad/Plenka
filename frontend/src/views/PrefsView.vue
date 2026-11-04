<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useRouter } from 'vue-router'
import AppBar from '@/components/layout/AppBar.vue'
import PrefsCatalog from '@/components/prefs/PrefsCatalog.vue'
import PrefsEngine from '@/components/prefs/PrefsEngine.vue'
import PrefsGeneral from '@/components/prefs/PrefsGeneral.vue'
import PrefsTrackers from '@/components/prefs/PrefsTrackers.vue'
import IconButton from '@/components/ui/IconButton.vue'
import Segmented from '@/components/ui/Segmented.vue'
import { msg } from '@/i18n'
import { useServer } from '@/stores/server'

defineOptions({ name: 'PrefsView' })

type Section = 'general' | 'catalog' | 'trackers' | 'engine'
/** `/prefs/:section?` — no section is General. */
const props = defineProps<{ section?: string }>()
const router = useRouter()
const server = useServer()

const PANELS: Record<Section, Component> = {
  general: PrefsGeneral,
  catalog: PrefsCatalog,
  trackers: PrefsTrackers,
  engine: PrefsEngine,
}
const tabs: { value: Section; label: string }[] = [
  { value: 'general', label: msg.prefs.tabGeneral },
  { value: 'catalog', label: msg.prefs.tabCatalog },
  { value: 'trackers', label: msg.prefs.tabTrackers },
  { value: 'engine', label: msg.prefs.tabEngine },
]

// The tab lives in the URL (replace, so tab switches do not pile up in history).
const active = computed<Section>({
  get: () => (props.section && props.section in PANELS ? (props.section as Section) : 'general'),
  set: (s) => void router.replace(s === 'general' ? '/prefs' : `/prefs/${s}`),
})
</script>

<template>
  <div class="page">
    <AppBar :title="msg.prefs.title" large>
      <IconButton v-if="active === 'general'" icon="refresh" :label="msg.common.retry" @click="server.loadInfo(true)" />
    </AppBar>

    <div class="page-body prefs">
      <Segmented v-model="active" :options="tabs" :label="msg.prefs.title" block scroll />
      <KeepAlive>
        <component :is="PANELS[active]" :key="active" />
      </KeepAlive>
    </div>
  </div>
</template>

<style scoped>
.prefs {
  max-width: 760px;
}
</style>
