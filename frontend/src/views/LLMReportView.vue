<template>
  <main>
    <div class="tab-container">
      <button
        class="tab-button"
        :class="{ active: currentTab === 1 }"
        @click="switchTab(1)"
      >
        日报
      </button>
      <button
        class="tab-button"
        :class="{ active: currentTab === 2 }"
        @click="switchTab(2)"
      >
        周报
      </button>
    </div>
    <div style="display: flex; justify-content: center" v-if="showButton">
      <button class="tab-button" @click="genDailyReport(0)">
        {{ genDailyReportStr }}
      </button>
    </div>
    <div class="reports-container" v-if="reports.length > 0">
      <div class="report-card" v-for="report in reports" :key="report.id">
        <div class="report-header">
          <h2 class="report-title" @click="viewReportDetail(report.id)">
            {{ report.question }}
          </h2>
          <span class="li-meta" v-if="currentTab == 1">
            <span
              class="iconfont"
              style="color: #fcc630; font-size: 25px"
              @click.stop="playerPodcast(report)"
              >{{ iconHandel(report) }}</span
            >
          </span>
        </div>
        <div class="report-meta">
          <span v-if="currentTab === 2">日期范围: {{ report.time_array }}</span>
          <span
            >生成时间:
            {{ moment(report.created_at).format("MM-DD HH:mm:ss") }}</span
          >
        </div>
      </div>
    </div>

    <div class="no-reports" v-else>
      <LoadingView2 v-if="loading" />
      <p v-else>暂无报告数据</p>
    </div>

    <div
      class="audio-player-container"
      v-show="showPlayer && audioList.length > 0"
    >
      <AudioPlayer
        ref="audioPlayerRef"
        :audio-data="audioList[0]"
        :auto-play="true"
        @ended="endedHandler"
        @close="togglePlayer"
        @time-update="onTimeUpdate"
        :icon-style="{ color: '#ffffff', fontSize: '20px' }"
      />
    </div>
  </main>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onActivated,
  onBeforeUnmount,
  onDeactivated,
  onMounted,
  ref,
  watch,
} from "vue";
import { fetchReports, genDailyReport } from "@/api/reports";
import type { Report } from "@/types/types";
import moment from "moment";
import LoadingView2 from "@/components/LoadingView2.vue";
import { iconHandel } from "@/util/tools.ts";
import AudioPlayer from "@/components/AudioPlayer.vue";

const reports = ref<Report[]>([]);
const loading = ref<boolean>(true);
const currentTab = ref<number>(1); // 默认显示周报
const genDailyReportStr = ref<string>("生成报告"); // 默认显示周报

// 获取报告列表
const loadReports = async () => {
  try {
    loading.value = true;
    reports.value = await fetchReports(currentTab.value);
  } catch (error) {
    console.error("获取报告列表失败:", error);
  } finally {
    loading.value = false;
  }
};

const switchTab = (tab: number) => {
  currentTab.value = tab;
};

const viewReportDetail = (id: number) => {
  const apiUrl = import.meta.env.VITE_APP_API_URL || "http://localhost:8080";
  window.open(`${apiUrl}/api/reports/${id}/llm_result`, "_blank");
};

// 监听标签页变化
watch(currentTab, () => {
  loadReports();
});

const showButton = computed(() => {
  // 只有在日报(tab=1)且不处于加载状态时才考虑显示按钮
  if (currentTab.value == 1 && !loading.value) {
    if (reports.value.length > 0) {
      let first_report = reports.value[0];
      let d = moment().subtract(1, "days").format("YYYY-MM-DD");
      if (first_report!.question.indexOf(d) == -1) {
        genDailyReportStr.value = "立即生成";
        return true;
      } else {
        return false;
      }
    }
  }
  return false;
});

onMounted(() => {
  loadReports();
  window.scrollTo(0, 0);
});

onDeactivated(() => {
  // 当组件被 keep-alive 缓存时，暂停音频播放
  pauseAudioOnDeactivated();
});

onActivated(() => {
  // 当从缓存恢复时不需要特殊处理，AudioPlayer会保持自己的状态
});

// 定义音频项的类型，符合 vue-audio-player 组件的要求
interface AudioItem {
  src: string; // 音频文件的 URL
  title: string; // 音频标题
  album?: string; // 专辑名称（可选）
  artist?: string; // 艺术家（可选）
  cover?: string; // 封面图片 URL（可选）
}

const currentPodcast = ref<Report>();
const audioPlayerRef = ref();
const showPlayer = ref<boolean>(false);
const audioList = ref<AudioItem[]>([]);

// 添加用于存储播放位置的引用
const currentPosition = ref<number>(0);

// 计算属性，安全地解析 llm_result
const playedPodcasts = computed(() => {
  return new Set();
});

