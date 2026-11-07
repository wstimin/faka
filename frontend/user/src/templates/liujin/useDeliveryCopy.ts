import { onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { copyText } from '../../utils/clipboard'
import { toast } from '../../composables/useToast'

export function useDeliveryCopy() {
  const { t } = useI18n()
  const copied = ref('')
  const copyError = ref('')
  let timer: ReturnType<typeof setTimeout> | undefined
  const copy = async (value: string, key: string) => {
    copyError.value = ''
    if (!value) return
    try {
      await copyText(value)
      copied.value = key
      clearTimeout(timer)
      timer = setTimeout(() => { copied.value = '' }, 2200)
    } catch {
      copied.value = ''
      copyError.value = t('liujin.copyFailed')
      toast.error(copyError.value)
    }
  }
  onUnmounted(() => clearTimeout(timer))
  return { copied, copyError, copy }
}
