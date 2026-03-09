<script setup lang="ts">
import {ref} from "vue";
import Notification from "./Notification.vue";

export interface NotificationOptions {
  id?: string;
  type?: "success" | "error" | "warning" | "info";
  message: string;
  duration?: number;
  closable?: boolean;
  showIcon?: boolean;
}

interface NotificationInstance extends NotificationOptions {
  id: string;
}

const notifications = ref<NotificationInstance[]>([]);

const addNotification = (options: NotificationOptions) => {
  const id = options.id || Date.now().toString();
  notifications.value.push({
    ...options,
    id,
  });

  // 如果设置了 duration，自动移除通知
  if (options.duration !== undefined && options.duration > 0) {
    setTimeout(() => {
      removeNotification(id);
    }, options.duration + 300); // 加上动画时间
  }
};

const removeNotification = (id: string) => {
  const index = notifications.value.findIndex(
      (notification) => notification.id === id,
  );
  if (index !== -1) {
    notifications.value.splice(index, 1);
  }
};

// 提供一些快捷方法
const success = (
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
) => {
  addNotification({
    type: "success",
    message,
    ...options,
  });
};

const error = (
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
) => {
  addNotification({
    type: "error",
    message,
    ...options,
  });
};

const warning = (
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
) => {
  addNotification({
    type: "warning",
    message,
    ...options,
  });
};

const info = (
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
) => {
  addNotification({
    type: "info",
    message,
    ...options,
  });
};

defineExpose({
  addNotification,
  removeNotification,
  success,
  error,
  warning,
  info,
});
</script>

<template>
  <div class="notification-container">
    <TransitionGroup name="list" tag="div">
      <Notification
          v-for="notification in notifications"
          :key="notification.id"
          :type="notification.type"
          :message="notification.message"
          :duration="notification.duration"
          :closable="notification.closable"
          :show-icon="notification.showIcon"
          @close="removeNotification(notification.id)"
      />
    </TransitionGroup>
  </div>
</template>

<style scoped>
.notification-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  /*width: 210px;*/
}

.list-enter-active,
.list-leave-active {
  transition: all 0.3s ease;
}

.list-enter-from {
  opacity: 0;
  transform: translateX(30px);
}

.list-leave-to {
  opacity: 0;
  transform: translateX(30px);
}

.list-move {
  transition: transform 0.3s ease;
}
</style>
