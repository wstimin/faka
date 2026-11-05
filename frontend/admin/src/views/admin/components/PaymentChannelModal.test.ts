import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import PaymentChannelModal from './PaymentChannelModal.vue'

const mocks = vi.hoisted(() => ({
  getPaymentChannel: vi.fn(), getMemberLevels: vi.fn(),
  createPaymentChannel: vi.fn(), updatePaymentChannel: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: mocks }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('@/components/admin/MediaPicker.vue', () => ({ default: { render: () => null } }))
vi.mock('@/components/ui/multi-select', () => ({ MultiSelect: { render: () => null } }))
vi.mock('@/components/ui/select', async () => {
  const { defineComponent, h } = await import('vue')
  const slot = defineComponent({ setup(_, { slots }) { return () => h('div', slots.default?.()) } })
  return { Select: slot, SelectContent: slot, SelectItem: slot, SelectTrigger: slot, SelectValue: slot }
})
vi.mock('@/components/ui/dialog', async () => {
  const { defineComponent, h } = await import('vue')
  const slot = defineComponent({ setup(_, { slots }) { return () => h('div', slots.default?.()) } })
  return { Dialog: slot, DialogScrollContent: slot, DialogHeader: slot, DialogTitle: slot }
})

async function flush() { for (let i = 0; i < 12; i++) await Promise.resolve(); await nextTick() }

describe('V免签 payment channel saving', () => {
  let container: HTMLDivElement
  let unmount: () => void
  const success = vi.fn()
  const close = vi.fn()
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getMemberLevels.mockResolvedValue({ data: { data: [] } })
    mocks.getPaymentChannel.mockResolvedValue({ data: { data: {
      id: 1, name: 'V免签', provider_type: 'vmq', channel_type: 'alipay', interaction_mode: 'redirect',
      is_active: true, config_json: {
        gateway_url: ' https://vmq.example.com/ ', merchant_key: ' test-key ',
        notify_url: 'https://shop.example.com/api/v1/payments/callback', return_url: 'https://shop.example.com/pay',
      },
    } } })
  })
  afterEach(() => { unmount?.(); container?.remove() })

  async function openEditor() {
    container = document.createElement('div')
    document.body.append(container)
    const channelId = ref<number | null>(null)
    const app = createApp(defineComponent({ setup: () => () => h(PaymentChannelModal, {
      modelValue: true, channelId: channelId.value, onSuccess: success, 'onUpdate:modelValue': close,
    }) }))
    app.mount(container)
    unmount = () => app.unmount()
    channelId.value = 1
    await flush()
  }

  it('submits VMQ fields, prevents duplicate requests and closes after success', async () => {
    let complete!: () => void
    mocks.updatePaymentChannel.mockImplementation(() => new Promise<void>(resolve => { complete = resolve }))
    await openEditor()
    const button = container.querySelector<HTMLButtonElement>('button[type="submit"]')!
    button.click()
    await flush()
    expect(button.disabled).toBe(true)
    expect(button.getAttribute('aria-busy')).toBe('true')
    expect(button.textContent).toBe('admin.settings.actions.saving')
    button.click()
    container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flush()
    expect(mocks.updatePaymentChannel).toHaveBeenCalledTimes(1)
    expect(mocks.updatePaymentChannel).toHaveBeenCalledWith(1, expect.objectContaining({
      provider_type: 'vmq', channel_type: 'alipay', interaction_mode: 'redirect',
      config_json: expect.objectContaining({ gateway_url: 'https://vmq.example.com/', merchant_key: 'test-key' }),
    }))
    expect(close).not.toHaveBeenCalled()
    complete()
    await flush()
    expect(success).toHaveBeenCalledTimes(1)
    expect(close).toHaveBeenCalledWith(false)
    expect(button.disabled).toBe(false)
  })

  it('shows a rejected save inline, keeps the editor open and allows retry', async () => {
    mocks.updatePaymentChannel.mockRejectedValueOnce(new Error('Gateway URL is invalid'))
    await openEditor()
    const button = container.querySelector<HTMLButtonElement>('button[type="submit"]')!
    button.click()
    await flush()
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('Gateway URL is invalid')
    expect(close).not.toHaveBeenCalled()
    expect(success).not.toHaveBeenCalled()
    expect(button.disabled).toBe(false)
    mocks.updatePaymentChannel.mockResolvedValueOnce({})
    button.click()
    await flush()
    expect(success).toHaveBeenCalledTimes(1)
    expect(container.querySelector('[role="alert"]')).toBeNull()
  })
})
