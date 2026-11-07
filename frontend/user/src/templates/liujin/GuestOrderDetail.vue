<template>
  <div class="lj-page lj-stack">
    <header class="lj-heading">
      <div><p class="lj-eyebrow">{{ t('liujin.orderEyebrow') }}</p><h1>{{ t('liujin.detailTitle') }}</h1><p class="lj-muted">{{ t('liujin.detailHint') }}</p></div>
      <RouterLink to="/guest/orders" class="lj-back"><ArrowLeft :size="16" />{{ t('guestOrderDetail.backSearch') }}</RouterLink>
    </header>
    <div v-if="viewState === 'loading'" class="lj-panel h-64 animate-pulse" :aria-label="t('common.loading')" aria-busy="true"></div>
    <section v-else-if="viewState === 'auth'" class="lj-panel">
      <h2 class="lj-section-title">{{ t('guestOrderDetail.authTitle') }}</h2>
      <p class="lj-muted mb-5">{{ t('guestOrderDetail.authHint') }}</p>
      <form class="grid gap-4 sm:grid-cols-2" @submit.prevent="handleAuthSubmit">
        <label for="lj-auth-email" class="lj-field">{{ t('liujin.email') }}<Input id="lj-auth-email" v-model="auth.email" type="email" autocomplete="email" required class="h-11" /></label>
        <label for="lj-auth-password" class="lj-field">{{ t('liujin.password') }}<Input id="lj-auth-password" v-model="auth.order_password" type="password" autocomplete="current-password" required class="h-11" /></label>
        <p v-if="authError" class="lj-error sm:col-span-2" role="alert">{{ authError }}</p>
        <div class="lj-actions sm:col-span-2"><Button type="submit">{{ t('guestOrderDetail.authSubmit') }}</Button><Button type="button" variant="outline" @click="clearAuth">{{ t('guestOrderDetail.authClear') }}</Button></div>
      </form>
    </section>
    <OrderContent v-else-if="viewState === 'detail' && order" :order="order" guest :downloading="fulfillmentDownloading" @download="handleDownloadFulfillment" @refresh="refreshOrder" />
    <div v-else class="lj-panel text-center"><p class="lj-muted">{{ t('guestOrderDetail.notFound') }}</p><Button variant="outline" class="mt-4" @click="refreshOrder">{{ t('errorBoundary.retry') }}</Button></div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useGuestOrderDetail } from '../../composables/useGuestOrderDetail'
import OrderContent from './components/OrderContent.vue'
const { t } = useI18n()
const { order, authError, auth, viewState, handleAuthSubmit, clearAuth, fulfillmentDownloading, handleDownloadFulfillment, refreshOrder } = useGuestOrderDetail()
</script>
