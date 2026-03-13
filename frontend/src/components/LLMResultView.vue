<script setup lang="ts">
import type { llmResult } from "@/types/types.ts";
import { indexParser } from "@/util/tools.ts";
import { defineProps, type PropType } from "vue";
import ShareButton from "@/components/ShareButton.vue";

const props = defineProps({
  llm_result: Object as PropType<llmResult>,
  title: String,
  source: String,
  categories: String,
  link: String,
  date: String,
});

const link = () => {
  window.open(props.link, "_blank");
};
</script>

<template>
  <div class="container">
    <div class="line"></div>
    <h1 class="title">{{ props.title }}</h1>

    <p class="subtitle">
      {{ props.llm_result!.subtitle }} &nbsp;&nbsp; {{ props.date }}
    </p>
    <p class="action">
      {{ props.source?.replaceAll(" ", "") }}&nbsp;&nbsp;{{
        props.categories
      }}
      &nbsp;&nbsp;<button class="button" @click="link">原文</button>
      <ShareButton :file_name="props.title" />
    </p>

    <div class="contentSummary_container shadow">
      <p>
        {{ props.llm_result!.contentSummary }}
      </p>
    </div>

    <div class="keyPoints shadow">
      <ul>
        <li v-for="(key, index) in props.llm_result!.keyPoints" :key="index">
          <span>{{ indexParser(index) }}</span>
          <span> {{ key }}</span>
        </li>
      </ul>
    </div>
    <div class="specifics shadow">
      <template
        v-for="(key, index) in props.llm_result!.specifics"
        :key="index"
      >
        <p v-if="key.length > 0">
          {{ index }}：<span>{{ key }}</span>
        </p>
      </template>
    </div>
    <blockquote class="opinion shadow">
      <p>
        {{ props.llm_result!.opinion }}
      </p>
    </blockquote>
    <p class="summarize" v-html="props.llm_result!.summarize" />
  </div>
</template>

<style scoped>
@import "../assets/button.css";

:deep(summarize) {
  background: linear-gradient(
    90deg,
    rgba(102, 126, 234, 0.2) 0%,
    rgba(118, 75, 162, 0.2) 100%
  );
  color: #5a4fcf;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.container {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
  background: rgba(255, 255, 255, 0.85);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  backdrop-filter: blur(4px);
}

.line {
  height: 4px;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 2px;
  margin-bottom: 24px;
}

.title {
  color: #5a4fcf;
  font-size: 25px;
  margin: 0 0 8px 0;
  font-weight: 700;
}

.subtitle {
  color: #666;
  font-size: 14px;
  margin: 0 0 10px 0;
}

.action {
  color: #666;
  font-size: 14px;
  margin: 0 0 10px 0;
}

.action a {
  text-decoration: none;
}

.contentSummary_container {
  margin: 24px 0;
  padding: 20px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  border: 1px solid rgba(102, 126, 234, 0.2);
}

.contentSummary_container p {
  margin: 0;
  color: #333;
  line-height: 1.7;
}

.keyPoints {
  margin: 24px 0;
  padding: 20px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  border: 1px solid rgba(102, 126, 234, 0.3);
}

.keyPoints ul {
  color: #333;
  padding-left: 0;
  margin: 0;
  list-style: none;
}

.keyPoints li {
  margin-bottom: 12px;
  display: flex;
  align-items: flex-start;
}

.keyPoints li span:first-child {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgba(102, 126, 234, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  font-size: 12px;
  font-weight: 600;
  color: #667eea;
}

.specifics {
  margin: 24px 0;
  padding: 16px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  border-left: 4px solid #667eea;
  border-right: 1px solid rgba(102, 126, 234, 0.3);
  border-top: 1px solid rgba(102, 126, 234, 0.3);
  border-bottom: 1px solid rgba(102, 126, 234, 0.3);
}

.shadow {
  /* 关键：一层柔和的阴影 */
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transition:
    transform 0.25s ease,
    box-shadow 0.25s ease;
}

/* 可选：鼠标悬停时再抬高一点，增强悬浮感 */
.shadow:hover {
  transform: translateY(-4px); /* 再抬高 4 px */
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.15);
}

.specifics p {
  margin: 0 0 8px 0;
  color: #667eea;
  font-weight: 600;
  font-size: 14px;
}

.specifics span {
  margin: 0;
  color: #333;
  line-height: 1.6;
}

.opinion {
  border-left: 4px solid #667eea;
  margin: 24px 0;
  background: #f8f7ff;
  padding: 16px 20px;
  border-radius: 0 8px 8px 0;
}

.opinion p {
  margin: 0;
  color: #5a4fcf;
  /*  font-style: italic; */
  line-height: 1.6;
}

.summarize {
  color: #333;
  line-height: 1.7;
}
</style>
