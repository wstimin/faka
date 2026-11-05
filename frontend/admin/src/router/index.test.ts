import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  beforeEach: vi.fn(),
  auth: { token: 'token', permissionsLoaded: true, isSuper: false, hasPermission: vi.fn(), loadAuthz: vi.fn(), logout: vi.fn() },
}))
vi.mock('vue-router', () => ({
  createRouter: () => ({ beforeEach: mocks.beforeEach }),
  createWebHistory: vi.fn(),
}))
vi.mock('@/stores/auth', () => ({ useAdminAuthStore: () => mocks.auth }))
vi.mock('@/utils/adminBase', () => ({ ADMIN_BASE: '/admin' }))
import './index'

const paymentRoute = { name: 'payment-channels', path: '/payment-channels', fullPath: '/payment-channels', meta: { requiresAuth: true, permission: 'GET:/admin/payment-channels' } }
const guard = mocks.beforeEach.mock.calls[0]![0]

describe('payment page access after removing the compliance prompt', () => {
  beforeEach(() => {
    mocks.auth.token = 'token'
    mocks.auth.permissionsLoaded = true
    mocks.auth.hasPermission.mockReturnValue(true)
  })

  it('allows an authorized ordinary admin without an acknowledgement', async () => {
    expect(await guard(paymentRoute)).toBe(true)
  })

  it('still requires login', async () => {
    mocks.auth.token = ''
    expect(await guard(paymentRoute)).toEqual({ path: '/login' })
  })

  it('still rejects an admin without payment permissions', async () => {
    mocks.auth.hasPermission.mockReturnValue(false)
    expect(await guard(paymentRoute)).toEqual({ path: '/forbidden', query: { from: '/payment-channels' } })
  })
})
