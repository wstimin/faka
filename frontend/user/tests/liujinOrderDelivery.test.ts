import test from 'node:test'
import assert from 'node:assert/strict'
import { copyableDeliveryText, deliveryGroups, deliveryState, deliveryText, orderItems, orderPaymentState, orderShipmentState } from '../src/templates/liujin/orderDelivery.ts'

test('paid orders without fulfillment await delivery instead of claiming content is ready', () => {
  assert.equal(deliveryState({ order_no: 'demo', status: 'paid' }), 'waiting')
  assert.equal(deliveryState({ order_no: 'demo', status: 'completed' }), 'waiting')
  assert.equal(deliveryState({ order_no: 'demo', status: 'completed' }, true), 'ready')
})

test('table payment and shipment states stay separate for paid, pending and partial orders', () => {
  const pending = { order_no: 'pending', status: 'pending_payment' }
  const waiting = { order_no: 'waiting', status: 'fulfilling' }
  const partial = { order_no: 'partial', status: 'partially_delivered' }
  assert.equal(orderPaymentState(pending), 'unpaid')
  assert.equal(orderShipmentState(pending), 'notShipped')
  assert.equal(orderPaymentState(waiting), 'paid')
  assert.equal(orderShipmentState(waiting), 'waiting')
  assert.equal(orderPaymentState(partial), 'paid')
  assert.equal(orderShipmentState(partial), 'partial')
  assert.equal(orderShipmentState({ order_no: 'done', status: 'completed' }), 'delivered')
})

test('refunds and unknown statuses do not invent an unpaid or delivered state', () => {
  for (const status of ['refunded', 'partially_refunded']) {
    assert.equal(orderPaymentState({ order_no: status, status }), status)
    assert.equal(orderShipmentState({ order_no: status, status }), 'unknown')
  }
  assert.equal(orderPaymentState({ order_no: 'future', status: 'new_status' }), 'unknown')
})

test('mixed deliveries retain the product association and copy only delivered content', () => {
  const order = { order_no: 'demo', status: 'partially_delivered', children: [
    { order_no: 'demo-01', status: 'delivered', items: [{ title: 'Product A' }], fulfillment: { status: 'delivered', payload: 'a@example.invalid----value\nsecond line' } },
    { order_no: 'demo-02', status: 'paid', items: [{ title: 'Product B' }], fulfillment: { status: 'pending', payload: 'not ready' } },
  ] }
  assert.equal(deliveryState(order), 'partial')
  assert.equal(deliveryGroups(order)[0]?.items?.[0].title, 'Product A')
  assert.equal(orderItems(order).length, 2)
  assert.equal(copyableDeliveryText(order), 'a@example.invalid----value\nsecond line')
})

test('terminal and refund states take precedence over historical delivery', () => {
  for (const status of ['pending_payment', 'canceled', 'expired', 'refunded', 'partially_refunded']) {
    assert.equal(deliveryState({ order_no: 'demo', status, fulfillment: { status: 'delivered' } }), status)
  }
})

test('a truncated delivery prevents copying a misleading partial all-content payload', () => {
  const full = { order_no: 'demo-1', status: 'delivered', fulfillment: { status: 'delivered', payload: 'full content', payload_line_count: 1 } }
  const preview = { order_no: 'demo-2', status: 'delivered', fulfillment: { status: 'delivered', payload: 'preview', payload_line_count: 101 } }
  assert.equal(copyableDeliveryText({ order_no: 'demo', status: 'delivered', children: [full, preview] }), '')
  assert.equal(copyableDeliveryText(full), 'full content')
})

test('structured manual deliveries keep labels and notes without guessing credential separators', () => {
  assert.equal(deliveryText({ delivery_data: { note: 'Read first', entries: [{ key: 'License', value: 'ABC---DEF' }, { value: 'more information' }] } }), 'Read first\nLicense: ABC---DEF\nmore information')
  assert.equal(deliveryText({ payload: 'original\r\nraw---content' }), 'original\r\nraw---content')
})

test('parent items already filled from children are not displayed twice', () => {
  const items = [{ title: 'A' }]
  assert.equal(orderItems({ order_no: 'demo', status: 'paid', items, children: [{ order_no: 'child', status: 'paid', items }] }).length, 1)
})

test('multiple complete deliveries can be copied together without pending placeholders', () => {
  const order = { order_no: 'demo', status: 'completed', children: [
    { order_no: 'a', status: 'delivered', fulfillment: { status: 'delivered', payload: 'first' } },
    { order_no: 'b', status: 'completed', fulfillment: { status: 'delivered', payload: 'second' } },
  ] }
  assert.equal(deliveryState(order), 'ready')
  assert.equal(copyableDeliveryText(order), 'first\n\nsecond')
})
