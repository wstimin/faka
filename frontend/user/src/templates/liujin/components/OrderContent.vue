<template>
  <div class="lj-stack">
    <section class="lj-state" :data-ready="state === 'ready'">
      <h2><component :is="state === 'ready' ? CircleCheck : state === 'waiting' || state === 'partial' ? Clock3 : Info" :size="24" />{{ t(`liujin.states.${state}`) }}</h2>
      <p class="lj-muted">{{ t(`liujin.hints.${state}`) }}</p>
      <div class="lj-actions mt-4">
        <Button v-if="allText" @click="copy(allText, 'all')"><component :is="copied === 'all' ? Check : Copy" />{{ copied === 'all' ? t('liujin.copied') : t('liujin.copyDelivered') }}</Button>
        <Button v-if="order.status === 'pending_payment'" as-child><RouterLink :to="{ path: '/pay', query: { order_no: order.order_no, ...(guest ? { guest: '1' } : {}) } }">{{ t('orders.payNow') }}</RouterLink></Button>
        <Button v-if="['waiting', 'partial'].includes(state)" variant="outline" @click="emit('refresh')"><RefreshCw />{{ t('payment.refreshStatus') }}</Button>
        <Button v-if="!guest && order.status === 'pending_payment'" variant="outline" @click="emit('cancel')">{{ t('orderDetail.cancel') }}</Button>
      </div>
    </section>

    <section aria-labelledby="lj-content-title">
      <h2 id="lj-content-title" class="lj-section-title">{{ t('liujin.yourContent') }}</h2>
      <p class="lj-muted mb-5">{{ t('liujin.contentHint') }}</p>
      <div class="lj-stack">
        <article v-for="(group, index) in groups" :key="group.order_no" class="lj-panel">
          <div class="flex flex-wrap justify-between gap-4">
            <OrderProducts :items="orderItems(group)" class="min-w-0 flex-1" />
            <Badge :variant="group.fulfillment?.status === 'delivered' ? 'success' : 'warning'">{{ group.fulfillment?.status === 'delivered' ? t('liujin.delivered') : t(`liujin.states.${deliveryState(group)}`) }}</Badge>
          </div>
          <div v-if="group.fulfillment?.status === 'delivered' && deliveryText(group.fulfillment)" class="lj-delivery">
            <div class="lj-delivery-bar">
              <strong><PackageCheck :size="20" aria-hidden="true" />{{ t('liujin.contentNumber', { n: index + 1 }) }}</strong>
              <div class="lj-actions">
                <Button class="lj-delivery-view" @click="openDelivery(group)"><Eye />{{ t('common.viewDetails') }}</Button>
                <Button v-if="isDeliveryTruncated(group.fulfillment)" class="lj-delivery-copy" :disabled="downloading" @click="emit('download', group.order_no)"><Download />{{ downloading ? t('orderDetail.fulfillmentDownloading') : t('orderDetail.fulfillmentDownload') }}</Button>
                <Button v-else class="lj-delivery-copy" @click="copy(deliveryText(group.fulfillment), group.order_no)"><component :is="copied === group.order_no ? Check : Copy" />{{ copied === group.order_no ? t('liujin.copied') : t('liujin.copyContent') }}</Button>
              </div>
            </div>
            <p v-if="isDeliveryTruncated(group.fulfillment)" class="lj-delivery-notice">{{ t('liujin.truncated') }}</p>
            <pre class="lj-payload lj-payload-preview" tabindex="0" :aria-label="t('liujin.deliveryDetails')">{{ deliveryText(group.fulfillment) }}</pre>
          </div>
          <p v-else class="lj-muted mt-5 rounded-lg bg-secondary p-4">{{ group.fulfillment?.status === 'delivered' ? t('liujin.noContent') : t('liujin.waitingContent') }}</p>
          <div v-if="group.fulfillment?.status === 'delivered'" class="mt-4 space-y-3">
            <details v-for="(block, blockIndex) in instructionBlocks(orderItems(group))" :key="blockIndex" class="lj-details border-t pt-3">
              <summary>{{ t('orderDetail.instructionsTitle') }}<span v-if="block.title"> · {{ block.title }}</span></summary>
              <div class="prose prose-sm max-w-none dark:prose-invert" v-html="block.html"></div>
            </details>
          </div>
        </article>
      </div>
      <p v-if="copied" role="status" class="lj-feedback">{{ copied === 'number' ? t('liujin.copiedOrderNo') : t('liujin.copiedContent') }}</p>
      <p v-if="copyError" role="alert" class="lj-error">{{ copyError }}</p>
    </section>

    <dialog ref="deliveryDialog" class="lj-delivery-dialog" aria-labelledby="lj-delivery-dialog-title" @close="detailGroup = null" @click.self="deliveryDialog?.close()">
      <header class="lj-delivery-dialog-header">
        <h2 id="lj-delivery-dialog-title"><PackageCheck :size="22" aria-hidden="true" />{{ t('liujin.deliveryDetails') }}</h2>
        <Button variant="ghost" size="icon" :aria-label="t('announcement.close')" @click="deliveryDialog?.close()"><X /></Button>
      </header>
      <div v-if="detailGroup" class="lj-delivery-dialog-body">
        <OrderProducts :items="orderItems(detailGroup)" />
        <p v-if="isDeliveryTruncated(detailGroup.fulfillment)" class="lj-muted">{{ t('liujin.truncated') }}</p>
        <pre class="lj-payload" tabindex="0" :aria-label="t('liujin.deliveryDetails')">{{ deliveryText(detailGroup.fulfillment) }}</pre>
        <div class="lj-actions">
          <Button v-if="isDeliveryTruncated(detailGroup.fulfillment)" :disabled="downloading" @click="emit('download', detailGroup.order_no)"><Download />{{ downloading ? t('orderDetail.fulfillmentDownloading') : t('orderDetail.fulfillmentDownload') }}</Button>
          <Button v-else @click="copy(deliveryText(detailGroup.fulfillment), detailGroup.order_no)"><component :is="copied === detailGroup.order_no ? Check : Copy" />{{ copied === detailGroup.order_no ? t('liujin.copied') : t('liujin.copyContent') }}</Button>
        </div>
        <p v-if="copied === detailGroup.order_no" role="status" class="lj-feedback">{{ t('liujin.copiedContent') }}</p>
        <p v-if="copyError" role="alert" class="lj-error">{{ copyError }}</p>
      </div>
    </dialog>

    <section class="lj-panel">
      <h2 class="lj-section-title">{{ t('liujin.orderInfo') }}</h2>
      <dl class="lj-facts mt-5">
        <div><dt>{{ t('orders.orderNo') }}</dt><dd class="flex flex-wrap items-center gap-2"><span>{{ order.order_no }}</span><Button variant="outline" @click="copy(order.order_no, 'number')">{{ copied === 'number' ? t('liujin.copied') : t('liujin.copyOrderNo') }}</Button></dd></div>
        <div><dt>{{ t('liujin.amountDue') }}</dt><dd class="text-xl font-semibold tabular-nums">{{ formatMoney(order.total_amount, order.currency) }}</dd></div>
        <div><dt>{{ t('orderDetail.createdAtLabel') }}</dt><dd>{{ formatDate(order.created_at) }}</dd></div>
        <div><dt>{{ t('liujin.orderStatus') }}</dt><dd>{{ statusLabel(order.status) }}</dd></div>
        <div v-if="order.paid_at"><dt>{{ t('orderDetail.paidAtLabel') }}</dt><dd>{{ formatDate(order.paid_at) }}</dd></div>
      </dl>
    </section>

    <details class="lj-panel lj-details">
      <summary>{{ t('liujin.moreDetails') }}</summary>
      <div class="space-y-4">
        <VaultOrderItem v-for="(item, index) in orderItems(order)" :key="index" :item="item" :currency="order.currency" />
        <dl class="lj-facts border-t pt-5">
          <div v-for="field in amountFields" :key="field.key"><dt>{{ t(field.label) }}</dt><dd>{{ formatMoney(order[field.key], order.currency) }}</dd></div>
          <div v-if="order.expires_at"><dt>{{ t('orderDetail.expiresAtLabel') }}</dt><dd>{{ formatDate(order.expires_at) }}</dd></div>
          <div v-if="order.canceled_at"><dt>{{ t('orderDetail.canceledAtLabel') }}</dt><dd>{{ formatDate(order.canceled_at) }}</dd></div>
        </dl>
        <div v-for="(record, index) in order.refund_records || []" :key="index" class="lj-muted border-t pt-4">{{ formatDate(record.created_at) }} · {{ formatMoney(record.amount, record.currency || order.currency) }} · {{ refundReasonText(record.remark) }}</div>
      </div>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, CircleCheck, Clock3, Copy, Download, Eye, Info, PackageCheck, RefreshCw, X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useOrderDisplayHelpers } from '../../../composables/useOrderDisplayHelpers'
