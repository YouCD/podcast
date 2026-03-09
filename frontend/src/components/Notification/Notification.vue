<script setup lang="ts">
import {defineEmits, defineProps, onMounted, onUnmounted, ref} from "vue";

interface Props {
  type?: "success" | "error" | "warning" | "info";
  message: string;
  duration?: number;
  closable?: boolean;
  showIcon?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: "info",
  duration: 3000,
  closable: true,
  showIcon: true,
});

const emit = defineEmits<{
  (e: "close"): void;
}>();

const isVisible = ref(true);
let timer: number | null = null;

const getIconByType = () => {
  switch (props.type) {
    case "success":
      return "✓";
    case "error":
      return "✕";
    case "warning":
      return "⚠";
    case "info":
      return "ℹ";
    default:
      return "ℹ";
  }
};

const getColorClass = () => {
  return `notification--${props.type}`;
};

const closeNotification = () => {
  isVisible.value = false;
  if (timer) {
    clearTimeout(timer);
  }
  emit("close");
};

const startTimer = () => {
  if (props.duration > 0) {
    timer = setTimeout(() => {
      closeNotification();
    }, props.duration) as unknown as number;
  }
};

onMounted(() => {
  startTimer();
});

onUnmounted(() => {
  if (timer) {
    clearTimeout(timer);
  }
});
</script>

<template>
  <transition name="slide-fade">
    <div v-if="isVisible" class="notification" :class="getColorClass()">
      <div class="notification__content">
        <span v-if="showIcon" class="notification__icon">{{
            getIconByType()
          }}</span>
        <span class="notification__message">{{ message }}</span>
      </div>
      <button
          v-if="closable"
          class="notification__close"
          @click="closeNotification"
      >
        ×
      </button>
    </div>
  </transition>
</template>

<style scoped>
.notification {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  font-family: Arial, sans-serif;
  font-size: 14px;
  z-index: 1000;
  margin-bottom: 10px;
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  /*   background: rgba(255, 255, 255, 0.85);
   min-width: 200px;
*/
  color: #333;
}

.notification--success {
  background: rgba(229, 255, 229, 0.9);
  border-color: rgba(45, 183, 45, 0.3);
}

.notification--error {
  background: rgba(255, 229, 229, 0.9);
  border-color: rgba(183, 45, 45, 0.3);
}

.notification--warning {
  background: rgba(255, 245, 229, 0.9);
  border-color: rgba(183, 130, 45, 0.3);
}

.notification--info {
  background: rgba(229, 242, 255, 0.9);
  border-color: rgba(45, 115, 183, 0.3);
}

.notification__content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.notification__icon {
  font-weight: bold;
  font-size: 16px;
}

.notification__message {
  flex: 1;
}

.notification__close {
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background-color 0.2s;
  color: #666;
}

.notification__close:hover {
  background-color: rgba(0, 0, 0, 0.1);
}

/* 过渡动画 */
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.3s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateX(20px);
  opacity: 0;
}

/* 深色模式支持 */
@media (prefers-color-scheme: dark) {
  .notification {
    background: rgba(30, 30, 30, 0.85);
    color: #fff;
    border-color: rgba(255, 255, 255, 0.1);
  }

  .notification--success {
    background: rgba(25, 50, 25, 0.9);
    border-color: rgba(45, 183, 45, 0.3);
    color: #a0ffa0;
  }

  .notification--error {
    background: rgba(50, 25, 25, 0.9);
    border-color: rgba(183, 45, 45, 0.3);
    color: #ffa0a0;
  }

  .notification--warning {
    background: rgba(50, 40, 25, 0.9);
    border-color: rgba(183, 130, 45, 0.3);
    color: #ffe0a0;
  }

  .notification--info {
    background: rgba(25, 35, 50, 0.9);
    border-color: rgba(45, 115, 183, 0.3);
    color: #a0d0ff;
  }

  .notification__close {
    color: #aaa;
  }

  .notification__close:hover {
    background-color: rgba(255, 255, 255, 0.1);
  }
}
</style>
