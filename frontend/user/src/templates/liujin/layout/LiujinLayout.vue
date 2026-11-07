<template>
  <VaultLayout class="liujin-scope" show-order-label><slot /></VaultLayout>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import VaultLayout from '../../vault/layout/VaultLayout.vue'
import '../styles/liujin.css'

// 同时覆盖 Teleport 到 body 的公告、提示和确认弹窗。
onMounted(() => document.body.classList.add('liujin-tokens'))
onUnmounted(() => document.body.classList.remove('liujin-tokens'))
</script>
