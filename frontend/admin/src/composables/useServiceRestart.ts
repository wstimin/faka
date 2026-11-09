import { onBeforeUnmount, ref } from 'vue'
import { adminAPI } from '@/api/admin'

export function useServiceRestart(
  isReady: () => Promise<boolean>,
  onReady: () => void,
  onError: (reason: 'restartFailed' | 'restartTimeout', error?: unknown) => void,
) {
  const restarting = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let disposed = false
  onBeforeUnmount(() => { disposed = true; clearTimeout(timer) })

  async function restart() {
    if (restarting.value || disposed) return
    restarting.value = true
    try {
      await adminAPI.restartSystemService()
    } catch (error) {
      restarting.value = false
      onError('restartFailed', error)
      return
    }
    if (disposed) return
    const deadline = Date.now() + 120000
    const poll = async () => {
      if (disposed) return
      if (Date.now() >= deadline) {
        restarting.value = false
        onError('restartTimeout')
        return
      }
      try {
        if (await isReady()) {
          if (!disposed) onReady()
          return
        }
      } catch {
        // Requests may fail while the old process exits and the new one starts.
      }
      if (!disposed) timer = setTimeout(poll, 2000)
    }
    timer = setTimeout(poll, 2000)
  }

  return { restarting, restart }
}
