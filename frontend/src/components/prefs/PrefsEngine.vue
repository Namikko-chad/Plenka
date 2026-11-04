<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import type { EngineSettings } from '@/api/types'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Toggle from '@/components/ui/Toggle.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { msg } from '@/i18n'
import { useDialog } from '@/stores/dialog'
import { useEngine } from '@/stores/engine'
import { useToast } from '@/stores/toast'
import { errorText } from '@/utils/errors'

/** Settings → Torrent engine: /api/engine/settings, reset to defaults. */
defineOptions({ name: 'PrefsEngine' })

const engine = useEngine()
const dialog = useDialog()
const toast = useToast()

const sets = ref<EngineSettings | null>(null)
const saving = ref(false)
watch(
  () => engine.sets,
  (s) => (sets.value = s ? { ...s } : null),
  { immediate: true },
)
const MB = 1024 * 1024
const cacheMb = computed({
  get: () => (sets.value ? Math.round(sets.value.cacheSize / MB) : 0),
  set: (v: number) => {
    if (sets.value) sets.value.cacheSize = Math.max(0, Math.round(v)) * MB
  },
})
/** Toggles phrased positively over the engine's disable* flags. */
const enabledFlag = (key: 'disableDHT' | 'disablePEX' | 'disableUPNP' | 'disableTCP' | 'disableUTP' | 'disableUpload') =>
  computed({
    get: () => !(sets.value?.[key] ?? false),
    set: (v: boolean) => {
      if (sets.value) sets.value[key] = !v
    },
  })
const dht = enabledFlag('disableDHT')
const pex = enabledFlag('disablePEX')
const upnp = enabledFlag('disableUPNP')
const tcp = enabledFlag('disableTCP')
const utp = enabledFlag('disableUTP')
const upload = enabledFlag('disableUpload')

const retrackers = [
  { value: 0, label: msg.prefs.retrackersNone },
  { value: 1, label: msg.prefs.retrackersAdd },
  { value: 2, label: msg.prefs.retrackersRemove },
  { value: 3, label: msg.prefs.retrackersReplace },
]

async function save() {
  if (!sets.value) return
  saving.value = true
  try {
    const s = sets.value
    s.readerReadAhead = Math.min(100, Math.max(5, Math.round(s.readerReadAhead)))
    s.preloadCache = Math.min(100, Math.max(0, Math.round(s.preloadCache)))
    await engine.save(s)
    toast.ok(msg.common.saved)
  } catch (e) {
    toast.error(errorText(e))
  } finally {
    saving.value = false
  }
}

async function reset() {
  const ok = await dialog.confirm({ title: msg.prefs.reset, text: msg.prefs.resetConfirm, confirmLabel: msg.prefs.reset, danger: true })
  if (!ok) return
  try {
    await engine.resetDefaults()
    toast.ok(msg.prefs.resetDone)
  } catch (e) {
    toast.error(errorText(e))
  }
}

onMounted(() => void engine.load())
</script>

<template>
  <form class="card card-pad" @submit.prevent="save">
    <h2 class="section-title title">{{ msg.prefs.engine }}</h2>
    <ErrorBanner v-if="engine.error" :error="engine.error" retry @retry="engine.load()" />
    <template v-else-if="sets">
      <div class="grid2">
        <div class="field">
          <label class="label" for="cache-size">{{ msg.prefs.cacheSize }}</label>
          <input id="cache-size" v-model.number="cacheMb" class="input" type="number" min="0" step="16" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="readahead">{{ msg.prefs.readahead }}</label>
          <input id="readahead" v-model.number="sets.readerReadAhead" class="input" type="number" min="5" max="100" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="preload">{{ msg.prefs.preload }}</label>
          <input id="preload" v-model.number="sets.preloadCache" class="input" type="number" min="0" max="100" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="conn">{{ msg.prefs.connections }}</label>
          <input id="conn" v-model.number="sets.connectionsLimit" class="input" type="number" min="0" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="down">{{ msg.prefs.downLimit }}</label>
          <input id="down" v-model.number="sets.downloadRateLimit" class="input" type="number" min="0" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="up">{{ msg.prefs.upLimit }}</label>
          <input id="up" v-model.number="sets.uploadRateLimit" class="input" type="number" min="0" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="disc">{{ msg.prefs.disconnect }}</label>
          <input id="disc" v-model.number="sets.torrentDisconnectTimeout" class="input" type="number" min="0" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="port">{{ msg.prefs.port }}</label>
          <input id="port" v-model.number="sets.peersListenPort" class="input" type="number" min="0" max="65535" inputmode="numeric" />
        </div>
        <div class="field">
          <label class="label" for="retr">{{ msg.prefs.retrackers }}</label>
          <select id="retr" v-model.number="sets.retrackersMode" class="select">
            <option v-for="r in retrackers" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
        </div>
      </div>

      <div class="toggles">
        <Toggle v-model="sets.useDisk" :label="msg.prefs.useDisk" />
        <div v-if="sets.useDisk" class="field">
          <label class="label" for="save-path">{{ msg.prefs.savePath }}</label>
          <input id="save-path" v-model="sets.torrentsSavePath" class="input" autocomplete="off" />
        </div>
        <Toggle v-if="sets.useDisk" v-model="sets.removeCacheOnDrop" :label="msg.prefs.removeCacheOnDrop" />
        <Toggle v-model="dht" :label="msg.prefs.dht" />
        <Toggle v-model="pex" :label="msg.prefs.pex" />
        <Toggle v-model="upnp" :label="msg.prefs.upnp" />
        <Toggle v-model="tcp" :label="msg.prefs.tcp" />
        <Toggle v-model="utp" :label="msg.prefs.utp" />
        <Toggle v-model="sets.enableIPv6" :label="msg.prefs.ipv6" />
        <Toggle v-model="upload" :label="msg.prefs.upload" />
        <Toggle v-model="sets.forceEncrypt" :label="msg.prefs.encrypt" />
        <Toggle v-model="sets.responsiveMode" :label="msg.prefs.responsive" />
        <Toggle v-model="sets.enableDebug" :label="msg.prefs.debug" />
      </div>

      <div class="actions">
        <UiButton variant="red" icon="refresh" @click="reset">{{ msg.prefs.reset }}</UiButton>
        <span class="spacer" />
        <UiButton variant="primary" type="submit" icon="check" :loading="saving">{{ msg.common.save }}</UiButton>
      </div>
    </template>
    <Skeleton v-else height="320px" />
  </form>
</template>

<style scoped>
.title {
  margin-bottom: 14px;
}
.grid2 {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 14px 16px;
}
.grid2 .field + .field {
  margin-top: 0;
}
.toggles {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
}
.toggles .field {
  margin: 4px 0 8px;
}
.actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 18px;
}
</style>
