<template>
  <div class="lj-page lj-stack">
    <header class="lj-heading">
      <div><p class="lj-eyebrow">{{ t('liujin.orderEyebrow') }}</p><h1>{{ t('liujin.detailTitle') }}</h1><p class="lj-muted">{{ t('liujin.detailHint') }}</p></div>
      <RouterLink to="/me/orders" class="lj-back"><ArrowLeft :size="16" />{{ t('orders.title') }}</RouterLink>
    </header>
    <div v-if="loading" class="lj-panel h-64 animate-pulse" :aria-label="t('common.loading')" aria-busy="true"></div>
    <OrderContent v-else-if="order" :order="order" :downloading="fulfillmentDownloading" @download="handleDownloadFulfillment" @refresh="debouncedLoadOrder()" @cancel="cancelOrder" />
    <div v-else class="lj-panel text-center"><p class="lj-muted">{{ t('orderDetail.notFound') }}</p><Button variant="outline" class="mt-4" @click="debouncedLoadOrder()">{{ t('errorBoundary.retry') }}</Button></div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useOrderDetail } from '../../composables/useOrderDetail'
import OrderContent from './components/OrderContent.vue'
const { t } = useI18n()
const { loading, order, debouncedLoadOrder, cancelOrder, fulfillmentDownloading, handleDownloadFulfillment } = useOrderDetail()
</script>
