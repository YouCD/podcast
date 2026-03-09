<template>
  <main>
    <!-- 移动端遮罩层 -->
    <div
        v-if="isMobile && isSidebarOpen"
        class="sidebar-overlay"
        @click="toggleSidebar(false)"
    ></div>
    <div>
      <McLayout
          class="leftClass"
          :class="{
          'sidebar-hidden': isMobile && !isSidebarOpen,
          'sidebar-open': isMobile && isSidebarOpen,
        }"
      >
        <div class="sidebar-header">
          <img src="/assets/favicon.png" alt="PodCast" width="40" height="40">
          <!-- 移动端关闭按钮 -->
          <button
              v-if="isMobile"
              class="sidebar-close-btn"
              @click="toggleSidebar(false)"
          >
            <span class="iconfont" style="font-size: 25px;">&#xe608;</span>
          </button>
        </div>
        <!--        <hr style="color: #d9d9d9" />-->
        <div class="session-list">
          <!-- 今天 -->
          <div v-if="groupedSessions.today.length > 0" class="session-group">
            <div class="session-group-title">今天</div>
            <div v-for="item in groupedSessions.today" :key="item.session_id">
              <button class="custom-button" @click="fetchMsg(item.session_id)">
                {{ item.title }}
              </button>
            </div>
          </div>

          <!-- 7 天内 -->
          <div v-if="groupedSessions.sevenDays.length > 0" class="session-group">
            <div class="session-group-title">7 天内</div>
            <div v-for="item in groupedSessions.sevenDays" :key="item.session_id">
              <button class="custom-button" @click="fetchMsg(item.session_id)">
                {{ item.title }}
              </button>
            </div>
          </div>

          <!-- 30 天内 -->
          <div v-if="groupedSessions.thirtyDays.length > 0" class="session-group">
            <div class="session-group-title">30 天内</div>
            <div v-for="item in groupedSessions.thirtyDays" :key="item.session_id">
              <button class="custom-button" @click="fetchMsg(item.session_id)">
                {{ item.title }}
              </button>
            </div>
          </div>

          <!-- 更早 -->
          <div v-if="groupedSessions.older.length > 0" class="session-group">
            <div class="session-group-title">更早</div>
            <div v-for="item in groupedSessions.older" :key="item.session_id">
              <button class="custom-button" @click="fetchMsg(item.session_id)">
                {{ item.title }}
              </button>
            </div>
          </div>
        </div>
      </McLayout>
      <McLayout
          class="rightClass"
          :class="{ 'full-width': isMobile && !isSidebarOpen }"
      >
        <McHeader :title="'PodCast'" id="mc_header" :logoImg="'assets/favicon.png'">
          <template #operationArea>
            <div class="operations">
              <!-- 移动端展开按钮 -->
              <button
                  v-if="isMobile"
                  class="sidebar-toggle-btn"
                  @click="toggleSidebar(true)"
                  title="展开会话列表"
              >
                ☰
              </button>
            </div>
          </template>
        </McHeader>

        <McLayoutContent
            v-if="startPage"
            style="
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            gap: 12px;
          "
        >
          <McIntroduction
              :logoImg="'assets/favicon.png'"
              :title="'PodCast'"
              :subTitle="'Hi，欢迎使用 PodCast'"
              :description="description"
              :logoWidth="64"
              :logoHeight="64"
          ></McIntroduction>
        </McLayoutContent>

        <McLayoutContent class="content-container" ref="conversationRef" v-else>
          <template v-if="sessionData" v-for="(msg, idx) in  sessionData.messages" :key="idx">
            <McBubble
                v-if="msg.role === 'user'"
                :content="msg.content"
                :align="'right'"
            >

            </McBubble>
            <McBubble
                v-else
                :loading="msg.loading"
            >
              <!-- 思考过程切换按钮 -->
              <div
                  v-if="msg.reasoning_content!=''"
                  class="think-toggle-btn"
                  @click="()=>{msg.showReasoningContent=!msg.showReasoningContent}"
              >
                <i class="icon-point"></i>
                <span>{{ msg.showReasoningContent ? '隐藏思考过程' : '显示思考过程' }}</span>
                <i :class="msg.showReasoningContent   ? 'icon-chevron-up' : 'icon-chevron-down'"></i>
              </div>
              <Typing v-if="msg.showReasoningContent && msg.reasoning_content!=''" :enableTyping="msg.reasoning_typing"
                      :msg="msg.reasoning_content"></Typing>
              <McMarkdownCard :content="msg.content" :typing="msg.typing"/>
            </McBubble>
          </template>
        </McLayoutContent>

        <div
            class="shortcut"
            style="display: flex; align-items: center; gap: 8px"
        >
          <Button
              style="margin-left: auto"
              icon="add"
              shape="circle"
              title="新建对话"
              size="md"
              @click="newConversation"
          />
        </div>

        <McLayoutSender>
          <McInput
              :value="inputValue"
              :maxLength="2000"
              @submit="onSubmit"
          >
            <template #extra>
              <div class="input-foot-wrapper">
                <div class="input-foot-left">
                  <span class="input-foot-maxlength"
                  >{{ inputValue.length }}/2000</span
                  >
                </div>
                <div class="input-foot-right">
                  <Button
                      style="padding:0 12px;"
                      icon="op-clearup"
                      shape="round"
                      :disabled="!inputValue"
                      @click="inputValue = ''"
                  >
                    <span :class="{'demo-button-content': inputValue.length > 0}">清空输入</span>
                  </Button>
                </div>
              </div>
            </template>
          </McInput>
        </McLayoutSender>
      </McLayout>
    </div>
  </main>
