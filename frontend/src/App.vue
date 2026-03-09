<script setup lang="ts">
import { RouterView } from "vue-router";
import NotificationContainer from "./components/Notification/NotificationContainer.vue";
import { onMounted, ref } from "vue";
import notificationService from "./components/Notification/notificationService";

const notificationContainer = ref();

onMounted(() => {
  notificationService.registerContainer(notificationContainer.value);
});
</script>

<template>
  <RouterView v-slot="{ Component }">
    <keep-alive include="HomeView">
      <component :is="Component" />
    </keep-alive>
  </RouterView>
  <NotificationContainer ref="notificationContainer" />
</template>
