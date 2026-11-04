<script setup lang="ts">
import { computed, onMounted } from 'vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Segmented from '@/components/ui/Segmented.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import StatusDot from '@/components/ui/StatusDot.vue'
import { languagePref, msg, setLanguage, type LanguagePref } from '@/i18n'
import { usePrefs, type PlayerMode } from '@/stores/prefs'
import { useServer } from '@/stores/server'

/** Settings → General: server status, interface language, player mode, app version. */
defineOptions({ name: 'PrefsGeneral' })

const server = useServer()
const prefs = usePrefs()
const appVersion = __APP_VERSION__
const info = computed(() => server.info)

// Language names are written in the language itself, so anyone can find their own.
const languages: { value: LanguagePref; label: string }[] = [
  { value: 'system', label: msg.prefs.languageSystem },
  { value: 'en', label: 'English' },
  { value: 'ru', label: 'Русский' },
]
function onLanguage(e: Event) {
  const v = (e.target as HTMLSelectElement).value as LanguagePref
  if (v !== languagePref) setLanguage(v)
}

const playerModes: { value: PlayerMode; label: string }[] = [
  { value: 'internal', label: msg.prefs.playerInternal },
  { value: 'external', label: msg.prefs.playerExternal },
  { value: 'ask', label: msg.prefs.playerAsk },
]

onMounted(() => void server.loadInfo(true))
</script>

<template>
  <div class="panel">
    <section class="card card-pad stack">
      <h2 class="section-title">{{ msg.prefs.status }}</h2>
      <ErrorBanner v-if="server.infoError && !info" :error="server.infoError" retry @retry="server.loadInfo(true)" />
      <ul v-else-if="info" class="status">
        <li><StatusDot :ok="info.torrents" /><span class="spacer">{{ msg.prefs.statusTorrents }}</span><span class="muted">{{ info.torrents ? msg.common.on : msg.common.off }}</span></li>
        <li>
          <StatusDot :ok="info.indexers.enabled > 0" /><span class="spacer">{{ msg.prefs.statusTrackers }}</span>
          <span class="muted">{{ info.indexers.enabled }} / {{ info.indexers.total }}</span>
        </li>
        <li><StatusDot :ok="info.tmdb" /><span class="spacer">{{ msg.prefs.statusTmdb }}</span><span class="muted">{{ info.tmdb ? msg.common.set : msg.common.notSet }}</span></li>
        <li><span class="status__pad" /><span class="spacer">{{ msg.prefs.version }}</span><span class="muted">{{ info.version }}</span></li>
      </ul>
      <Skeleton v-else height="120px" />
    </section>

    <section class="card card-pad stack">
      <h2 class="section-title">{{ msg.prefs.language }}</h2>
      <select class="select" :value="languagePref" :aria-label="msg.prefs.language" @change="onLanguage">
        <option v-for="l in languages" :key="l.value" :value="l.value">{{ l.label }}</option>
      </select>
    </section>

    <section class="card card-pad stack">
      <h2 class="section-title">{{ msg.prefs.player }}</h2>
      <span class="label" style="margin: 0">{{ msg.prefs.playerMode }}</span>
      <Segmented v-model="prefs.playerMode" :options="playerModes" :label="msg.prefs.playerMode" block />
      <p class="small" style="margin: 0">{{ msg.prefs.playerHint }}</p>
    </section>

    <p class="small version">{{ msg.app.name }} · {{ msg.prefs.appVersion }} {{ appVersion }}</p>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.status {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.status li {
  display: flex;
  align-items: center;
  gap: 12px;
}
.status__pad {
  width: 8px;
}
.version {
  text-align: center;
  margin: 0;
}
</style>