</template>
<script lang="ts">
import {defineComponent} from "vue";
import notificationService from "@/components/Notification/notificationService.ts";

export default defineComponent({
  async beforeRouteEnter(to, from, next) {
    // 在 beforeRouteEnter 内部使用 useUserStore
    const userStore = useUserStore();
    if (!userStore.isValidAuth) {
      notificationService.error("请先登录后再使用此功能");
      console.warn("认证失败，令牌可能已过期");
      next("/"); // 使用 next 跳转，而不是 router.push
      return;
    }
    // 需要处理的逻辑
    next();
  },
});


</script>
<script setup lang="ts">
import "@devui-design/icons/icomoon/devui-icon.css";
import {
  McBubble,
  McHeader,
  McInput,
  McIntroduction,
  McLayout,
  McLayoutContent,
  McLayoutSender,
  McMarkdownCard,
} from "@matechat/core";

import {fetchEventSource} from "@microsoft/fetch-event-source";
import {computed, nextTick, onBeforeMount, onMounted, onUnmounted, ref} from "vue";
import {v4 as uuidv4} from "uuid";
import {useUserStore} from "@/stores/user.ts";
import type {sessionItem, messageInfo, msgRequest} from "@/types/types.ts";
import {useChatRecordsStore} from "@/stores/chat.ts";
import {storeToRefs} from "pinia";
import {Button} from "vue-devui/button";
import "vue-devui/button/style.css";
import {changeTitleHandler} from "@/api/chat.ts";
import Typing from "@/components/Typing.vue";

const currentUserId = ref<string>("9527");
const userStore = useUserStore();
const {tokenPayload,} = storeToRefs(userStore);

const chatRecordsStore = useChatRecordsStore();
const {
  createNewSessionAction,
  fetchSessionList,
  fetchSessionDataBySessionId,
  filterSession,
  sendMessageAction,
  setMsg,
  changeTitleHandlerAction,
  SetReasoningTyping,
} = chatRecordsStore;
const {sessionList, sessionData} = storeToRefs(chatRecordsStore);

// 按时间分组的会话列表
const groupedSessions = computed(() => {
  const today = new Date();
  const sevenDaysAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000);
  const thirtyDaysAgo = new Date(today.getTime() - 30 * 24 * 60 * 60 * 1000);

  const groups = {
    today: [] as sessionItem[],
    sevenDays: [] as sessionItem[],
    thirtyDays: [] as sessionItem[],
    older: [] as sessionItem[]
  };

  sessionList.value.forEach((item) => {
    const updateDate = item.updated_at ? new Date(item.updated_at) : new Date();

    if (updateDate >= new Date(today.setHours(0, 0, 0, 0))) {
      groups.today.push(item);
    } else if (updateDate >= sevenDaysAgo) {
      groups.sevenDays.push(item);
    } else if (updateDate >= thirtyDaysAgo) {
      groups.thirtyDays.push(item);
    } else {
      groups.older.push(item);
    }
  });

  return groups;
});

const currentSession = ref<sessionItem | null>(null);
const startPage = ref<boolean>(true);
const inputValue = ref("");
const conversationRef = ref<any>(null);
const response = ref("");

// ===== 新增：移动端侧边栏控制 =====
const isSidebarOpen = ref(false);
const isMobile = ref(false);
const MOBILE_BREAKPOINT = 768; // 移动端断点

// 思考过程控制
const thinkContent = ref(""); // 累积思考内容

// 检测是否为移动端
const checkMobile = () => {
  isMobile.value = window.innerWidth <= MOBILE_BREAKPOINT;
  if (!isMobile.value) {
    isSidebarOpen.value = true; // 桌面端默认展开
  } else {
    isSidebarOpen.value = false; // 移动端默认收起
  }
};

