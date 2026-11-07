<template>
  <div class="lj-order-table-scroll" role="region" :aria-label="t('orders.title')" tabindex="0">
    <table class="lj-order-table">
      <thead><tr>
        <th scope="col">{{ t('orders.orderNo') }}</th>
        <th scope="col">{{ t('liujin.table.product') }}</th>
        <th scope="col">{{ t('liujin.table.quantityAmount') }}</th>
        <th scope="col">{{ t('liujin.table.actions') }}</th>
        <th scope="col">{{ t('liujin.table.payment') }}</th>
        <th scope="col">{{ t('liujin.table.shipment') }}</th>
      </tr></thead>
      <tbody><OrderCard v-for="order in orders" :key="order.order_no" :order="order" :guest="guest" /></tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { StoreOrder } from '../orderDelivery'
import OrderCard from './OrderCard.vue'
defineProps<{ orders: StoreOrder[]; guest?: boolean }>()
const { t } = useI18n()
</script>
