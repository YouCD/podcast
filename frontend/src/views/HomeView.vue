<template>
  <main>
    <div class="header">
      <div class="menu-container">
        <a class="menu-button" @click="toggleMenu">☰</a>
        <div v-if="showMenu" class="menu-dropdown">
          <div class="menu-item" @click="handleMenuClick('login')">
            <span class="menu-icon">
              <span class="iconfont">&#xe8ef;</span>
            </span>
            <span class="login-button" v-if="!isValidAuth">登录</span>
            <a class="login-button" v-if="isValidAuth">{{
                tokenPayload?.name
              }}</a>
          </div>
          <div
              class="menu-item"
              v-if="isValidAuth"
              @click="handleMenuClick('ai')"
          >
            <span class="menu-icon">
              <span class="iconfont">&#xe69a;</span>
            </span>
            <span>AI</span>
          </div>
          <div
              class="menu-item"
              v-if="isValidAuth"
              @click="handleMenuClick('template')"
          >
            <span class="menu-icon">
              <span class="iconfont">&#xe631;</span>
            </span>
            <span>模板</span>
          </div>
          <div
              class="menu-item"
              v-if="isValidAuth"
              @click="handleMenuClick('prompt')"
          >
            <span class="menu-icon">
              <span class="iconfont">&#xe647;</span>
            </span>
            <span>Prompt</span>
          </div>

          <div class="menu-divider"></div>
          <div class="menu-item" @click="handleMenuClick('logout')">
            <span class="menu-icon">
              <span class="iconfont">&#xe8ef;</span>
            </span>
            <span>退出登录</span>
          </div>
        </div>
      </div>
    </div>
    <LoadingView2 v-if="showNodata"/>
    <div class="center-container" v-if="!hasRss && !showNodata">暂无数据</div>

    <!-- 登录模态框 -->
    <div v-if="showLoginModal" class="modal-overlay" @click="closeLoginModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>用户登录</h3>
          <button class="close-button" @click="closeLoginModal">×</button>
        </div>
        <form @submit.prevent="handleLoginSubmit" class="login-form">
          <div class="input-group">
            <label for="username">用户名</label>
            <input
                id="username"
                v-model="loginForm.name"
                type="text"
                placeholder="请输入用户名"
                required
            />
          </div>
          <div class="input-group">
            <label for="password">密码</label>
            <input
                id="password"
                v-model="loginForm.password"
                type="password"
                placeholder="请输入密码"
                required
            />
          </div>
          <button type="submit" class="submit-button">登录</button>
        </form>
      </div>
    </div>
    <template
        v-for="[categoryName, posts] in Array.from(rssMap.entries())"
        :key="categoryName"
    >
      <!-- 单个卡片 -->
      <MainView v-if="posts.length > 0">
        <!-- 左上角标题 -->
        <span class="title">{{ categoryName }}</span>
        <!-- 玻璃列表 -->
        <ul class="glass-list">
          <li
              v-for="p in posts"
              :key="p.id"
              @touchstart="onTouchStart(p, $event)"
              @touchmove="onTouchMove($event)"
              @touchend="onTouchEnd(p)"
              @click="openLink(p)"
              :style="
              clickedItem === p.id
                ? { background: 'rgba(255, 255, 255, 0.4)' }
                : {}
            "
          >
            <div class="li-header">
              <p class="li-title">{{ p.title }}</p>
            </div>
          </li>
        </ul>
      </MainView>
    </template>
    <!-- 悬浮按钮 -->
    <div
        class="floating_button"
        style="top: 50%"
        v-if="showFloatingButton"
        @click="handleFloatingButtonClick('read_list')"
    >
      已读
    </div>
    <div
        class="floating_button"
        style="top: calc(50% + 50px)"
        v-if="showFloatingButton"
        @click="handleFloatingButtonClick('llm_report')"
    >
      报告
    </div>
    <div
        class="floating_button"
        style="top: calc(50% + 100px)"
        v-if="showFloatingButton && !hasRss"
        @click="handleFetchNotReadClick()"
    >
      未读
    </div>

    <div v-if="status && !showNodata" class="status">
      <span v-if="status?.read_count > 0">
        <span class="iconfont">&#xe661;</span> {{ status?.read_count }}
      </span>
      &nbsp;&nbsp;&nbsp;
      <span v-if="status?.low_quality_count > 0">
        <span class="iconfont">&#xe743;</span>&nbsp;{{
          status?.low_quality_count
        }}
      </span>
    </div>
  </main>
