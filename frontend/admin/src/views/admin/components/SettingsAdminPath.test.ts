import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import SettingsAdminPath from './SettingsAdminPath.vue'

const mocks = vi.hoisted(() => ({
  getAdminPath: vi.fn(), updateAdminPath: vi.fn(), restartSystemService: vi.fn(),
  confirmAction: vi.fn(), notifySuccess: vi.fn(), assign: vi.fn(), fetch: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: mocks }))
vi.mock('@/utils/confirm', () => ({ confirmAction: mocks.confirmAction }))
vi.mock('@/utils/notify', () => ({ notifySuccess: mocks.notifySuccess }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: { url: string }) => key + (params?.url ? ` ${params.url}` : '') }) }))
vi.mock('@/components/ui/button', async () => {
  const { defineComponent, h } = await import('vue')
  return { Button: defineComponent({ setup(_, { slots }) { return () => h('button', slots.default?.()) } }) }
})
vi.mock('@/components/ui/input', async () => {
  const { defineComponent, h } = await import('vue')
  return { Input: defineComponent({ props: ['modelValue'], emits: ['update:modelValue'], setup(props, { emit }) {
    return () => h('input', { value: props.modelValue, onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value) })
  } }) }
})
vi.mock('@/components/ui/label', async () => {
  const { defineComponent, h } = await import('vue')
  return { Label: defineComponent({ setup(_, { slots }) { return () => h('label', slots.default?.()) } }) }
})

const result = (data: unknown) => ({ data: { data } })
const active = { current_path: '/admin', saved_path: '/admin', can_change: true, can_restart: false, restart_required: false, block_reason: '' }
async function flush() { for (let i = 0; i < 20; i++) await Promise.resolve(); await nextTick() }

describe('admin entry settings', () => {
  let container: HTMLDivElement
  let unmount: (() => void) | undefined
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    mocks.fetch.mockReset()
    const fakeWindow = Object.create(window)
    Object.defineProperty(fakeWindow, 'location', { value: { origin: 'https://shop.example', assign: mocks.assign } })
    vi.stubGlobal('window', fakeWindow)
    vi.stubGlobal('fetch', mocks.fetch)
    mocks.getAdminPath.mockResolvedValue(result({ ...active }))
    mocks.updateAdminPath.mockResolvedValue(result({ saved_path: '/private-console', restart_required: true }))
    mocks.restartSystemService.mockResolvedValue(result({ restarting: true }))
    mocks.confirmAction.mockResolvedValue(true)
  })
  afterEach(() => { unmount?.(); unmount = undefined; container?.remove(); vi.unstubAllGlobals(); vi.useRealTimers() })
  async function mount() {
    container = document.createElement('div')
    document.body.append(container)
    const app = createApp(SettingsAdminPath)
    app.mount(container)
    unmount = () => app.unmount()
    await flush()
  }
  async function edit(path: string) {
    const input = container.querySelector('input')!
    input.value = path
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await flush()
  }
  const button = (key: string) => [...container.querySelectorAll('button')].find(node => node.textContent === `admin.adminPath.${key}`)!

  it('saves without exiting an unmanaged service and keeps the new URL visible', async () => {
    await mount()
    await edit('/private-console')
    button('save').click()
    await flush()
    expect(mocks.confirmAction).toHaveBeenCalledWith(expect.objectContaining({ description: expect.stringContaining('https://shop.example/private-console/login') }))
    expect(mocks.updateAdminPath).toHaveBeenCalledWith({ path: '/private-console' })
    expect(mocks.restartSystemService).not.toHaveBeenCalled()
    expect(container.querySelector('a')?.href).toBe('https://shop.example/private-console/login')
    expect(container.textContent).toContain('admin.adminPath.manualRestart')
  })

  it('waits for both the new running path and its actual admin HTML before redirecting', async () => {
    mocks.getAdminPath.mockResolvedValue(result({ ...active, can_restart: true }))
    mocks.fetch
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { current_path: '/admin' } }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { current_path: '/private-console' } }) })
      .mockResolvedValueOnce({ ok: true, text: async () => '<html>storefront fallback</html>' })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { current_path: '/private-console' } }) })
      .mockResolvedValueOnce({ ok: true, text: async () => '<base href="/private-console/">' })
    await mount()
    await edit('/private-console')
    button('saveRestart').click()
    await flush()
    expect(mocks.restartSystemService).toHaveBeenCalledOnce()
    expect(mocks.assign).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(2000)
    expect(mocks.assign).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(2000)
    expect(mocks.assign).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(2000)
    expect(mocks.assign).toHaveBeenCalledWith('https://shop.example/private-console/login')
  })

  it('retains the pending URL if restart fails and allows retrying', async () => {
    mocks.getAdminPath.mockResolvedValue(result({ ...active, can_restart: true }))
    mocks.restartSystemService.mockRejectedValue(new Error('unavailable'))
    await mount()
    await edit('/private-console')
    button('saveRestart').click()
    await flush()
    expect(container.textContent).toContain('admin.adminPath.restartFailed')
    expect(container.querySelector('a')?.href).toContain('/private-console/login')
    expect(button('restart').disabled).toBe(false)
    expect(mocks.assign).not.toHaveBeenCalled()
  })

  it('does not write after invalid input or a cancelled confirmation', async () => {
    await mount()
    await edit('//external.example')
    button('save').click()
    await flush()
    expect(mocks.confirmAction).not.toHaveBeenCalled()
    expect(mocks.updateAdminPath).not.toHaveBeenCalled()
    await edit('/private-console')
    mocks.confirmAction.mockResolvedValue(false)
    button('save').click()
    await flush()
    expect(mocks.updateAdminPath).not.toHaveBeenCalled()
  })

  it('disables changes when an environment variable overrides the path', async () => {
    mocks.getAdminPath.mockResolvedValue(result({ ...active, can_change: false, block_reason: 'environment_override' }))
    await mount()
    expect(container.querySelector('input')?.disabled).toBe(true)
    expect(button('save').disabled).toBe(true)
    expect(container.textContent).toContain('admin.adminPath.environment_override')
  })

  it('keeps the saved address and allows retrying after restart times out', async () => {
    mocks.getAdminPath.mockResolvedValue(result({ ...active, can_restart: true }))
    mocks.fetch.mockRejectedValue(new Error('service offline'))
    await mount()
    await edit('/private-console')
    button('saveRestart').click()
    await flush()
    await vi.advanceTimersByTimeAsync(120000)
    expect(container.textContent).toContain('admin.adminPath.restartTimeout')
    expect(container.querySelector('a')?.href).toContain('/private-console/login')
    expect(button('restart').disabled).toBe(false)
    expect(mocks.assign).not.toHaveBeenCalled()
  })

  it('stops restart checks when leaving the settings page', async () => {
    mocks.getAdminPath.mockResolvedValue(result({ ...active, can_restart: true }))
    await mount()
    await edit('/private-console')
    button('saveRestart').click()
    await flush()
    unmount!()
    unmount = undefined
    await vi.advanceTimersByTimeAsync(10000)
    expect(mocks.fetch).not.toHaveBeenCalled()
    expect(mocks.assign).not.toHaveBeenCalled()
  })
})
