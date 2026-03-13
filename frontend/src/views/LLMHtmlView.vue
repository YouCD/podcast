<script setup lang="ts">
import { useRoute } from "vue-router";
import {
  computed,
  createVNode,
  nextTick,
  onBeforeMount,
  onMounted,
  onUnmounted,
  ref,
  render,
  watch,
} from "vue";
import { fetchLlmHtml, sendTimeStay } from "@/api/rss.ts";
import type { llmResult, NextLlmHtml } from "@/types/types.ts";
import moment from "moment";
import router from "@/router";
import useCategoryStore from "@/stores/rss.ts";
import ShareButton from "@/components/ShareButton.vue";
import Graph from "@/components/Graph.vue";

const store = useCategoryStore();
/* ---------- 基础数据 ---------- */
const route = useRoute();
const rss = ref<NextLlmHtml>();
const reportHTML = ref<HTMLDivElement>();

// 判断是否是PC端
const isPC = computed(() => {
  const userAgent = navigator.userAgent;
  const mobileAgents = [
    "Android",
    "iPhone",
    "iPad",
    "iPod",
    "BlackBerry",
    "Windows Phone",
  ];
  return !mobileAgents.some((agent) => userAgent.includes(agent));
});

const showPreviousButton = computed(() => {
  return !!rss.value?.previous;
});

const showNextButton = computed(() => {
  return isPC.value && !!rss.value?.next;
});

const parsedLlmResult = computed(() => {
  if (!rss.value?.current) return null;
  try {
    return JSON.parse(rss.value.current.llm_result) as llmResult;
  } catch (e) {
    console.error("Failed to parse llm_result:", e);
    return null;
  }
});

//  利用路由守卫 发送页面停留时间
router.beforeEach((to, from, next) => {
  // 执行权限验证或其他操作
  // console.log('进入路由：', to.path)
  // console.log('离开路由：', from.path)
  if (to.path === "/") {
    if (!rss.value) return;
    sendTimeStayHandlers(rss.value.current.md5, rss.value.current.id);
    store.removeRssItem(rss.value.current.categories, rss.value.current.id);
  }

  next();
});

/* 加载上一条 */
const doLoadPrevious = async () => {
  if (!rss.value?.previous) return;

  await sendTimeStayHandlers(rss.value.current.md5, rss.value.current.id);
  store.removeRssItem(rss.value.current.categories, rss.value.current.id);

  try {
    // 获取上一条的前一条
    const previousData = await fetchLlmHtml(
      rss.value.previous.id,
      store.notRead,
    );
    if (previousData) {
      rss.value = {
        current: previousData.current,
        next: rss.value.current,
        previous: previousData.previous,
      };
      window.scrollTo({ top: 0, behavior: "instant" });
    }
  } catch (e) {
    console.error("加载上一条失败:", e);
  }
};

/* 加载下一条 */
const doLoadNext = async () => {
  if (!rss.value?.next) return;

  await sendTimeStayHandlers(rss.value.current.md5, rss.value.current.id);
  store.removeRssItem(rss.value.current.categories, rss.value.current.id);

  try {
    // 获取下一条的下一条
    const nextData = await fetchLlmHtml(rss.value.next.id, store.notRead);
    if (nextData) {
      rss.value = {
        current: nextData.current,
        next: nextData.next,
        previous: rss.value.current,
      };
      window.scrollTo({ top: 0, behavior: "instant" });
    }
  } catch (e) {
    console.error("加载下一条失败:", e);
  }
};

// 键盘事件处理
const handleKeyDown = (e: KeyboardEvent) => {
  // 只在 PC 端生效
  if (!isPC.value) return;

  if (e.key === "ArrowUp") {
    e.preventDefault();
    if (rss.value?.previous) {
      doLoadPrevious();
    }
  } else if (e.key === "ArrowDown") {
    e.preventDefault();
    if (rss.value?.next) {
      doLoadNext();
    }
  }
};

//  发送时间
const sendTimeStayHandlers = async (md5: string, id: number) => {
  const endTime = moment().unix();
  const timeDiff = endTime - startTimeA.value;
  await sendTimeStay({
    md5: md5,
    time_stay: timeDiff,
  });
  startTimeA.value = moment().unix();
  console.log("已记录时间差:", timeDiff);
};

