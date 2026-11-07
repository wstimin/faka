<template>
  <tr class="lj-order-row" :data-order-no="order.order_no">
    <td><span class="lj-table-number">{{ order.order_no }}</span><small class="lj-table-date">{{ formatDate(order.created_at) }}</small></td>
    <td>
      <div v-for="(item, index) in items" :key="index" class="lj-table-product" :title="`${getLocalizedText(item.title)} · ${orderItemSkuText(item)} · ${item.quantity}`">
        <Package :size="15" aria-hidden="true" /><span>{{ getLocalizedText(item.title) || t('liujin.productFallback') }}</span>
      </div>
    </td>
    <td><small class="lj-table-quantity">{{ t('orderDetail.quantityLabel') }} {{ quantity }}</small><strong class="lj-table-amount">{{ formatMoney(order.total_amount, order.currency) }}</strong></td>
    <td>
      <div class="lj-table-actions">
        <Button variant="outline" size="sm" @click="openDetails"><Eye />{{ canCopy ? t('liujin.viewCards') : t('orders.viewDetails') }}</Button>
        <Button v-if="order.status === 'pending_payment'" as-child><RouterLink :to="{ path: '/pay', query: { order_no: order.order_no, ...(guest ? { guest: '1' } : {}) } }">{{ t('orders.payNow') }}</RouterLink></Button>
      </div>
    <dialog ref="detailsDialog" class="lj-delivery-dialog lj-order-dialog" :aria-labelledby="dialogTitleId" @close="detailsOrder = null" @click.self="detailsDialog?.close()">
      <header class="lj-card-dialog-header">
        <h2 :id="dialogTitleId"><Eye :size="16" aria-hidden="true" />{{ t('liujin.viewCards') }}</h2>
        <Button variant="ghost" size="icon" :aria-label="t('announcement.close')" @click="detailsDialog?.close()"><X /></Button>
      </header>
      <div class="lj-card-dialog-body">
        <div v-if="detailsLoading" class="h-48 animate-pulse" :aria-label="t('common.loading')" aria-busy="true"></div>
        <div v-else-if="detailsError" class="lj-stack"><p class="lj-error" role="alert">{{ detailsError }}</p><Button variant="outline" @click="loadDetails">{{ t('errorBoundary.retry') }}</Button></div>
        <template v-else-if="detailsOrder">
          <p v-if="isTruncated" class="lj-muted">{{ t('liujin.truncated') }}</p>
          <p v-else-if="deliveryState(detailsOrder) === 'partial'" class="lj-muted">{{ t('liujin.hints.partial') }}</p>
          <pre v-if="previewText" class="lj-card-content" tabindex="0" :aria-label="t('liujin.deliveryDetails')">{{ previewText }}</pre>
          <p v-else class="lj-muted">{{ t(`liujin.hints.${deliveryState(detailsOrder)}`) }}</p>
          <div v-if="previewText" class="lj-card-dialog-actions">
            <Button variant="outline" size="sm" :disabled="!completeText" @click="copy(completeText, 'content')"><component :is="copied === 'content' ? Check : Copy" />{{ copied === 'content' ? t('liujin.copied') : t('liujin.copyContent') }}</Button>
            <Button size="sm" :disabled="downloading" @click="downloadFulfillment"><Download />{{ downloading ? t('orderDetail.fulfillmentDownloading') : t('liujin.downloadContent') }}</Button>
          </div>
          <p v-if="copied" class="lj-feedback" role="status">{{ t('liujin.copiedContent') }}</p>
          <p v-if="copyError" class="lj-error" role="alert">{{ copyError }}</p>
          <p v-if="downloadError" class="lj-error" role="alert">{{ downloadError }}</p>
        </template>
      </div>
    </dialog>
    </td>
    <td><Badge :variant="paymentState === 'paid' ? 'success' : paymentState === 'unpaid' ? 'warning' : 'neutral'">{{ t(`liujin.payment.${paymentState}`) }}</Badge></td>
    <td><Badge :variant="shipmentState === 'delivered' ? 'success' : shipmentState === 'unknown' ? 'neutral' : 'warning'">{{ t(`liujin.shipment.${shipmentState}`) }}</Badge></td>
  </tr>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy, Download, Eye, Package, X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { guestOrderAPI, userOrderAPI } from '../../../api'