import VaultOrderItem from '../../vault/components/VaultOrderItem.vue'
import OrderProducts from './OrderProducts.vue'
import { copyableDeliveryText, deliveryGroups, deliveryState, deliveryText, isDeliveryTruncated, orderItems, type StoreOrder } from '../orderDelivery'
import { useDeliveryCopy } from '../useDeliveryCopy'

const props = defineProps<{ order: StoreOrder; guest?: boolean; downloading: boolean }>()
const emit = defineEmits<{ download: [orderNo: string]; refresh: []; cancel: [] }>()
const { t } = useI18n()
const { copied, copyError, copy } = useDeliveryCopy()
const deliveryDialog = ref<HTMLDialogElement | null>(null)
const detailGroup = ref<StoreOrder | null>(null)
const openDelivery = (group: StoreOrder) => {
  detailGroup.value = group
  deliveryDialog.value?.showModal()
}
const { instructionBlocks, statusLabel, formatMoney, formatDate, refundReasonText, hasAmount } = useOrderDisplayHelpers(toRef(props, 'order'))
const groups = computed(() => deliveryGroups(props.order))
const state = computed(() => deliveryState(props.order))
const allText = computed(() => copyableDeliveryText(props.order))
const amountFields = computed(() => [
  { key: 'original_amount', label: 'orderDetail.amountOriginal' },
  { key: 'discount_amount', label: 'orderDetail.amountDiscount' },
  { key: 'promotion_discount_amount', label: 'orderDetail.promotionDiscountLabel' },
  { key: 'member_discount_amount', label: 'orderDetail.amountMemberDiscount' },
  { key: 'wholesale_discount_amount', label: 'orderDetail.amountWholesaleDiscount' },
  { key: 'wallet_paid_amount', label: 'orderDetail.amountWalletPaid' },
  { key: 'online_paid_amount', label: 'orderDetail.amountOnlinePaid' },
  { key: 'refunded_amount', label: 'orderDetail.amountRefunded' },
].filter(field => field.key === 'original_amount' || hasAmount(props.order[field.key])))
</script>