// 控制侧边栏开关并管理滚动
const toggleSidebar = (open?: boolean) => {
  const newState = open !== undefined ? open : !isSidebarOpen.value;
  isSidebarOpen.value = newState;

  // 控制body滚动
  if (newState && isMobile.value) {
    // 打开侧边栏时禁止背景滚动
    document.body.style.overflow = 'hidden';
    document.body.style.touchAction = 'none';
  } else {
    // 关闭侧边栏时恢复滚动
    document.body.style.overflow = '';
    document.body.style.touchAction = '';
  }
};


onMounted(async () => {
  currentUserId.value = String(tokenPayload.value?.id || "9527");
  await fetchSessionList();
  // 初始化移动端检测
  checkMobile();
  window.addEventListener("resize", checkMobile);
});

const fetchMsg = async (sessionId: string) => {
  await fetchSessionDataBySessionId(sessionId);
  startPage.value = false;
  let result = await filterSession(sessionId);
  currentSession.value = result;
  // 移动端选择会话后自动关闭侧边栏
  if (isMobile.value) {
    toggleSidebar(false);
  }
  await scrollToBottom();
};


const description = [
  "PodCast一个使用 Go 语言开发的 RSS 内容抓取与分析平台",
  "主要用于抓取 RSS 信息、通过 AI 对内容进行智能分析和处理，并生成结构化的数据展示",
];

const newConversation = async () => {
  startPage.value = false;

  currentSession.value = {
    session_id: uuidv4(),
    title: "新会话",
    user_id: Number(currentUserId.value),
  };

  await createNewSessionAction(currentSession.value!.session_id);

  // 移动端新建对话后关闭侧边栏
  if (isMobile.value) {
    toggleSidebar(false);
  }
};

const onSubmit = async (evt: string) => {
  if (!evt) return;
  inputValue.value = "";
  // 创建会话
  await newConversation()

  let uuidStr = uuidv4()

  await setMsg(currentSession.value!.session_id, "user", uuidStr, evt)

  // 创建会话
  if (sessionData.value) {
    if (sessionData.value!.messages.length === 1) {
      await createNewSessionAction(currentSession.value!.session_id);
    }
  }

  let u=uuidv4()
  await setMsg(currentSession.value!.session_id, "assistant", u, undefined, undefined, true)

  await getAIAnswer(evt,u);
};

const scrollToBottom = async () => {
  await nextTick();
  const comp = conversationRef.value;
  if (!comp) return;
  const rootEl = comp.$el as HTMLElement;
  const scrollEl = rootEl.querySelector(".content-container") || rootEl;
  scrollEl.scrollTop = scrollEl.scrollHeight;
};

const getAIAnswer = async (content: string,uuidStr :string) => {
  await scrollToBottom();
  response.value = "";
  const ctrl = new AbortController();
  const token = localStorage.getItem("authToken");
  const requestData = {
    content: content,
    session_id: currentSession.value!.session_id,
  };
  thinkContent.value = ""



  await fetchEventSource("/api/chat/stream", {
    method: "POST",
    openWhenHidden: true,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
      Authorization: `Bearer ${token}`,
    },
    signal: ctrl.signal,
    body: JSON.stringify(requestData),
    onmessage: (event: any) => {
      if (event.event === "think") {
        const data = JSON.parse(event.data);
        thinkContent.value += data.message;
        setMsg(currentSession.value!.session_id, "assistant", uuidStr, undefined, data.message)
      }

      if (event.event === "message") {
        try {
          const data = JSON.parse(event.data);
          if (data.message !== undefined) {
            setMsg(currentSession.value!.session_id, "assistant", uuidStr, data.message)
            response.value += data.message;
            if (data.data === '{"end":true}') {
              console.log("Ai END");
              // nextTick(() => {
              //   conversationRef.value?.scrollTo({
              //     top: conversationRef.value.scrollHeight,
              //     behavior: "smooth",
              //   });
              // });
              scrollToBottom();
              return;
            }
            scrollToBottom();
          }
        } catch (e) {
          console.error("SSE parse error", e);
        }
      }
    },
    onclose: async () => {
      console.log("SSE closed");
      await sendMessageAction({
        session_id: currentSession.value!.session_id,
        reasoning_content: thinkContent.value,
        content: response.value,
        role: "assistant",
        uuid: uuidStr,
      })
      SetReasoningTyping(uuidStr, false)

      if (currentSession.value!.title == "新会话") {
        await changeTitleHandlerAction()
        await fetchSessionList();
      }
    },
    onerror: (err: any) => {
      ctrl?.abort();
    },
  });
};
</script>

