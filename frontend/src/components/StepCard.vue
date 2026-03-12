<template>
  <div :class="['step-card', `status-${step.status}`]">
    <div class="step-header" @click="toggleExpand">
      <div class="step-left">
        <div :class="['step-icon', step.status]">
          <span v-if="step.status === 'running'" class="spinner"></span>
          <span v-else-if="step.status === 'completed'" class="checkmark">✓</span>
          <span v-else-if="step.status === 'failed'" class="cross">✗</span>
          <span v-else class="pending-dot">●</span>
        </div>
        <div class="step-info">
          <div class="step-title">
            <span class="step-id">步骤 {{ step.step_id }}</span>
            <span class="step-desc">{{ step.description }}</span>
          </div>
          <div class="step-tool-info">
            <span class="tool-badge">{{ step.tool_name }}</span>
          </div>
        </div>
      </div>
      <div class="step-right">
        <span :class="['status-text', step.status]">{{ getStatusText(step.status) }}</span>
        <i :class="['expand-arrow', { expanded: step.expanded }]">▶</i>
      </div>
    </div>
    
    <div class="step-details" v-if="step.expanded">
      <!-- 工具参数 -->
      <div class="detail-section" v-if="step.tool_args && Object.keys(step.tool_args).length > 0">
        <div class="section-title">
          <span class="section-icon">⚙️</span>
          <span>工具参数</span>
        </div>
        <div class="tool-args">
          <pre>{{ JSON.stringify(step.tool_args, null, 2) }}</pre>
        </div>
      </div>
      
      <!-- 执行原因 -->
      <div class="detail-section" v-if="step.reason">
        <div class="section-title">
          <span class="section-icon">💡</span>
          <span>执行原因</span>
        </div>
        <div class="reason-content">{{ step.reason }}</div>
      </div>
      
      <!-- 执行结果 -->
      <div class="detail-section" v-if="step.result">
        <div class="section-title" @click.stop="toggleResultExpand">
          <span class="section-icon">📊</span>
          <span>执行结果</span>
          <i :class="['result-expand-arrow', { expanded: resultExpanded }]">▶</i>
        </div>
        <div class="result-content" v-if="resultExpanded">
          <div class="result-text" v-if="isJsonResult">
            <pre>{{ formattedResult }}</pre>
          </div>
          <div class="result-text" v-else>{{ step.result }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { StepInfo } from '@/types/types';

const props = defineProps<{
  step: StepInfo;
}>();

const emit = defineEmits<{
  (e: 'toggle-expand'): void;
}>();

const resultExpanded = ref(true);

const toggleExpand = () => {
  emit('toggle-expand');
};

const toggleResultExpand = (event: MouseEvent) => {
  event.stopPropagation();
  resultExpanded.value = !resultExpanded.value;
};

const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    pending: '待执行',
    running: '执行中',
    completed: '已完成',
    failed: '失败'
  };
  return statusMap[status] || status;
};

const isJsonResult = computed(() => {
  if (!props.step.result) return false;
  try {
    JSON.parse(props.step.result);
    return true;
  } catch {
    return false;
  }
});

const formattedResult = computed(() => {
  if (!props.step.result || !isJsonResult.value) return '';
  try {
    return JSON.stringify(JSON.parse(props.step.result), null, 2);
  } catch {
    return props.step.result;
  }
});
</script>

<style scoped>
.step-card {
  background: #fff;
  border: 1px solid #e9ecef;
  border-radius: 10px;
  margin-bottom: 10px;
  overflow: hidden;
  transition: all 0.3s;
}

.step-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.step-card.status-running {
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
}

.step-card.status-completed {
  border-color: #28a745;
}

.step-card.status-failed {
  border-color: #dc3545;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.step-header:hover {
  background: #f8f9fa;
}

.step-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.step-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}

.step-icon.pending {
  background: #f8f9fa;
  color: #adb5bd;
}

.step-icon.running {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.step-icon.completed {
  background: #28a745;
  color: white;
}

.step-icon.failed {
  background: #dc3545;
  color: white;
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.checkmark, .cross {
  font-weight: bold;
}

.pending-dot {
  font-size: 18px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.step-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.step-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.step-id {
  font-size: 12px;
  color: #999;
  font-weight: 500;
}

.step-desc {
  font-size: 14px;
  color: #333;
  font-weight: 500;
}

.step-tool-info {
  display: flex;
  align-items: center;
}

.tool-badge {
  font-size: 11px;
  padding: 2px 8px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border-radius: 10px;
  font-weight: 500;
}

.step-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-text {
  font-size: 12px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: 12px;
}

.status-text.pending {
  background: #f8f9fa;
  color: #6c757d;
}

.status-text.running {
  background: #fff3cd;
  color: #856404;
}

.status-text.completed {
  background: #d4edda;
  color: #155724;
}

.status-text.failed {
  background: #f8d7da;
  color: #721c24;
}

.expand-arrow {
  font-size: 10px;
  color: #999;
  transition: transform 0.3s;
}

.expand-arrow.expanded {
  transform: rotate(90deg);
}

.step-details {
  border-top: 1px solid #e9ecef;
  padding: 16px;
  background: #fafbfc;
}

.detail-section {
  margin-bottom: 14px;
}

.detail-section:last-child {
  margin-bottom: 0;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  color: #666;
  margin-bottom: 8px;
  cursor: pointer;
}

.section-icon {
  font-size: 14px;
}

.result-expand-arrow {
  font-size: 9px;
  color: #999;
  margin-left: auto;
  transition: transform 0.3s;
}

.result-expand-arrow.expanded {
  transform: rotate(90deg);
}

.tool-args {
  background: #282c34;
  border-radius: 8px;
  padding: 12px;
  overflow-x: auto;
}

.tool-args pre {
  margin: 0;
  font-size: 12px;
  color: #abb2bf;
  font-family: 'Fira Code', 'Consolas', monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

.reason-content {
  font-size: 13px;
  color: #666;
  line-height: 1.6;
  padding: 10px 12px;
  background: #fff;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.result-content {
  background: #282c34;
  border-radius: 8px;
  padding: 12px;
  overflow-x: auto;
  max-height: 400px;
  overflow-y: auto;
}

.result-text pre {
  margin: 0;
  font-size: 12px;
  color: #abb2bf;
  font-family: 'Fira Code', 'Consolas', monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

.result-text {
  font-size: 13px;
  color: #abb2bf;
  line-height: 1.5;
}
</style>
