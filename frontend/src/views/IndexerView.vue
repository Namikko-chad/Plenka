<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { ConfigValue, IndexerDetail, IndexerSetting, IndexerTestResult } from '@/api/types'
import AppBar from '@/components/layout/AppBar.vue'
import Badge from '@/components/ui/Badge.vue'
import Chip from '@/components/ui/Chip.vue'
import ErrorBanner from '@/components/ui/ErrorBanner.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import StatusDot from '@/components/ui/StatusDot.vue'
import Toggle from '@/components/ui/Toggle.vue'
import UiButton from '@/components/ui/UiButton.vue'
import { useAsync } from '@/composables/useAsync'
import { msg } from '@/i18n'
import { useIndexers } from '@/stores/indexers'
import { useToast } from '@/stores/toast'
import { errorText } from '@/utils/errors'
import { formatDateTime } from '@/utils/format'
import { sanitizeHtml } from '@/utils/sanitize'
import { formValueOf, isInfo, toWireValue, type FormValue } from '@/utils/indexerForm'

const props = defineProps<{ id: string }>()
const indexers = useIndexers()
const toast = useToast()

const detail = useAsync<IndexerDetail>(() => indexers.get(props.id))
const d = computed(() => detail.data.value)

const enabled = ref(false)
const values = ref<Record<string, FormValue>>({})
const sitelink = ref('')
const customLink = ref('')
const saving = ref(false)
const testing = ref(false)
const testResult = ref<IndexerTestResult | null>(null)

const CUSTOM = '__custom__'
const fields = computed(() => (d.value?.settings ?? []).filter((s) => s.name !== 'sitelink'))
const editable = computed(() => fields.value.filter((s) => !isInfo(s.type)))

function reset(det: IndexerDetail) {
  enabled.value = det.enabled
  const v: Record<string, FormValue> = {}
  for (const s of det.settings ?? []) {
    if (!isInfo(s.type) && s.name !== 'sitelink') v[s.name] = formValueOf(s, det.config?.[s.name])
  }
  values.value = v
  const link = String(det.config?.sitelink ?? det.links[0] ?? '')
  if (!link || det.links.includes(link)) {
    sitelink.value = link
    customLink.value = ''
  } else {
    sitelink.value = CUSTOM
    customLink.value = link
  }
}

async function load() {
  const r = await detail.run()
  if (r) reset(r)
}

function passwordSet(s: IndexerSetting) {
  return d.value?.config?.[`${s.name}__set`] === '1'
}

function toggleMulti(name: string, value: string) {
  const cur = values.value[name]
  const arr = Array.isArray(cur) ? cur : []
  values.value[name] = arr.includes(value) ? arr.filter((x) => x !== value) : [...arr, value]
}

// persist saves the form; it returns false (after a toast) when saving failed.
async function persist(): Promise<boolean> {
  const det = d.value
  if (!det) return false
  try {
    const config: Record<string, ConfigValue> = {}
    for (const s of editable.value) {
      config[s.name] = toWireValue(s, values.value[s.name] ?? '', det.config?.[s.name])
    }
    const link = sitelink.value === CUSTOM ? customLink.value.trim() : sitelink.value
    if (link) config.sitelink = link
    const r = await indexers.save(det.id, { enabled: enabled.value, config })
    detail.data.value = r
    reset(r)
    return true
  } catch (e) {
    toast.error(errorText(e))
    return false
  }
}

async function save() {
  if (saving.value) return
  saving.value = true
  try {
    if (await persist()) toast.ok(msg.indexers.saved)
  } finally {
    saving.value = false
  }
}