const startTimeA = ref(moment().unix());

/* 动态插入 Graph 组件到 container-box 下方 */
const insertGraphComponent = () => {
  nextTick(() => {
    const containerBox = document.querySelector(".container-box");
    if (containerBox && rss.value?.current?.dgraph) {
      // 创建一个新的 div 来容纳 Graph 组件
      const graphContainer = document.createElement("div");
      graphContainer.id = "dynamic-graph-container";
      graphContainer.style.marginTop = "20px";

      // 插入到 container-box 后面
      containerBox.parentNode?.insertBefore(
        graphContainer,
        containerBox.nextSibling,
      );
      if (!rss.value.current.dgraph) {
        return;
      }
      // 创建并挂载 Graph 组件
      // console.log('rss.value.current.dgraph', rss.value.current.dgraph.nodes)
      const graphVNode = createVNode(Graph, {
        dgraphResponse: rss.value.current.dgraph,
        style: {
          height: "200px",
          width: "100%",
          borderRadius: "12px",
        },
        showThumbnail: false,
      });
      render(graphVNode, graphContainer);
    }
  });
};

/* 监听 rss 数据变化，动态插入 Graph 组件 */
watch(
  () => rss.value?.current?.llm_html,
  () => {
    if (rss.value?.current?.llm_html) {
      // 使用 setTimeout 确保 DOM 已经更新
      setTimeout(() => {
        insertGraphComponent();
      }, 100);
    }
  },
);

/* ---------- 上滑加载逻辑（必须持续触摸 3s）---------- */
const LOAD_TIP_SHOW = ref(false);
let startY = 0;
let reachBottomFlag = false; // 是否已滑到底部
let touching = false; // 手指是否仍在屏幕上
let loadingTimer: number | null = null;
let isLoading = false;
const tips = ref("持续上滑 1 秒加载下一条");

/* 真正加载下一条 */
const doLoadNextSwipe = async () => {
  if (isLoading || !rss.value?.next) {
    tips.value = "已经没有更多内容了";
    return;
  }
  isLoading = true;
  /* 立刻消失，体验更干净 */
  LOAD_TIP_SHOW.value = false;
  if (hideTipTimer) clearTimeout(hideTipTimer);

  await sendTimeStayHandlers(rss.value.current.md5, rss.value.current.id);
  store.removeRssItem(rss.value.current.categories, rss.value.current.id);
  try {
    const nextData = await fetchLlmHtml(rss.value.next.id, store.notRead);
    if (nextData) {
      rss.value = {
        current: nextData.current,
        next: nextData.next,
        previous: rss.value.current,
      };
    }
    window.scrollTo({ top: 0, behavior: "instant" });
  } catch (e) {
    console.error("加载下一条失败:", e);
  } finally {
    isLoading = false;
  }
};

let hideTipTimer: number | null = null;
/* 重置状态 */
/* 立即重置触摸状态，但提示框延迟消失 */
const resetTouch = () => {
  touching = false;
  reachBottomFlag = false;
  if (loadingTimer) {
    clearTimeout(loadingTimer);
    loadingTimer = null;
  }
  /* 如果之前已经有一个待隐藏的定时器，先清掉 */
  if (hideTipTimer) {
    clearTimeout(hideTipTimer);
  }
  /* 1.8 s 后再隐藏提示 */
  hideTipTimer = window.setTimeout(() => {
    LOAD_TIP_SHOW.value = false;
  }, 1800);
};

/* 触摸开始 */
const onTouchStart = (e: TouchEvent) => {
  if (isLoading) return;
  startY = e.touches[0]!.clientY;
  touching = true;
  const scrollH = document.documentElement.scrollHeight;
  const scrollT = window.scrollY;
  const clientH = window.innerHeight;
  reachBottomFlag = scrollT + clientH >= scrollH - 2;
};

/* 触摸移动 */
const onTouchMove = (e: TouchEvent) => {
  if (!touching || isLoading) return;
  const deltaY = e.touches[0]!.clientY - startY;
  /* 只有上滑且已到底部才处理 */
  if (deltaY < 0 && reachBottomFlag) {
    if (!LOAD_TIP_SHOW.value) LOAD_TIP_SHOW.value = true;
    /* 如果还没开始倒计时，就启动 3s 定时器 */
    if (!loadingTimer) {
      loadingTimer = window.setTimeout(() => {
        /* 3s 后手指仍在屏幕才加载 */
        if (touching) {
          resetTouch();
          doLoadNextSwipe();
        }
      }, 200);
    }
  } else {
    /* 非底部上滑，直接取消 */
    resetTouch();
  }
};

