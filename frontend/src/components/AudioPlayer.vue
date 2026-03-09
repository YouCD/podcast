<template>
  <div class="audio-player">
    <audio
        ref="audioRef"
        :src="audioData?.src"
        @ended="onEnded"
        @loadedmetadata="onLoadedMetadata"
        @timeupdate="onTimeUpdate"
        @play="onPlay"
        @pause="onPause"
    ></audio>

    <div class="close-button" @click="onClose">×</div>
    <div class="player-title" v-if="audioData?.title">
      <div class="title-text">{{ audioData.title }}</div>
    </div>

    <div class="player-controls">
      <button class="control-button" @click="togglePlay">
        <span v-if="!isPlaying" class="play-icon">
          <svg class="icon" aria-hidden="true" :style="iconStyle">
            <use xlink:href="#icon-24gl-playCircle"></use>
          </svg>
        </span>
        <span v-else class="pause-icon">
          <svg class="icon" :style="iconStyle" aria-hidden="true">
            <use xlink:href="#icon-zanting"></use>
          </svg>
        </span>
      </button>

      <div class="progress-container" @click="onProgressClick">
        <div class="progress-bar">
          <div
              class="progress-loaded"
              :style="{ width: loadedProgress + '%' }"
          ></div>
          <div
              class="progress-played"
              :style="{ width: playedProgress + '%' }"
          ></div>
          <div
              class="progress-handle"
              :style="{ left: playedProgress + '%' }"
          ></div>
        </div>
      </div>

      <div class="time-display">
        <span>{{ formatTime(currentTime) }} / {{ formatTime(duration) }}</span>
      </div>

      <button class="control-button" @click="toggleMute">
        <span v-if="isMuted || volume === 0" class="mute-icon">
          <svg class="icon" :style="iconStyle" aria-hidden="true">
            <use xlink:href="#icon-shengyinjingyin"></use>
          </svg>
        </span>
        <span v-else-if="volume > 0.5" class="volume-high-icon">
          <svg class="icon" :style="iconStyle" aria-hidden="true">
            <use xlink:href="#icon-yinliang"></use>
          </svg>
        </span>
        <span v-else class="volume-low-icon">
          <svg class="icon" :style="iconStyle" aria-hidden="true">
            <use xlink:href="#icon-yinliang"></use>
          </svg>
        </span>
      </button>

      <input
          type="range"
          class="volume-slider"
          min="0"
          max="1"
          step="0.01"
          :value="volume"
          @input="onVolumeChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import {onBeforeUnmount, onMounted, ref, watch} from "vue";

// 定义音频数据接口
interface AudioData {
  src?: string;
  title?: string;
}

const props = defineProps<{
  audioData?: AudioData;
  autoPlay?: boolean;
  iconStyle?: Record<string, string>; // 添加自定义样式prop
}>();

const emit = defineEmits<{
  (e: "ended"): void;
  (e: "close"): void;
  (e: "time-update", time: number): void;
}>();

const audioRef = ref<HTMLAudioElement | null>(null);
const isPlaying = ref(false);
const isMuted = ref(false);
const duration = ref(0);
const currentTime = ref(0);
const volume = ref(1);
const playedProgress = ref(0);
const loadedProgress = ref(0);