import { loadGuestOrderAuth } from '../../../utils/guestOrderAuth'
import { useOrderDisplayHelpers } from '../../../composables/useOrderDisplayHelpers'
import { copyableDeliveryText, deliveredGroups, deliveryState, deliveryText, isDeliveryTruncated, orderItems, orderPaymentState, orderShipmentState, type StoreOrder } from '../orderDelivery'
import { useDeliveryCopy } from '../useDeliveryCopy'

const props = defineProps<{ order: StoreOrder; guest?: boolean }>()
const { t } = useI18n()
const { formatMoney, formatDate, getLocalizedText, orderItemSkuText } = useOrderDisplayHelpers(ref(null))
const { copied, copyError, copy } = useDeliveryCopy()
const items = computed(() => orderItems(props.order))
const quantity = computed(() => items.value.reduce((sum, item) => sum + (Number(item.quantity) || 0), 0))
const paymentState = computed(() => orderPaymentState(props.order))
const shipmentState = computed(() => orderShipmentState(props.order))
const detailsDialog = ref<HTMLDialogElement | null>(null)
const dialogTitleId = useId()
const detailsOrder = ref<StoreOrder | null>(null)
const detailsGroups = computed(() => detailsOrder.value ? deliveredGroups(detailsOrder.value) : [])
const previewText = computed(() => detailsGroups.value.map(group => deliveryText(group.fulfillment)).join('\n\n'))
const completeText = computed(() => detailsOrder.value ? copyableDeliveryText(detailsOrder.value) : '')
const isTruncated = computed(() => detailsGroups.value.some(group => isDeliveryTruncated(group.fulfillment)))
const detailsLoading = ref(false)
const detailsError = ref('')
const downloading = ref(false)
const downloadError = ref('')
const canCopy = computed(() => ['delivered', 'completed', 'partially_delivered', 'partially_refunded'].includes(props.order.status))
const fetchOrder = async (): Promise<StoreOrder> => {
  const result = props.guest
    ? await guestOrderAPI.detail(props.order.order_no, loadGuestOrderAuth())
    : await userOrderAPI.detail(props.order.order_no)
  if (!result.data.data) throw new Error('order_not_found')
  return result.data.data
}
const loadDetails = async () => {
  if (detailsLoading.value) return
  detailsLoading.value = true
  detailsError.value = ''
  try {
    detailsOrder.value = await fetchOrder()
  } catch {
    detailsError.value = t('liujin.contentLoadFailed')
  } finally {
    detailsLoading.value = false
  }
}
const openDetails = () => {
  detailsOrder.value = null
  downloadError.value = ''
  detailsDialog.value?.showModal()
  void loadDetails()
}
const downloadFulfillment = async () => {
  if (downloading.value) return
  downloading.value = true
  downloadError.value = ''
  try {
    const parts = await Promise.all(detailsGroups.value.map(async (group) => {
      if (!isDeliveryTruncated(group.fulfillment)) return deliveryText(group.fulfillment)
      const result = props.guest
        ? await guestOrderAPI.downloadFulfillment(group.order_no, loadGuestOrderAuth())
        : await userOrderAPI.downloadFulfillment(group.order_no)
      return (result.data as Blob).text()
    }))
    const url = URL.createObjectURL(new Blob([parts.join('\n\n')], { type: 'text/plain; charset=utf-8' }))
    const link = document.createElement('a')
    link.href = url
    link.download = `fulfillment-${props.order.order_no}.txt`
    link.click()
    URL.revokeObjectURL(url)
  } catch {
    downloadError.value = t('orderDetail.fulfillmentDownloadFailed')
  } finally {
    downloading.value = false
  }
}
</script>