</template>

<script setup lang="ts">
// 组件卸载时移除事件监听
import {onErrorCaptured, onMounted, onUnmounted, ref} from "vue";
import type {Rss, status} from "@/types/types.ts";
import router from "@/router";
import {fetchRssStatus, sendTimeStay} from "@/api/rss";
import LoadingView2 from "@/components/LoadingView2.vue";
import notificationService from "@/components/Notification/notificationService.ts";
import MainView from "@/components/MainView.vue";
import {storeToRefs} from "pinia";
import useCategoryStore from "@/stores/rss.ts";
import {userLogin} from "@/api/user.ts";
import {useUserStore} from "@/stores/user.ts"; // 导入登录API

const store = useCategoryStore();

const {rssMap, hasRss} = storeToRefs(store);
const userStore = useUserStore();
const {isValidAuth, tokenPayload} = storeToRefs(userStore);
const {setAuthToken} = userStore;

// 添加登录相关的响应式变量
const showLoginModal = ref(false);
const loginForm = ref({
  name: "",
  password: "",
});

const status = ref<status | undefined>(undefined);
onMounted(async () => {
  status.value = await fetchRssStatus();
  await store.loadAll();
  console.log("rssMap.value.size:", rssMap.value.size);
  showNodata.value = false;
  setTimeout(() => {
    showFloatingButton.value = true;
  }, 800);
  return;
});
// 添加这些变量用于长按删除功能
const touchStartTime = ref<number>(0);
const touchTimer = ref<NodeJS.Timeout | null>(null);
const longPressDuration = 2000; // 长按持续时间(毫秒)

// const playedPodcasts = ref<Set<number>>(new Set())
const showNodata = ref<boolean>(true);
const showFloatingButton = ref<boolean>(false);
const isLongPressed = ref(false);
const clickedItem = ref<number | null>(null);
const showMenu = ref(false);

const toggleMenu = () => {
  showMenu.value = !showMenu.value;
};

const handleMenuClick = (action: string) => {
  showMenu.value = false;
  switch (action) {
    case "login":
      if (!isValidAuth.value) {
        handleLoginClick();
      }
      // 这里可以添加设置页面的路由跳转
      break;
    case "ai":
      handleAIClick();
      // 这里可以添加个人资料页面的路由跳转
      break;
    case "template":
      router.push("/template");
      break;
    case "prompt":
      router.push("/prompt");
      break;
    case "logout":
      userStore.logout();
      router.push("/");
      notificationService.success("已退出登录", {duration: 3000});
      break;
  }
};

// 点击外部区域关闭菜单
const handleClickOutside = (event: MouseEvent) => {
  const menuElement = document.querySelector(".menu-dropdown");
  const buttonElement = document.querySelector(".menu-button");

  if (
      menuElement &&
      buttonElement &&
      !menuElement.contains(event.target as Node) &&
      !buttonElement.contains(event.target as Node)
  ) {
    showMenu.value = false;
  }
};

// 监听点击事件
onMounted(() => {
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside);
});

// ========== 错误捕获（新增） ==========
onErrorCaptured(() => false);

/* ===== 滑动修复 ===== */
const hasMoved = ref(false);

const openLink = (p: Rss) => {
  // 如果是长按操作，不执行跳转
  if (isLongPressed.value) {
    console.log("长按操作，不执行跳转");
    return;
  }

  // 立即添加背景样式
  clickedItem.value = p.id;

  // 延迟100ms跳转，给播放器足够的卸载时间
  setTimeout(() => {
    router.push({
      name: "llm_html",
      params: {id: p.id.toString()},
      query: {md5: p.md5},
    });
  }, 100);
};

// 更新登录按钮处理函数
const handleLoginClick = () => {
  showLoginModal.value = true;
};

// 处理登录提交
const handleLoginSubmit = async () => {
  try {
    const userInfo = {
      name: loginForm.value.name,
      password: loginForm.value.password,
    };

    const tokenInfo = await userLogin(userInfo);

    if (tokenInfo.success && tokenInfo.token) {
      // 将token保存到本地存储
      notificationService.success("登录成功", {duration: 3000});
      showLoginModal.value = false;
      // 清空表单
      loginForm.value = {name: "", password: ""};
      setAuthToken(tokenInfo.token);
      console.log("tokenInfo:", tokenInfo);
      console.log("isValidAuth:", isValidAuth);
    } else {
      notificationService.error(tokenInfo.message || "登录失败", {
        duration: 3000,
      });
    }
  } catch (error) {
    console.error("登录错误:", error);
    notificationService.error("登录失败，请重试", {duration: 3000});
  }
};