/* 触摸结束/取消 */
const onTouchEnd = () => {
  resetTouch();
};

/* 生命周期 */
onMounted(() => {
  document.addEventListener("touchstart", onTouchStart, { passive: true });
  document.addEventListener("touchmove", onTouchMove, { passive: true });
  document.addEventListener("touchend", onTouchEnd);
  document.addEventListener("touchcancel", onTouchEnd);
  window.addEventListener("keydown", handleKeyDown);
  window.scroll(0, 0);
});

onUnmounted(() => {
  document.removeEventListener("touchstart", onTouchStart);
  document.removeEventListener("touchmove", onTouchMove);
  document.removeEventListener("touchend", onTouchEnd);
  document.removeEventListener("touchcancel", onTouchEnd);
  window.removeEventListener("keydown", handleKeyDown);
  if (loadingTimer) clearTimeout(loadingTimer);
  if (hideTipTimer) clearTimeout(hideTipTimer);
  resetTouch();
});

/* 初始数据 */
onBeforeMount(async () => {
  try {
    const data = await fetchLlmHtml(Number(route.params.id), store.notRead);
    if (data) {
      rss.value = {
        current: data.current,
        next: data.next,
        previous: null,
      };
    }
  } catch (e) {
    console.error("获取失败:", e);
  }
});
</script>

<template>
  <Transition name="slide-fade" mode="out-in">
    <div>
      <!-- 悬浮按钮 -->
      <div class="floating_button" style="top: 60%">
        <ShareButton :file_name="rss?.current.title" />
      </div>
      <div
        class="web-con"
        v-if="rss && rss.current"
        v-html="rss.current.llm_html"
      ></div>
    </div>
  </Transition>

  <!-- PC端导航按钮 -->
  <div class="navigation-buttons" :style="{ top: isPC ? '50%' : '56%' }">
    <button
      v-if="showPreviousButton"
      class="nav-button"
      @click="doLoadPrevious"
      title="上一条 (↑)"
    >
      <span class="iconfont">&#xe63c;</span>
    </button>
    <button
      v-if="showNextButton"
      class="nav-button"
      @click="doLoadNext"
      title="下一条 (↓)"
    >
      <span class="iconfont">&#xe63d;</span>
    </button>
  </div>

  <!-- 底部悬停提示 -->
  <Transition name="fade">
    <div v-if="LOAD_TIP_SHOW" class="load-tip">
      <span>{{ tips }}</span>
    </div>
  </Transition>
</template>

<style scoped>
@import "../assets/icon.css";
@import "../assets/floating_button.css";

.web-con {
  padding: 10px;
}

.load-tip {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.65);
  color: #fff;
  padding: 10px 20px;
  border-radius: 20px;
  font-size: 14px;
  z-index: 999;
}

/* 内容切换动画 start */
.slide-fade-enter-active {
  transition: all 0.25s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.25s ease-in;
}

.slide-fade-enter-from {
  transform: translateY(50px);
  opacity: 0;
}

.slide-fade-leave-to {
  transform: translateY(-50px);
  opacity: 0;
}

/* 内容切换动画 end */

/* 导航按钮样式 */
.navigation-buttons {
  position: fixed;
  right: 10px;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  gap: 10px;
  z-index: 1000;
}

.nav-button {
  width: 35px;
  height: 35px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  font-size: 18px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
  /* 防止点击时出现正方形效果 */
  -webkit-tap-highlight-color: transparent;
  outline: none;
  user-select: none;
}

.nav-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.6);
}

/* 关键修复：确保active状态保持圆形 */
.nav-button:active {
  border-radius: 50%;
  transform: translateY(0);
}

.nav-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

/* 内容切换动画 start */
.slide-fade-enter-active {
  transition: all 0.25s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.25s ease-in;
}

.slide-fade-enter-from {
  transform: translateY(50px);
  opacity: 0;
}

.slide-fade-leave-to {
  transform: translateY(-50px);
  opacity: 0;
}

/* 内容切换动画 end */
</style>