<style scoped>
@import "../assets/custom_button.css";

.think-toggle-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin-bottom: 8px;
  background: #f5f5f5;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  color: #666;
  transition: background 0.2s;

  &:hover {
    background: #e8e8e8;
  }

  i {
    font-size: 12px;
  }
}

.leftClass {
  width: 220px;
  margin: 5px;
  height: calc(100vh - 10px);
  padding: 0 8px;
  gap: 8px;
  background: #fff;
  border: 1px solid #ddd;
  border-radius: 16px;
  float: left;
  transition: transform 0.3s ease;
  position: relative;
}

.session-list {
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 4px;
}

.session-group {
  margin-bottom: 20px;
}

.session-group:last-child {
  margin-bottom: 8px;
}

.session-group-title {
  font-size: 12px;
  color: #999;
  margin-bottom: 10px;
  padding: 6px 10px;
  font-weight: 600;
  background: linear-gradient(135deg, #f5f7fa 0%, #e9ecef 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  position: relative;
  overflow: hidden;
}

.session-group-title::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
  border-radius: 0 4px 4px 0;
}

/* 为不同分组添加不同的图标 */
.session-group-title::after {
  font-size: 14px;
  margin-left: auto;
}

.session-group:nth-child(1) .session-group-title::after {
  content: '🌅'; /* 今天 */
}

.session-group:nth-child(2) .session-group-title::after {
  content: '📅'; /* 7 天内 */
}

.session-group:nth-child(3) .session-group-title::after {
  content: '🗓️'; /* 30 天内 */
}

.session-group:nth-child(4) .session-group-title::after {
  content: '📁'; /* 更早 */
}

.session-group .custom-button {
  margin: 4px 0;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 10px;
  background: #fafafa;
  border: 1px solid transparent;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.session-group .custom-button:hover {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  transform: translateX(4px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
  border-color: transparent;
}

.session-group .custom-button:active {
  transform: translateX(2px) scale(0.98);
}

/* 滚动条美化 */
.session-list::-webkit-scrollbar {
  width: 6px;
}

.session-list::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.session-list::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, #c1c4ca 0%, #a8aab0 100%);
  border-radius: 3px;
}

.session-list::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, #a8aab0 0%, #8f9297 100%);
}

.rightClass {
  width: calc(100% - 10px - 220px);
  margin: 5px auto;
  height: calc(100vh - 10px);
  padding: 20px;
  gap: 8px;
  background: #fff;
  border: 1px solid #ddd;
  border-radius: 16px;
  float: right;
  transition: width 0.3s ease;
}

.rightClass :deep(.mc-header-logo) {
  width: 32px;
  height: 32px;
}

/* 移动端样式 */
@media screen and (max-width: 768px) {
  .leftClass {
    position: fixed;
    left: 0;
    top: 0;
    z-index: 1000;
    margin: 0;
    border-radius: 0;
    box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
  }

  .leftClass.sidebar-hidden {
    transform: translateX(-100%);
  }

  .leftClass.sidebar-open {
    transform: translateX(0);
  }

  .rightClass {
    width: calc(100% - 10px);
    margin: 0 5px;
    float: none;
    height: 100vh;
  }

  .leftClass {
    height: 100vh;
  }

  .rightClass.full-width {
    width: calc(100% - 10px);
  }

  .sidebar-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 999;
  }

  .sidebar-toggle-btn {
    background: none;
    border: none;
    font-size: 20px;
    cursor: pointer;
    padding: 8px;
    margin-right: 8px;
    color: #333;
  }


  .sidebar-header {
    color: black;
  }

  #mc_header :deep(.mc-header-title) {
    color: black;
  }

  :deep(.filled) {
    color: black;
  }
}

.sidebar-header {
  padding: 16px 5px 8px 5px;
  //margin-bottom: 8px;
  text-align: center;
  font-weight: 700;
  display: flex;
  justify-content: space-between; /* 水平居中 */
  align-items: center; /* 垂直居中 */
}


.content-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
}

.input-foot-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  height: 100%;
  margin-right: 8px;
}

.input-foot-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.input-foot-left span {
  font-size: 14px;
  line-height: 18px;
  color: #252b3a;
  cursor: pointer;
}

.input-foot-dividing-line {
  width: 1px;
  height: 14px;
  background-color: #d7d8da;
}

.input-foot-maxlength {
  font-size: 14px;
  color: #71757f;
}

.demo-button-content {
  color: black;
}

.input-foot-right .demo-button-content {
  font-size: 14px;
}

.input-foot-right > *:not(:first-child) {
  margin-left: 8px;
}
</style>