// test checks what is in the form: it saves first, then runs the check and
// refreshes the tracker's last error.
async function test() {
  if (testing.value || saving.value) return
  testing.value = true
  testResult.value = null
  try {
    if (!(await persist())) return
    testResult.value = await indexers.test(props.id)
  } catch (e) {
    testResult.value = { ok: false, results: 0, error: errorText(e), elapsedMs: 0 }
  } finally {
    testing.value = false
  }
  try {
    detail.data.value = await indexers.get(props.id)
  } catch {
    // the result banner already shows the outcome
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <AppBar :title="d?.name ?? id" back="/prefs/trackers" />
    <div class="page-body">
      <ErrorBanner v-if="detail.error.value" :error="detail.error.value" retry @retry="load" />

      <div v-else-if="!d" class="stack">
        <Skeleton height="90px" />
        <Skeleton height="220px" />
      </div>

      <template v-else>
        <section class="card card-pad stack">
          <div class="row-wrap">
            <Badge :tone="d.type === 'public' ? 'accent' : 'purple'">
              {{ d.type === 'public' ? msg.indexers.typePublic : d.type === 'private' ? msg.indexers.typePrivate : msg.indexers.typeSemi }}
            </Badge>
            <Badge v-if="d.language" tone="neutral">{{ d.language }}</Badge>
            <Badge :tone="d.configured ? 'ok' : 'warning'">
              {{ d.configured ? msg.indexers.configured : msg.indexers.notConfigured }}
            </Badge>
          </div>
          <p v-if="d.description" class="ixd__desc">{{ d.description }}</p>
          <p v-if="d.lastError" class="ixd__err">{{ msg.indexers.lastError }}: {{ d.lastError }}</p>
          <p v-if="d.lastCheckedAt" class="small" style="margin: 0">{{ msg.indexers.lastChecked }}: {{ formatDateTime(d.lastCheckedAt) }}</p>
        </section>

        <form class="card card-pad stack" @submit.prevent="save">
          <h2 class="section-title">{{ msg.indexers.settings }}</h2>
          <Toggle v-model="enabled" :label="msg.indexers.enabledLabel" />

          <div v-if="d.links.length" class="field">
            <label class="label" for="ix-link">{{ msg.indexers.links }}</label>
            <select id="ix-link" v-model="sitelink" class="select">
              <option v-for="l in d.links" :key="l" :value="l">{{ l }}</option>
              <option :value="CUSTOM">{{ msg.indexers.customLink }}</option>
            </select>
            <input
              v-if="sitelink === CUSTOM"
              v-model="customLink"
              class="input"
              style="margin-top: 8px"
              type="url"
              placeholder="https://"
              :aria-label="msg.indexers.customLink"
            />
          </div>

          <template v-for="s in fields" :key="s.name">
            <!-- info HTML is sanitized to a tiny allowlist (utils/sanitize.ts) -->
            <div v-if="isInfo(s.type)" class="ixd__info">
              <span v-if="s.label" class="label">{{ s.label }}</span>
              <div class="ixd__info-html" v-html="sanitizeHtml(String(s.default ?? ''))" />
            </div>

            <Toggle
              v-else-if="s.type === 'checkbox'"
              :model-value="values[s.name] === true"
              :label="s.label || s.name"
              @update:model-value="(v: boolean) => (values[s.name] = v)"
            />

            <div v-else-if="s.type === 'select'" class="field">
              <label class="label" :for="`ix-${s.name}`">{{ s.label || s.name }}</label>
              <select :id="`ix-${s.name}`" v-model="values[s.name]" class="select">
                <option v-for="o in s.options ?? []" :key="o.value" :value="o.value">{{ o.label }}</option>
              </select>
            </div>

            <div v-else-if="s.type === 'multi-select'" class="field">
              <span class="label">{{ s.label || s.name }}</span>
              <div class="row-wrap">
                <Chip
                  v-for="o in s.options ?? []"
                  :key="o.value"
                  :active="Array.isArray(values[s.name]) && (values[s.name] as string[]).includes(o.value)"
                  @click="toggleMulti(s.name, o.value)"
                >
                  {{ o.label }}
                </Chip>
              </div>
            </div>

            <div v-else class="field">
              <label class="label" :for="`ix-${s.name}`">{{ s.label || s.name }}</label>
              <input
                :id="`ix-${s.name}`"
                :value="String(values[s.name] ?? '')"
                class="input"
                :type="s.type === 'password' ? 'password' : 'text'"
                :autocomplete="s.type === 'password' ? 'new-password' : 'off'"
                :placeholder="s.type === 'password' && passwordSet(s) ? '••••••••' : ''"
                @input="values[s.name] = ($event.target as HTMLInputElement).value"
              />
              <span v-if="s.type === 'password' && passwordSet(s)" class="hint">{{ msg.indexers.passwordSet }}</span>
            </div>
          </template>

          <p v-if="!fields.length" class="small" style="margin: 0">{{ msg.indexers.noSettings }}</p>

          <div class="row-wrap ixd__actions">
            <UiButton icon="zap" :loading="testing" @click="test">{{ msg.indexers.test }}</UiButton>
            <span class="spacer" />
            <UiButton variant="primary" type="submit" icon="check" :loading="saving">{{ msg.common.save }}</UiButton>
          </div>

          <div v-if="testResult" class="ixd__test" :class="testResult.ok ? 'ixd__test--ok' : 'ixd__test--err'" role="status">
            <StatusDot :ok="testResult.ok" />
            <span>{{
              testResult.ok ? msg.indexers.testOk(testResult.results, testResult.elapsedMs) : msg.indexers.testFail(testResult.error)
            }}</span>
          </div>
        </form>

        <section v-if="d.categories?.length" class="stack">
          <h2 class="section-title">{{ msg.indexers.categories }}</h2>
          <div class="row-wrap">
            <Badge v-for="c in d.categories.slice(0, 40)" :key="c.id" tone="neutral">{{ c.id }} {{ c.name }}</Badge>
          </div>
        </section>
      </template>
    </div>
  </div>
</template>

<style scoped>
.ixd__desc {
  margin: 0;
  line-height: 1.5;
}
.ixd__err {
  margin: 0;
  color: var(--c-error);
  font-size: 12px;
  overflow-wrap: anywhere;
}
.ixd__info {
  padding: 10px 12px;
  border-radius: var(--r-control);
  background: rgba(var(--rgb-accent), 0.04);
  border: 1px solid rgba(var(--rgb-accent), 0.15);
}
.ixd__info-html {
  font-size: 12px;
  line-height: 1.5;
  color: var(--c-muted);
  overflow-wrap: anywhere;
}
.ixd__info-html :deep(p) {
  margin: 0 0 6px;
}
.ixd__info-html :deep(ul),
.ixd__info-html :deep(ol) {
  margin: 4px 0;
  padding-left: 18px;
}
.ixd__actions {
  margin-top: 6px;
}
.ixd__test {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--r-control);
  font-size: 12px;
}
.ixd__test--ok {
  background: rgba(var(--rgb-ok), 0.07);
  color: var(--c-ok);
}
.ixd__test--err {
  background: rgba(var(--rgb-error), 0.07);
  color: var(--c-error);
  overflow-wrap: anywhere;
}
</style>
