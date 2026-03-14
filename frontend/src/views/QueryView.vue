<!-- 查询页：容器页面，承载核心组件 QueryCard -->
<template>
  <div class="query-page">
    <QueryCard />
    <div class="footer">验证码查询系统 v1.0</div>
  </div>
</template>

<script setup lang="ts">
// 此页面仅负责布局与承载核心查询组件
import QueryCard from '../components/QueryCard.vue'
</script>

<style scoped>
.query-page { max-width: 500px; margin: 40px auto; padding: 0 20px; }
.footer { text-align: center; color: #999; font-size: 14px; margin-top: 40px; }
</style>