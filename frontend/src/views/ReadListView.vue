<script setup lang="ts">
import {onMounted, ref} from "vue";
import type {Rss} from "@/types/types.ts";
import {fetchRead24hRss} from "@/api/rss.ts";
import router from "@/router";
import MainView from "@/components/MainView.vue";
import LoadingView2 from "@/components/LoadingView2.vue";

const showNodata = ref<boolean>(true);
const posts = ref<Rss[]>([]);

onMounted(() => {
  fetchCategoryPostsLocal();
  window.scrollTo(0, 0);
});

const fetchCategoryPostsLocal = async () => {
  try {
    const list = await fetchRead24hRss();
    posts.value = list;
    setTimeout(() => {
      showNodata.value = list.length === 0;
    }, 500);
  } catch (e) {
    console.error(`获取 失败:`, e);
  }
};
const openLink = (p: Rss) => {
  router.push({
    name: "llm_html",
    params: {id: p.id.toString()},
    query: {md5: p.md5},
  });
};
</script>

<template>
  <main>
    <LoadingView2 v-if="showNodata"/>

    <MainView v-if="!showNodata">
      <ul class="glass-list">
        <li v-for="p in posts" :key="p.id">
          <div class="li-header">
            <p class="li-title" @click="openLink(p)">{{ p.title }}</p>
          </div>
        </li>
      </ul>
    </MainView>
  </main>
</template>

<style scoped>
@import "../assets/icon.css";

main {
  min-height: 100vh;
  padding: 2rem 0 100px 0; /* 为固定在底部的播放器留出空间 */
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  flex-direction: column;
  width: 100%;
  gap: 1.5rem;
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
}

/* -------------- 玻璃列表 -------------- */
.glass-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 85dvh;
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
  /*
  padding: 0.1rem 1rem;
  margin-bottom: 0.6rem;
  background: rgba(255, 255, 255, 0.2);
   */
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
</style>