const playerPodcast = async (p: Report) => {
  if (p.podcast_mp3_url == "") {
    const result = await genDailyReport(p.id);
    // 如果返回了 HTML 内容，在新窗口中打开
    if (result) {
      // 使用 Blob URL 方式打开 HTML 内容（现代方法，避免使用 document.write）
      const blob = new Blob([result], { type: "text/html" });
      const url = URL.createObjectURL(blob);
      window.open(url, "_blank");
      // 可选：在页面卸载时清理 URL
      setTimeout(() => URL.revokeObjectURL(url), 60000);
    }

    return;
  }
  // 先确保播放器是关闭状态，避免状态冲突
  showPlayer.value = false;

  // 在下一个tick中初始化新的音频
  nextTick(() => {
    // 先清空音频列表
    const apiUrl = import.meta.env.VITE_APP_API_URL;
    audioList.value = [];
    audioList.value.push({
      src: `${apiUrl}/api/reports/${p.id}/play`,
      title: p.question,
      album: p.question,
    });
    currentPodcast.value = p;

    // 检查是否有保存的播放位置
    const savedPosition = localStorage.getItem(`podcast-position-${p.id}`);
    if (savedPosition) {
      // 设置播放位置为保存的时间减去10秒
      currentPosition.value = Math.max(0, parseFloat(savedPosition) - 10);
    } else {
      currentPosition.value = 0;
    }

    // 缓存已播放的播客
    cacheToLocalStorage(p.id);

    showPlayer.value = true;

    // 在下一个tick中设置播放位置
    nextTick(() => {
      if (audioPlayerRef.value && currentPosition.value > 0) {
        // 使用audioPlayer的暴露方法设置播放位置
        audioPlayerRef.value.setCurrentTime(currentPosition.value);
      }
    });
  });
};

//  缓存已播放的播客
const cacheToLocalStorage = (id: number) => {
  // 缓存已播放的播客 MD5
  playedPodcasts.value.add(id);
};
const togglePlayer = () => {
  // 保存当前播放位置
  if (audioPlayerRef.value && currentPodcast.value) {
    const currentTime = audioPlayerRef.value.getCurrentTime();
    localStorage.setItem(
      `podcast-position-${currentPodcast.value.id}`,
      currentTime.toString(),
    );
  }

  // 先隐藏播放器再清空列表
  showPlayer.value = false;
  setTimeout(() => {
    audioList.value = [];
    currentPosition.value = 0; // 重置当前位置
  }, 50);
};

const endedHandler = () => {
  if (currentPodcast.value) {
    // 播放完成后清除保存的位置
    localStorage.removeItem(`podcast-position-${currentPodcast.value.id}`);

    if (currentPodcast.value.question) {
      // 可以在这里添加播放结束后的处理逻辑
      console.log("Audio playback ended for:", currentPodcast.value.question);
    }
  }
};

// 在组件停用前暂停音频播放
const pauseAudioOnDeactivated = () => {
  // AudioPlayer组件内部会处理自己的状态
};

// 在组件卸载前正确清理音频播放器
const cleanupAudioPlayer = () => {
  // AudioPlayer组件内部会处理自己的状态
  showPlayer.value = false;
  audioList.value = [];
};

onBeforeUnmount(() => {
  cleanupAudioPlayer();
});

// 实现处理音频播放时间更新的方法
const onTimeUpdate = (time: number) => {
  // 更新当前播放位置
  currentPosition.value = time;
  // 可以在这里处理音频播放时间更新的逻辑
  // 例如：记录播放位置、更新进度条等
  // console.log("当前播放时间:", time);
};
</script>

<style scoped>
@import "../assets/icon.css";

main {
  min-height: 100vh;
  padding: 2rem 1rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.tab-container {
  display: flex;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 2rem;
}

.tab-button {
  padding: 0.5rem 1.5rem;
  border: none;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.8);
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 1rem;
  /* 防止点击时出现正方形效果 */
  -webkit-tap-highlight-color: transparent;
  outline: none;
  user-select: none;
}

.tab-button:hover {
  background: rgba(255, 255, 255, 0.25);
}

.tab-button.active {
  background: rgba(255, 255, 255, 0.3);
  color: white;
  font-weight: bold;
}

.reports-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 800px;
  margin: 0 auto;
  padding: 10px;
}

.report-card {
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(10px);
  border-radius: 12px;
  padding: 1.2rem;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.report-card:hover {
  background: rgba(255, 255, 255, 0.25);
  transform: translateY(-2px);
  box-shadow: 0 6px 12px rgba(0, 0, 0, 0.15);
}

.report-header {
  margin-bottom: 0.8rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.report-title {
  color: white;
  font-size: 1.1rem;
  margin: 0; /* 移除默认的 margin */
  font-weight: 600;
  line-height: 1.4;
  align-self: center; /* 确保标题垂直居中 */
  -webkit-touch-callout: none; /* 禁止 iOS 弹出菜单 */
  user-select: none;
}

.report-meta {
  display: flex;
  justify-content: space-between;
  color: rgba(255, 255, 255, 0.8);
  font-size: 0.85rem;
}

.no-reports {
  text-align: center;
  color: rgba(255, 255, 255, 0.8);
  font-size: 1.1rem;
  padding: 3rem 1rem;
}

@media (max-width: 768px) {
  main {
    padding: 1rem 0.5rem;
  }

  .tab-container {
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .tab-button {
    padding: 0.4rem 1rem;
    font-size: 0.9rem;
  }

  .report-card {
    padding: 1rem;
  }

  .report-meta {
    flex-direction: column;
    gap: 0.3rem;
  }
}

/* 固定在底部的音频播放器容器 */
.audio-player-container {
  position: fixed;
  bottom: 0;
  left: 0;
  width: 100%;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.1);
  z-index: 1000;

  box-sizing: border-box;
}

/* 滚动动画 */
@keyframes marquee {
  0% {
    transform: translateX(100%);
  }
  100% {
    transform: translateX(-100%);
  }
}
</style>
