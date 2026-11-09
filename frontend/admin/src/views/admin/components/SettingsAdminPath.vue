<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { confirmAction } from '@/utils/confirm'
import { notifySuccess } from '@/utils/notify'
import { useServiceRestart } from '@/composables/useServiceRestart'

interface EntryState {
  current_path: string
  saved_path: string
  can_change: boolean
  can_restart: boolean
  restart_required: boolean
  block_reason: string
}

const { t } = useI18n()
const state = ref<EntryState | null>(null)
const path = ref('')
const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const savedURL = computed(() => new URL(`${state.value?.saved_path}/login`, window.location.origin).href)
let disposed = false
const { restarting, restart } = useServiceRestart(
  async () => {
    // Confirm the new process and its actual admin HTML, not a storefront fallback.
    const response = await fetch('/api/v1/admin/system/admin-path', {
      headers: { Authorization: `Bearer ${localStorage.getItem('admin_token') || ''}` },
      cache: 'no-store', signal: AbortSignal.timeout(5000),
    })
    const payload = await response.json()
    if (!response.ok || payload.data?.current_path !== state.value?.saved_path) return false
    const entry = await fetch(savedURL.value, { cache: 'no-store', signal: AbortSignal.timeout(5000) })
    return entry.ok && (await entry.text()).includes(`<base href="${state.value!.saved_path}/">`)
  },
  () => window.location.assign(savedURL.value),
  reason => { error.value = t(`admin.adminPath.${reason}`) },
)
const busy = computed(() => submitting.value || restarting.value)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await adminAPI.getAdminPath()
    state.value = result.data.data
    path.value = state.value!.saved_path || state.value!.current_path
  } catch {
    error.value = t('admin.adminPath.loadFailed')
  } finally {
    loading.value = false
  }
}

async function save(restartOnly = false) {
  if (!state.value || busy.value) return
  const target = restartOnly ? state.value.saved_path : path.value.trim()
  if (!/^\/(?:[A-Za-z0-9._~@-]+\/)*[A-Za-z0-9._~@-]+$/.test(target)
    || target.length > 200 || target.split('/').some(segment => segment === '.' || segment === '..')) {
    error.value = t('admin.adminPath.invalid')
    return
  }
  const ok = await confirmAction({
    title: t('admin.adminPath.title'),
    description: t(state.value.can_restart ? 'admin.adminPath.confirmAuto' : 'admin.adminPath.confirmManual', { url: new URL(`${target}/login`, window.location.origin).href }),
    confirmText: t(state.value.can_restart ? 'admin.adminPath.saveRestart' : 'admin.adminPath.save'),
    variant: 'destructive',
  })
  if (!ok || disposed) return
  submitting.value = true
  error.value = ''
  try {
    if (!restartOnly) {
      const result = await adminAPI.updateAdminPath({ path: target })
      Object.assign(state.value, result.data.data)
      path.value = target
    }
    if (state.value.restart_required && state.value.can_restart) {
      await restart()
    } else {
      notifySuccess(t(state.value.restart_required ? 'admin.adminPath.manualRestart' : 'admin.adminPath.saved'))
    }
  } catch {
    error.value = t('admin.adminPath.saveFailed')
  } finally {
    submitting.value = false
  }
}

onMounted(load)
onUnmounted(() => { disposed = true })
</script>

<template>
  <section class="rounded-xl border border-border bg-card">
    <div class="border-b border-border bg-muted/40 px-6 py-4">
      <h2 class="text-lg font-semibold">{{ t('admin.adminPath.title') }}</h2>
      <p class="mt-1 text-xs text-muted-foreground">{{ t('admin.adminPath.subtitle') }}</p>
    </div>
    <div class="space-y-4 p-6" :aria-busy="loading || busy">
      <p v-if="loading" class="text-sm text-muted-foreground">{{ t('admin.adminPath.loading') }}</p>
      <template v-else-if="state">
        <p class="text-sm">{{ t('admin.adminPath.current') }} <code>{{ state.current_path }}/</code></p>
        <p v-if="!state.can_change" class="text-sm text-amber-700 dark:text-amber-300">{{ t(`admin.adminPath.${state.block_reason}`) }}</p>
        <div class="space-y-2">
          <Label for="admin-entry-path">{{ t('admin.adminPath.label') }}</Label>
          <Input id="admin-entry-path" v-model="path" maxlength="200" placeholder="/shop-console" :disabled="!state.can_change || busy" autocomplete="off" spellcheck="false" />
          <p class="text-xs text-muted-foreground">{{ t('admin.adminPath.hint') }}</p>
        </div>
        <p v-if="state.restart_required" class="break-all rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm" role="status">
          {{ t('admin.adminPath.pending') }} <a :href="savedURL" class="font-medium underline">{{ savedURL }}</a>
        </p>
        <p v-if="busy" class="text-sm" role="status">{{ t('admin.adminPath.restarting') }}</p>
        <div class="flex flex-wrap gap-2">
          <Button :disabled="!state.can_change || busy || path === state.saved_path" @click="save()">{{ t(state.can_restart ? 'admin.adminPath.saveRestart' : 'admin.adminPath.save') }}</Button>
          <Button v-if="state.restart_required && state.can_restart && !busy" variant="outline" @click="save(true)">{{ t('admin.adminPath.restart') }}</Button>
        </div>
        <p v-if="!state.can_restart" class="text-xs text-muted-foreground">{{ t('admin.adminPath.manualRestart') }}</p>
      </template>
      <p v-if="error" class="text-sm text-destructive" role="alert">{{ error }}</p>
      <Button v-if="!state && !loading" variant="outline" @click="load">{{ t('admin.adminPath.retry') }}</Button>
    </div>
  </section>
</template>
