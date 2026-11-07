<template>
  <div class="lj-page lj-stack">
    <header class="lj-heading">
      <div><p class="lj-eyebrow">{{ t('liujin.orderEyebrow') }}</p><h1>{{ t('guestOrders.title') }}</h1><p class="lj-muted">{{ t('liujin.searchHint') }}</p></div>
      <RouterLink to="/products" class="lj-back"><ArrowLeft :size="16" />{{ t('orders.continueShopping') }}</RouterLink>
    </header>
    <details class="lj-panel lj-details" :open="!orders.length">
      <summary>{{ orders.length ? t('liujin.changeSearch') : t('guestOrders.search') }}</summary>
      <div v-if="hasSavedAuth" class="mb-4 flex flex-wrap justify-between gap-2">
        <p class="lj-muted">{{ t('guestOrders.savedHint', { email: savedAuth.email }) }}</p>
        <Button variant="ghost" :disabled="loading" @click="clearSaved">{{ t('guestOrders.clearSaved') }}</Button>
      </div>
      <form class="lj-form" @submit.prevent="handleSearch">
        <label class="lj-field" for="lj-search-email">{{ t('liujin.email') }}<Input id="lj-search-email" v-model="email" type="email" autocomplete="email" required class="h-11" /></label>
        <label class="lj-field" for="lj-search-password">{{ t('liujin.password') }}<Input id="lj-search-password" v-model="orderPassword" type="password" autocomplete="current-password" required class="h-11" /></label>
        <label class="lj-field" for="lj-search-number">{{ t('liujin.optionalOrderNo') }}<Input id="lj-search-number" v-model="orderNo" type="text" class="h-11" /></label>
        <Button type="submit" :disabled="loading" class="h-11"><Search :size="16" />{{ loading ? t('guestOrders.searching') : t('guestOrders.search') }}</Button>
      </form>
      <p v-if="error" class="lj-error" role="alert">{{ error }}</p>
    </details>
    <div v-if="loading" :aria-label="t('common.loading')" aria-busy="true" class="lj-stack"><div v-for="n in 3" :key="n" class="lj-panel h-40 animate-pulse"></div></div>
    <template v-else-if="orders.length">
      <div class="flex flex-wrap justify-between gap-2"><h2 class="lj-section-title">{{ t('liujin.purchases') }}</h2><p class="lj-muted">{{ t('liujin.orderCount', { count: pagination.total }) }}</p></div>
      <OrderTable :orders="orders" guest />
      <nav v-if="pagination.total_page > 1" class="lj-actions justify-center" :aria-label="t('orders.title')">
        <Button variant="outline" :disabled="pagination.page <= 1" :aria-label="t('liujin.previous')" @click="changePage(pagination.page - 1)"><ChevronLeft /></Button>
        <span class="lj-muted">{{ pagination.page }} / {{ pagination.total_page }}</span>
        <Button variant="outline" :disabled="pagination.page >= pagination.total_page" :aria-label="t('liujin.next')" @click="changePage(pagination.page + 1)"><ChevronRight /></Button>
      </nav>
    </template>
    <div v-else class="lj-panel py-12 text-center"><PackageOpen class="mx-auto mb-4 text-muted-foreground" :size="32" /><p class="lj-muted">{{ emptyMessage }}</p></div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowLeft, ChevronLeft, ChevronRight, PackageOpen, Search } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useGuestOrders } from '../../composables/useGuestOrders'
import OrderTable from './components/OrderTable.vue'
const { t } = useI18n()
const { savedAuth, email, orderPassword, orderNo, loading, error, orders, pagination, hasSavedAuth, clearSaved, handleSearch, emptyMessage, changePage } = useGuestOrders()
</script>