// 关闭登录模态框
const closeLoginModal = () => {
  showLoginModal.value = false;
  loginForm.value = {name: "", password: ""};
};

// 悬浮按钮点击处理函数
const handleFloatingButtonClick = (type: string) => {
  if (type === "read_list") {
    // 延迟100ms跳转，给播放器足够的卸载时间
    setTimeout(() => {
      router.push({
        name: "read_list",
      });
    }, 100);
  }
  if (type === "llm_report") {
    // 延迟100ms跳转，给播放器足够的卸载时间
    setTimeout(() => {
      router.push({
        name: "llm_report",
      });
    }, 100);
  }
};

// 添加长按删除相关的方法
const onTouchStart = (post: Rss, e: TouchEvent) => {
  hasMoved.value = false;
  isLongPressed.value = false;
  lastY = e.touches[0]!.clientY;
  touchStartTime.value = Date.now();

  if (touchTimer.value) clearTimeout(touchTimer.value);

  /* 关键：2 秒内必须“一次都没动”才算长按 */
  touchTimer.value = setTimeout(() => {
    if (!hasMoved.value) {
      // 全程没移动
      isLongPressed.value = true;
      deletePost(post);
    }
  }, longPressDuration);
};
const onTouchMove = (event: TouchEvent) => {
  const y = event.touches[0]!.clientY;
  cancelAnimationFrame(rafId);
  rafId = requestAnimationFrame(() => {
    if (Math.abs(y - lastY) > 6) {
      // 6 px 以内算手指抖动，忽略
      hasMoved.value = true; // 标记“已移动”→ 长按失效
      onScroll(); // 滚动时禁用点击
    }
  });
};

const onTouchEnd = (post: Rss) => {
  cancelAnimationFrame(rafId);
  if (touchTimer.value) clearTimeout(touchTimer.value);

  const duration = Date.now() - touchStartTime.value;

  // 只有“未移动 + 非长按 + 短触”才跳转
  if (!hasMoved.value && !isLongPressed.value && duration < 300) {
    openLink(post);
  }

  setTimeout(() => {
    isLongPressed.value = false;
    hasMoved.value = false;
  }, 100);
};

/* --------------- 新增 --------------- */
let rafId = 0; // RAF 句柄
let lastY = 0; // 上次 Y 坐标
const scrollingClass = "scrolling"; // CSS 类名
const glassListRef = ref<HTMLElement | null>(null); // 需要绑定到 ul 上

/* 滚动状态检测：滚动开始 / 结束 */
function onScroll() {
  const el = glassListRef.value;
  if (!el) return;
  el.classList.add(scrollingClass); // 开始滚动 → 禁用点击
  cancelAnimationFrame(rafId);
  rafId = requestAnimationFrame(() => {
    el.classList.remove(scrollingClass); // 滚动停止 1 帧后恢复点击
  });
}

// 实际删除文章的方法
const deletePost = async (post: Rss) => {
  // 从对应分类中移除文章
  store.removeRssItem(post.categories, post.id);

  // 同时也要从已播放的播客集合中删除
  notificationService.info("已添加到已读列表", {duration: 3000});
  await sendTimeStay({md5: post.md5 as string, time_stay: 1});
};

const handleFetchNotReadClick = async () => {
  await store.notReadHandler();
};

const handleAIClick = () => {
  router.push({
    name: "chat",
  });
};
</script>

<style scoped>
@import "../assets/floating_button.css";
@import "../assets/icon.css";

/* -------------- 页面背景 -------------- */
main {
  min-height: 100vh;
  padding: 0 0 2rem 0; /* 为固定在底部的播放器留出空间 */
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  flex-direction: column;
  width: 100%;
  gap: 1.5rem;
}

/* 页头 */
.header {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding: 1rem;
  //position: absolute;
  top: 0;
  right: 0;
  left: 0;
  z-index: 10;
  width: 100%;
  height: 50px;
  //background: black;
}

/* 左上角 title */
.title {
  position: absolute;
  top: 0;
  left: 12px;
  transform: translateY(-50%);
  background: rgba(255, 255, 255, 0.9);
  color: #5b21b6;
  font-size: 0.85rem;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 999px;
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
  max-width: 140px;
  -webkit-touch-callout: none; /* 禁止 iOS 弹出菜单 */
  user-select: none;
}

