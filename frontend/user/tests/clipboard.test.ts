import test from 'node:test'
import assert from 'node:assert/strict'
import { copyText } from '../src/utils/clipboard.ts'

test('clipboard reports a rejected legacy copy and always removes its temporary input', async () => {
  const originalNavigator = Object.getOwnPropertyDescriptor(globalThis, 'navigator')
  const originalDocument = Object.getOwnPropertyDescriptor(globalThis, 'document')
  let removed = false
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: {} })
  Object.defineProperty(globalThis, 'document', { configurable: true, value: {
    createElement: () => ({ style: {}, focus() {}, select() {} }),
    body: { appendChild() {}, removeChild() { removed = true } },
    execCommand: () => false,
  } })
  try {
    await assert.rejects(copyText('example'), /Clipboard copy failed/)
    assert.equal(removed, true)
  } finally {
    if (originalNavigator) Object.defineProperty(globalThis, 'navigator', originalNavigator)
    else Reflect.deleteProperty(globalThis, 'navigator')
    if (originalDocument) Object.defineProperty(globalThis, 'document', originalDocument)
    else Reflect.deleteProperty(globalThis, 'document')
  }
})
