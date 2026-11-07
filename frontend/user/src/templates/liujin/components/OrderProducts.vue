<template>
  <div>
    <div v-for="(item, index) in items" :key="index" class="lj-item">
      <img v-if="orderItemImage(item)" :src="orderItemImage(item)" alt="" class="lj-item-image" loading="lazy" />
      <div v-else class="lj-item-image grid place-items-center text-muted-foreground"><Package :size="24" :stroke-width="1.5" /></div>
      <div class="min-w-0">
        <h3>{{ getLocalizedText(item.title) || t('liujin.productFallback') }}</h3>
        <p class="lj-muted mt-1">{{ orderItemSkuText(item) }} · {{ t('orderDetail.quantityLabel') }} {{ item.quantity }}</p>
      </div>
    </div>
    <p v-if="!items.length" class="lj-muted">{{ t('orderDetail.noItems') }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Package } from 'lucide-vue-next'
import { useOrderDisplayHelpers } from '../../../composables/useOrderDisplayHelpers'
defineProps<{ items: any[] }>()
const { t } = useI18n()
const { getLocalizedText, orderItemImage, orderItemSkuText } = useOrderDisplayHelpers(ref(null))
</script>
