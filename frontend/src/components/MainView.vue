<script setup lang="ts"></script>

<template>
  <div class="card">
    <slot></slot>
  </div>
</template>

<style scoped>
/* -------------- 卡片外壳 -------------- */
.card {
  position: relative;
  width: 95%;
  max-width: 800px;
  margin: 0 auto;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 16px;
  padding: 2.2rem 1.5rem 1.5rem;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}
</style>