/* -------------- 玻璃列表 -------------- */
.glass-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 260px; /* 控制卡片高度 */
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.4) transparent;
}

.glass-list::-webkit-scrollbar {
  width: 6px;
}

.glass-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.4);
  border-radius: 3px;
}

.glass-list li {
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  transition: background 0.25s,
  transform 0.25s;
}

.glass-list li:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: translateY(-2px);
}

.glass-list li.empty {
  text-align: center;
  opacity: 0.7;
  cursor: default;
}

/* 单行标题 + 日期来源 */
.li-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  margin-top: 2px;
}

.li-title {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  margin: 0;
  font-size: 1rem;
  opacity: 0.85;
  line-height: 1.4;
  padding: 0 0.5rem 0 0.5rem;
  -webkit-touch-callout: none; /* 禁止 iOS 弹出菜单 */
  user-select: none;
}

.li-meta {
  font-size: 0.75rem;
  opacity: 0.8;
  white-space: nowrap;
}

/* 加载动画样式 */
.center-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 80vh;
  color: white;
}

/* 菜单 */
.menu-container {
  position: relative;
  display: inline-block;
}

.menu-button {
  font-size: 20px;
  color: white;
  cursor: pointer;
  //padding: 8px;
  border-radius: 4px;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.menu-button:hover {
  background: linear-gradient(
      135deg,
      rgba(255, 255, 255, 0.3) 0%,
      rgba(255, 255, 255, 0.2) 100%
  );
  transform: scale(1.1);
}

.menu-button:active {
  transform: scale(0.95);
}

.menu-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  background: linear-gradient(
      135deg,
      rgba(102, 126, 234, 0.95) 0%,
      rgba(118, 75, 162, 0.95) 100%
  );
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  min-width: 180px;
  z-index: 1000;
  animation: slideDown 0.2s ease-out;
  border: 1px solid rgba(255, 255, 255, 0.2);
  overflow: hidden;
  backdrop-filter: blur(15px);
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.menu-item {
  padding: 12px 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: all 0.2s ease;
  font-size: 14px;
  color: white;
  font-weight: 500;
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.2);
}

.menu-item:active {
  background: rgba(255, 255, 255, 0.3);
}

.menu-icon {
  font-size: 16px;
  width: 20px;
  text-align: center;
}

.menu-divider {
  height: 1px;
  background: linear-gradient(
      90deg,
      transparent,
      rgba(255, 255, 255, 0.4),
      transparent
  );
  margin: 4px 16px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .menu-dropdown {
    min-width: 160px;
    right: -10px;
  }

  .menu-item {
    padding: 10px 14px;
    font-size: 13px;
  }
}

/* 登入按钮 */
.login-button {
  //background: rgba(255, 255, 255, 0.2);
  //color: white;
  //border: 1px solid rgba(255, 255, 255, 0.3);
  //border-radius: 999px;
  //padding: 0.5rem 1rem;
  //font-size: 0.9rem;
  cursor: pointer;
  backdrop-filter: blur(10px);
  transition: all 0.3s ease;
  z-index: 20;
}

/*
.login-button:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: scale(1.05);
}

.login-button:active {
  transform: scale(0.98);
}
 */
/* 登录模态框 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  width: 90%;
  max-width: 400px;
  overflow: hidden;
  animation: modalAppear 0.3s ease-out;
}

@keyframes modalAppear {
  from {
    opacity: 0;
    transform: scale(0.8);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem 1.5rem 0 1.5rem;
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  margin: 0;
  color: #333;
}

.close-button {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #999;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-button:hover {
  color: #333;
}

.login-form {
  padding: 1.5rem;
}

.input-group {
  margin-bottom: 1rem;
}

.input-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: #555;
  font-weight: 500;
}

.input-group input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 1rem;
  box-sizing: border-box;
}

.input-group input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.2);
}

.submit-button {
  width: 100%;
  padding: 0.75rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.2s;
}

.submit-button:hover {
  opacity: 0.9;
}

.submit-button:active {
  opacity: 0.8;
}

/* 1. 把滚动权完全交给浏览器 */
.glass-list {
  touch-action: pan-y; /* 关键：只保留纵向滚动 */
  -webkit-overflow-scrolling: touch; /* iOS 惯性 */
  overflow-y: auto;
}

/* 2. 滚动期间禁止所有交互（JS 会动态加这个 class） */
.glass-list.scrolling {
  pointer-events: none;
}

.status {
  display: flex;
  justify-content: center;
  align-items: center;
  color: white;
}
</style>
