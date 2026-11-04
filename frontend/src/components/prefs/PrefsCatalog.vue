<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { msg } from '@/i18n'
import { useDialog } from '@/stores/dialog'
import { useServer } from '@/stores/server'
import { useToast } from '@/stores/toast'
import { errorText } from '@/utils/errors'

/** Settings → Catalog and search: the TMDB key (part of /api/config). The catalog language
 * follows the interface language. */
defineOptions({ name: 'PrefsCatalog' })

const server = useServer()
const dialog = useDialog()
const toast = useToast()

const configError = ref<unknown>(null)
const tmdbKey = ref('')
const saving = ref(false)

async function loadConfig() {
  try {
    await server.loadConfig(true)
    configError.value = null
  } catch (e) {
    configError.value = e
  }
}

async function save() {
  saving.value = true
  try {
    await server.saveConfig({ tmdbApiKey: tmdbKey.value.trim() })
    tmdbKey.value = ''
    toast.ok(msg.common.saved)
    void server.loadInfo(true)
  } catch (e) {
    toast.error(errorText(e))
  } finally {
    saving.value = false
  }
}

async function clearKey() {
  const ok = await dialog.confirm({ title: msg.prefs.tmdbKeyClear, confirmLabel: msg.common.delete, danger: true })
  if (!ok) return
  try {
    await server.saveConfig({ tmdbApiKey: '-' })
    toast.ok(msg.prefs.tmdbKeyCleared)
    void server.loadInfo(true)
  } catch (e) {
    toast.error(errorText(e))
  }
}

onMounted(() => void loadConfig())
</script>

<template>
  <form class="card card-pad" @submit.prevent="save">
    <h2 class="section-title title">{{ msg.prefs.app }}</h2>
    <ErrorBanner v-if="configError" :error="configError" retry @retry="loadConfig" />
    <template v-else-if="server.config">
      <div class="field">
        <label class="label" for="tmdb-key">{{ msg.prefs.tmdbKey }}</label>
        <div class="row">
          <input
            id="tmdb-key"
            v-model="tmdbKey"
            class="input"
            type="password"
            autocomplete="off"
            :placeholder="server.config.tmdbApiKeySet ? `${msg.common.set} ••••••••` : msg.prefs.tmdbKeyPh"
          />
          <UiButton v-if="server.config.tmdbApiKeySet" variant="red" icon="trash" @click="clearKey">{{ msg.common.delete }}</UiButton>
        </div>
        <span v-if="server.config.tmdbApiKeySet" class="hint">{{ msg.prefs.tmdbKeySet }}</span>
      </div>
      <div class="actions">
        <UiButton variant="primary" type="submit" icon="check" :loading="saving">{{ msg.common.save }}</UiButton>
      </div>
    </template>
    <Skeleton v-else height="140px" />
  </form>
</template>

<style scoped>
.title {
  margin-bottom: 14px;
}
.actions {
  display: flex;
  margin-top: 18px;
}
</style>
