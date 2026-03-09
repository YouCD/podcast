<script setup lang="ts">
import notificationService from "../components/Notification/notificationService";
</script>

<template>
  <div class="demo-container">
    <h1>Notification 组件演示</h1>
    <div class="button-group">
      <button @click="() => notificationService.success('这是一条成功的消息')">
        成功通知
      </button>
      <button @click="() => notificationService.error('这是一条错误的消息')">
        错误通知
      </button>
      <button @click="() => notificationService.warning('这是一条警告的消息')">
        警告通知
      </button>
      <button @click="() => notificationService.info('这是一条信息提示')">
        信息通知
      </button>

      <button
          @click="
          () =>
            notificationService.success('这条消息不会自动关闭', { duration: 0 })
        "
      >
        常驻通知
      </button>

      <button
          @click="
          () =>
            notificationService.info('这条消息会很快消失', { duration: 1000 })
        "
      >
        快速消失
      </button>
    </div>

    <div class="instructions">
      <h2>使用说明</h2>
      <p>
        1. Notification 组件已经全局注册，在任何组件中都可以通过
        notificationService 使用
      </p>
      <p>2. 支持四种类型的通知: success, error, warning, info</p>
      <p>3. 可以设置自动关闭时间，设置为 0 则不自动关闭</p>
      <p>4. 用户可以手动关闭通知</p>
    </div>
  </div>
</template>

<style scoped>
.demo-container {
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
}

.button-group {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin: 20px 0;
}

button {
  padding: 10px 15px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  background-color: #42b983;
  color: white;
  font-size: 14px;
  transition: background-color 0.3s;
}

button:hover {
  background-color: #359c6d;
}

.instructions {
  margin-top: 30px;
  padding: 20px;
  background-color: #f5f5f5;
  border-radius: 4px;
}

.instructions h2 {
  margin-top: 0;
}

.instructions p {
  margin: 10px 0;
}

@media (prefers-color-scheme: dark) {
  .instructions {
    background-color: #333;
    color: white;
  }
}
</style>
