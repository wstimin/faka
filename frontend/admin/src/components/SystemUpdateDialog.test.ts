import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import SystemUpdateDialog from './SystemUpdateDialog.vue'

const mocks = vi.hoisted(() => ({
  checkSystemUpdate: vi.fn(), getUpdateCapability: vi.fn(), getUpdateStatus: vi.fn(),
  startSystemUpdate: vi.fn(), restartSystemService: vi.fn(), getPublicConfig: vi.fn(),
  rollbackSystemUpdate: vi.fn(), confirmAction: vi.fn(), notifyError: vi.fn(), notifySuccess: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: mocks }))
vi.mock('@/utils/confirm', () => ({ confirmAction: mocks.confirmAction }))
vi.mock('@/utils/notify', () => ({ notifyError: mocks.notifyError, notifySuccess: mocks.notifySuccess }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/ui/dialog', async () => {
  const { defineComponent, h } = await import('vue')
  const slot = defineComponent({ setup(_, { slots }) { return () => h('div', slots.default?.()) } })
  return { Dialog: slot, DialogContent: slot, DialogHeader: slot, DialogTitle: slot, DialogFooter: slot }
})
vi.mock('@/components/ui/button', async () => {
  const { defineComponent, h } = await import('vue')
  return { Button: defineComponent({ setup(_, { slots }) { return () => h('button', slots.default?.()) } }) }
})

const response = (data: unknown) => ({ data: { data } })
const idle = { status: 'idle', stage: 'idle', percent: 0, restart_required: false }
const running = { status: 'running', stage: 'downloading', percent: 37, target_version: 'v1.5.3-faka.2', restart_required: false }
const succeeded = { ...running, status: 'succeeded', stage: 'done', percent: 100, restart_required: true }
async function flush() { for (let i = 0; i < 12; i++) await Promise.resolve(); await nextTick() }

describe('modified build update dialog', () => {
  let unmount: () => void
  let container: HTMLDivElement
  const reload = vi.fn()
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    const fakeWindow = Object.create(window)
    Object.defineProperty(fakeWindow, 'location', { value: { reload } })
    vi.stubGlobal('window', fakeWindow)
    mocks.confirmAction.mockResolvedValue(true)
    mocks.checkSystemUpdate.mockResolvedValue(response({ current_version: 'v1.5.3-faka.1', latest_version: running.target_version, has_update: true, source: 'https://github.com/wstimin/faka/releases' }))
    mocks.getUpdateCapability.mockResolvedValue(response({ capability: { can_update: true, can_restart: true, has_backup: false }, state: idle }))
    mocks.startSystemUpdate.mockResolvedValue(response(running))
    mocks.restartSystemService.mockResolvedValue(response({ restarting: true }))
    mocks.getUpdateStatus.mockResolvedValue(response(running))
    mocks.getPublicConfig.mockResolvedValue(response({ app_version: 'v1.5.3-faka.1' }))
  })
  afterEach(() => { unmount?.(); container?.remove(); vi.unstubAllGlobals(); vi.useRealTimers() })

  async function startUpdate() {
    container = document.createElement('div')
    document.body.append(container)
    const open = ref(false)
    const app = createApp(defineComponent({ setup: () => () => h(SystemUpdateDialog, { open: open.value }) }))
    app.mount(container)
    unmount = () => app.unmount()
    open.value = true
    await nextTick(); await flush()
    const button = [...container.querySelectorAll('button')].find(item => item.textContent?.includes('admin.systemUpdate.upgradeNow'))
    expect(button).toBeDefined()
    button!.click()
    await flush()
  }

  it('shows download progress, restarts once and refreshes only after the target version is running', async () => {
    await startUpdate()
    await vi.advanceTimersByTimeAsync(1500); await flush()
    expect(container.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('37')
    mocks.getUpdateStatus.mockResolvedValue(response(succeeded))
    await vi.advanceTimersByTimeAsync(1500); await flush()
    expect(mocks.restartSystemService).toHaveBeenCalledTimes(1)
    expect(mocks.confirmAction).toHaveBeenCalledTimes(1)
    expect(mocks.confirmAction.mock.calls[0]?.[0].description).toBe('admin.systemUpdate.autoConfirmDescription')
    await vi.advanceTimersByTimeAsync(2000); await flush()
    expect(reload).not.toHaveBeenCalled()
    mocks.getPublicConfig.mockResolvedValue(response({ app_version: running.target_version }))
    await vi.advanceTimersByTimeAsync(2000); await flush()
    expect(reload).not.toHaveBeenCalled() // The old process still reports restart_required.
    mocks.getUpdateStatus.mockResolvedValue(response(idle))
    await vi.advanceTimersByTimeAsync(2000); await flush()
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('does not restart after a failed update', async () => {
    await startUpdate()
    mocks.getUpdateStatus.mockResolvedValue(response({ ...running, status: 'failed', error: 'checksum mismatch' }))
    await vi.advanceTimersByTimeAsync(1500); await flush()
    expect(mocks.restartSystemService).not.toHaveBeenCalled()
    expect(mocks.notifyError).toHaveBeenCalledWith('checksum mismatch')
    expect(reload).not.toHaveBeenCalled()
  })

  it('does not stop an unmanaged service', async () => {
    mocks.getUpdateCapability.mockResolvedValue(response({ capability: { can_update: true, can_restart: false }, state: idle }))
    await startUpdate()
    mocks.getUpdateStatus.mockResolvedValue(response(succeeded))
    await vi.advanceTimersByTimeAsync(1500); await flush()
    expect(mocks.restartSystemService).not.toHaveBeenCalled()
  })
})
