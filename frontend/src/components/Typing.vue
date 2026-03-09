<script setup lang="ts">
import {onMounted, ref, watch} from 'vue';
import markdownit from 'markdown-it';

const mdt = markdownit({breaks: true, linkify: true, html: true});
const props = defineProps({
  msg: {
    type: String,
    default: ""
  },
  // 是否启用打字机效果
  enableTyping: {
    type: Boolean,
    default: false
  }
})

const displayText = ref('');
const isTyping = ref(false);

// 打字机效果函数
const typeText = (text: string) => {
  if (!text) {
    displayText.value = '';
    return;
  }

  // 如果不启用打字机效果，直接显示全部文字
  if (!props.enableTyping) {
    displayText.value = text;
    isTyping.value = false;
    return;
  }

  isTyping.value = true;
  let index = 0;
  const speed = 5; // 每个字符的间隔时间 (ms)

  const type = () => {
    if (index <= text.length) {
      displayText.value = text.substring(0, index);
      index++;
      setTimeout(type, speed);
    } else {
      isTyping.value = false;
    }
  };

  type();
};

// 监听 props.msg 和 props.enableTyping 变化
watch(() => [props.msg, props.enableTyping] as const, ([newMsg]) => {
  // 过滤掉 undefined 和 null 值，避免打印undefined
  if (newMsg === undefined || newMsg === null) {
    return;
  }

  typeText(newMsg);
}, {immediate: true});

onMounted(() => {
  if (props.msg) {
    typeText(props.msg);
  }
});

</script>

<template>
  <div
      v-html="mdt.render(displayText)"
      style="background: #eeeeee;padding: 12px 16px;border-radius: 12px"
      :class="{ typing: isTyping }"
  >
  </div>
</template>

<style scoped>
.typing {
  border-right: 2px solid #333;
  animation: blink 0.7s infinite;
}

@keyframes blink {
  0%, 100% {
    border-color: transparent;
  }
  50% {
    border-color: #333;
  }
}
</style>