// 格式化时间显示
const formatTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs < 10 ? "0" : ""}${secs}`;
};

// 播放方法
const play = () => {
  if (!audioRef.value) return;

  audioRef.value.play().catch((err) => {
    console.warn("Playback failed:", err);
  });
};

// 暂停方法
const pause = () => {
  if (!audioRef.value) return;

  audioRef.value.pause();
};

// 播放/暂停切换
const togglePlay = () => {
  if (!audioRef.value) return;

  if (isPlaying.value) {
    pause();
  } else {
    play();
  }
};

// 静音切换
const toggleMute = () => {
  if (!audioRef.value) return;

  isMuted.value = !isMuted.value;
  audioRef.value.muted = isMuted.value;
};

// 音量变化处理
const onVolumeChange = (event: Event) => {
  if (!audioRef.value) return;

  const target = event.target as HTMLInputElement;
  const newVolume = parseFloat(target.value);
  volume.value = newVolume;
  audioRef.value.volume = newVolume;
  isMuted.value = newVolume === 0;
};

// 进度条点击处理
const onProgressClick = (event: MouseEvent) => {
  if (!audioRef.value) return;

  const progressBar = (event.target as HTMLElement).closest(".progress-bar");
  if (!progressBar) return;

  const rect = progressBar.getBoundingClientRect();
  const pos = (event.clientX - rect.left) / rect.width;
  const time = pos * duration.value;

  audioRef.value.currentTime = time;
};

// 播放事件处理
const onPlay = () => {
  isPlaying.value = true;
};

// 暂停事件处理
const onPause = () => {
  isPlaying.value = false;
};

// 加载元数据
const onLoadedMetadata = () => {
  if (audioRef.value) {
    duration.value = audioRef.value.duration || 0;

    // 如果设置了自动播放，则开始播放
    if (props.autoPlay) {
      play();
    }
  }
};

// 时间更新
const onTimeUpdate = () => {
  if (audioRef.value) {
    currentTime.value = audioRef.value.currentTime;
    playedProgress.value = (currentTime.value / duration.value) * 100;
    // 向父组件发送当前播放时间
    emit("time-update", currentTime.value);
  }
};

// 进度加载更新
const updateLoadProgress = () => {
  if (audioRef.value) {
    const buffered = audioRef.value.buffered;
    if (buffered.length > 0) {
      loadedProgress.value =
          (buffered.end(buffered.length - 1) / duration.value) * 100;
    }
  }
};

// 播放结束
const onEnded = () => {
  isPlaying.value = false;
  emit("ended");
};

// 关闭播放器
const onClose = () => {
  emit("close");
};

// 监听音频源变化
watch(
    () => props.audioData?.src,
    (newSrc) => {
      if (audioRef.value && newSrc) {
        audioRef.value.src = newSrc;
        // 重置状态
        currentTime.value = 0;
        playedProgress.value = 0;
        isPlaying.value = false;
      }
    },
);

// 定时更新加载进度
let progressUpdateTimer: ReturnType<typeof setInterval> | null = null;

onMounted(() => {
  if (audioRef.value) {
    audioRef.value.addEventListener("progress", updateLoadProgress);
    progressUpdateTimer = setInterval(updateLoadProgress, 1000);
  }
});

onBeforeUnmount(() => {
  if (audioRef.value) {
    audioRef.value.removeEventListener("progress", updateLoadProgress);
  }
  if (progressUpdateTimer) {
    clearInterval(progressUpdateTimer);
  }
});

// 添加设置当前播放时间的方法
const setCurrentTime = (time: number) => {
  if (audioRef.value) {
    audioRef.value.currentTime = time;
  }
};

// 添加获取当前播放时间的方法
const getCurrentTime = () => {
  if (audioRef.value) {
    return audioRef.value.currentTime;
  }
  return 0;
};

// 暴露方法给父组件
defineExpose({
  togglePlay,
  play,
  pause,
  setCurrentTime,
  getCurrentTime,
});
</script>

<style scoped>
@import "@/assets/icon.css";

.audio-player {
  background: rgba(255, 255, 255, 0.15);
  border-radius: 20px;
  padding: 10px 15px;
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  position: relative;
}

.close-button {
  position: absolute;
  top: 5px;
  right: 10px;
  font-size: 24px;
  font-weight: bold;
  color: white;
  cursor: pointer;
  z-index: 1001;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background-color 0.3s;
}

.close-button:hover {
  background-color: rgba(0, 0, 0, 0.1);
}

.player-title {
  margin-top: 10px;
  margin-bottom: 8px;
  text-align: center;
}

.title-text {
  color: white;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.player-controls {
  display: flex;
  align-items: center;
  gap: 10px;
}

.control-button {
  background: transparent;
  border: none;
  color: white;
  cursor: pointer;
  font-size: 16px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background-color 0.2s;
}

.control-button:hover {
  background: rgba(255, 255, 255, 0.2);
}

.progress-container {
  flex: 1;
  cursor: pointer;
}

.progress-bar {
  position: relative;
  height: 5px;
  background: rgba(255, 255, 255, 0.3);
  border-radius: 3px;
}

.progress-loaded {
  position: absolute;
  height: 100%;
  background: rgba(255, 255, 255, 0.5);
  border-radius: 3px;
}

.progress-played {
  position: absolute;
  height: 100%;
  background: #fff;
  border-radius: 3px;
}

.progress-handle {
  position: absolute;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 12px;
  height: 12px;
  background: #fff;
  border-radius: 50%;
  margin-top: 0;
}

.time-display {
  color: white;
  font-size: 12px;
  min-width: 70px;
  text-align: center;
}

.volume-slider {
  width: 60px;
  height: 5px;
  -webkit-appearance: none;
  background: rgba(255, 255, 255, 0.3);
  border-radius: 3px;
  outline: none;
}

.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 12px;
  height: 12px;
  background: #fff;
  border-radius: 50%;
  cursor: pointer;
}

/* 隐藏默认的 audio 元素 */
audio {
  display: none;
}
</style>
