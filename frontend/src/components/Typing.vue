<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue";
import markdownit from "markdown-it";

const mdt = markdownit({ breaks: true, linkify: true, html: true });
const props = defineProps({
  msg: {
    type: String,
    default: "",
  },
  // 是否启用打字机效果
  enableTyping: {
    type: Boolean,
    default: false,
  },
});

const displayText = ref("");
const isTyping = ref(false);
let typingTimer: ReturnType<typeof setTimeout> | null = null;
let lastMsg = ""; // 记录上一次的消息内容

// 清除打字机定时器
const clearTypingTimer = () => {
  if (typingTimer) {
    clearTimeout(typingTimer);
    typingTimer = null;
  }
};

// 打字机效果函数
const typeText = (text: string) => {
  // 清除之前的定时器
  clearTypingTimer();

  if (!text) {
    displayText.value = "";
    lastMsg = "";
    return;
  }

  // 如果不启用打字机效果，直接显示全部文字
  if (!props.enableTyping) {
    displayText.value = text;
    isTyping.value = false;
    lastMsg = text;
    return;
  }

  // 检查是否是追加内容（新内容以旧内容开头）
  const isAppend = text.startsWith(lastMsg) && lastMsg.length > 0;

  // 如果是追加内容，从当前位置继续打字
  let startIndex = isAppend ? displayText.value.length : 0;

  // 如果不是追加（内容完全不同），重置显示
  if (!isAppend) {
    displayText.value = "";
  }

  isTyping.value = true;
  const speed = 5; // 每个字符的间隔时间 (ms)

  const type = () => {
    if (startIndex <= text.length) {
      displayText.value = text.substring(0, startIndex);
      startIndex++;
      typingTimer = setTimeout(type, speed);
    } else {
      isTyping.value = false;
      typingTimer = null;
      lastMsg = text;
    }
  };

  type();
};

// 监听 props.msg 和 props.enableTyping 变化
watch(
  () => [props.msg, props.enableTyping] as const,
  ([newMsg, newEnableTyping]) => {
    // 过滤掉 undefined 和 null 值，避免打印undefined
    if (newMsg === undefined || newMsg === null) {
      return;
    }

    // 如果 enableTyping 从 true 变为 false，立即显示全部内容并停止打字机效果
    if (!newEnableTyping && isTyping.value) {
      clearTypingTimer();
      displayText.value = newMsg;
      isTyping.value = false;
      lastMsg = newMsg;
      return;
    }

    typeText(newMsg);
  },
  { immediate: true },
);

onMounted(() => {
  if (props.msg) {
    typeText(props.msg);
  }
});

onUnmounted(() => {
  clearTypingTimer();
});
</script>

<template>
  <div
    v-html="mdt.render(displayText)"
    style="background: #eeeeee; padding: 12px 16px; border-radius: 12px"
    :class="{ typing: isTyping }"
  ></div>
</template>

<style scoped>
.typing {
  border-right: 2px solid #333;
  animation: blink 0.7s infinite;
}

@keyframes blink {
  0%,
  100% {
    border-color: transparent;
  }
  50% {
    border-color: #333;
  }
}
</style>
