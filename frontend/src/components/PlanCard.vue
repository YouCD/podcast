<template>
  <div class="plan-card" v-if="plan">
    <div class="plan-header" @click="toggleExpand">
      <div class="plan-icon">
        <span class="icon">📋</span>
      </div>
      <div class="plan-title">
        <span class="title-text">执行计划</span>
        <span class="query-text">{{ plan.query }}</span>
      </div>
      <div class="plan-status">
        <span :class="['status-badge', isComplete ? 'completed' : 'running']">
          {{ isComplete ? '已完成' : '执行中' }}
        </span>
        <span class="steps-count">{{ stepsCount }} 个步骤</span>
        <i :class="['expand-icon', { collapsed: !expanded }]">▼</i>
      </div>
    </div>
    
    <div class="plan-steps" v-if="expanded && plan.steps && plan.steps.length > 0">
      <div class="steps-header">
        <span>执行步骤 ({{ plan.steps.length }})</span>
      </div>
      <div 
        v-for="(step, index) in plan.steps" 
        :key="step.id"
        :class="['step-item', `step-status-${getStepStatus(step.id)}`]"
      >
        <div class="step-indicator">
          <div class="step-number">{{ step.id }}</div>
          <div class="step-line" v-if="index < plan.steps.length - 1"></div>
        </div>
        <div class="step-content">
          <div class="step-header-row">
            <span class="step-description">{{ step.description }}</span>
            <span :class="['step-status', getStepStatus(step.id)]">
              {{ getStatusText(getStepStatus(step.id)) }}
            </span>
          </div>
          <div class="step-tool">
            <span class="tool-label">工具:</span>
            <span class="tool-name">{{ step.tool_name }}</span>
          </div>
          <div class="step-reason" v-if="step.reason">
            <span class="reason-label">原因:</span>
            <span class="reason-text">{{ step.reason }}</span>
          </div>
          
          <!-- 执行结果 -->
          <div class="step-result-section" v-if="getStepResult(step.id)">
            <div class="result-header" @click.stop="toggleResultExpand(step.id)">
              <span class="result-icon">📊</span>
              <span class="result-title">执行结果</span>
              <i :class="['result-expand-arrow', { expanded: isResultExpanded(step.id) }]">▶</i>
            </div>
            <div class="result-content" v-if="isResultExpanded(step.id)">
              <div class="result-text" v-if="isJsonResult(step.id)">
                <pre>{{ formatResult(step.id) }}</pre>
              </div>

              <div class="result-text" v-else>{{ getStepResult(step.id) }}  </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import type { PlanData, StepInfo } from '@/types/types';

const props = defineProps<{
  plan: PlanData;
  steps?: StepInfo[];
}>();

// 调试日志
watch(() => props.steps, (newSteps) => {
  console.log('PlanCard received steps:', newSteps);
  if (newSteps && newSteps.length > 0) {
    console.log('First step:', newSteps[0]);
    console.log('First step result:', newSteps[0]?.result);
  }
}, { immediate: true });

const expanded = ref(true);
const expandedResults = ref<Set<number>>(new Set());

const isComplete = computed(() => props.plan?.is_complete ?? false);

const stepsCount = computed(() => {
  if (props.plan?.steps && Array.isArray(props.plan.steps)) {
    return props.plan.steps.length;
  }
  return 0;
});

const toggleExpand = () => {
  expanded.value = !expanded.value;
};

const toggleResultExpand = (stepId: number) => {
  if (expandedResults.value.has(stepId)) {
    expandedResults.value.delete(stepId);
  } else {
    expandedResults.value.add(stepId);
  }
};

const isResultExpanded = (stepId: number) => {
  return expandedResults.value.has(stepId);
};

// 获取步骤的执行状态
const getStepStatus = (stepId: number): 'pending' | 'running' | 'completed' | 'failed' => {
  if (!props.steps) return 'pending';
  const stepInfo = props.steps.find(s => s.step_id === stepId);
  return stepInfo?.status || 'pending';
};

// 获取步骤的执行结果
const getStepResult = (stepId: number): string | undefined => {
  if (!props.steps) return undefined;
  const stepInfo = props.steps.find(s => s.step_id === stepId);
  return stepInfo?.result;
};

// 检查结果是否为 JSON
const isJsonResult = (stepId: number): boolean => {
  const result = getStepResult(stepId);
  if (!result) return false;
  try {
    JSON.parse(result);
    return true;
  } catch {
    return false;
  }
};

// 格式化 JSON 结果
const formatResult = (stepId: number): string => {
  const result = getStepResult(stepId);
  if (!result) return '';
  try {
    return JSON.stringify(JSON.parse(result), null, 2);
  } catch {
    return result;
  }
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
</script>

<style scoped>
.plan-card {
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border: 1px solid #e9ecef;
  border-radius: 12px;
  margin-bottom: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.plan-header {
  display: flex;
  align-items: center;
  padding: 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.plan-header:hover {
  background: #f1f3f5;
}

.plan-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 10px;
  margin-right: 12px;
}

.plan-icon .icon {
  font-size: 20px;
}

.plan-title {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title-text {
  font-weight: 600;
  font-size: 15px;
  color: #333;
}

.query-text {
  font-size: 13px;
  color: #666;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 400px;
}

.plan-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.running {
  background: #fff3cd;
  color: #856404;
}

.status-badge.completed {
  background: #d4edda;
  color: #155724;
}

.steps-count {
  font-size: 12px;
  color: #999;
}

.expand-icon {
  font-size: 12px;
  color: #999;
  transition: transform 0.3s;
}

.expand-icon.collapsed {
  transform: rotate(180deg);
}

.plan-steps {
  border-top: 1px solid #e9ecef;
  padding: 16px;
  background: #fafbfc;
}

.steps-header {
  font-size: 13px;
  color: #666;
  margin-bottom: 12px;
  font-weight: 500;
}

.step-item {
  display: flex;
  gap: 12px;
  padding: 12px 0;
}

.step-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 28px;
}

.step-number {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  background: #e9ecef;
  color: #666;
}

.step-status-running .step-number {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  animation: pulse 1.5s infinite;
}

.step-status-completed .step-number {
  background: #28a745;
  color: white;
}

.step-status-failed .step-number {
  background: #dc3545;
  color: white;
}

.step-line {
  width: 2px;
  flex: 1;
  background: #e9ecef;
  margin-top: 4px;
}

.step-content {
  flex: 1;
  padding-bottom: 8px;
}

.step-header-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 6px;
}

.step-description {
  font-size: 14px;
  color: #333;
  font-weight: 500;
  flex: 1;
}

.step-status {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  margin-left: 8px;
}

.step-status.pending {
  background: #f8f9fa;
  color: #6c757d;
}

.step-status.running {
  background: #fff3cd;
  color: #856404;
}

.step-status.completed {
  background: #d4edda;
  color: #155724;
}

.step-status.failed {
  background: #f8d7da;
  color: #721c24;
}

.step-tool {
  font-size: 12px;
  margin-bottom: 4px;
}

.tool-label {
  color: #999;
}

.tool-name {
  color: #667eea;
  font-weight: 500;
  margin-left: 4px;
}

.step-reason {
  font-size: 12px;
  color: #666;
  line-height: 1.5;
  margin-bottom: 8px;
}

.reason-label {
  color: #999;
}

.reason-text {
  margin-left: 4px;
}

/* 执行结果样式 */
.step-result-section {
  margin-top: 10px;
  border: 1px solid #e9ecef;
  border-radius: 8px;
  overflow: hidden;
}

.result-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 12px;
  background: #f8f9fa;
  cursor: pointer;
  transition: background 0.2s;
}

.result-header:hover {
  background: #f1f3f5;
}

.result-icon {
  font-size: 14px;
}

.result-title {
  font-size: 12px;
  font-weight: 500;
  color: #666;
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

.result-content {
  background: #282c34;
  padding: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.result-text pre {
  margin: 0;
  font-size: 11px;
  color: #abb2bf;
  font-family: 'Fira Code', 'Consolas', monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

.result-text {
  font-size: 12px;
  color: #abb2bf;
  line-height: 1.5;
}

@keyframes pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(102, 126, 234, 0.4);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(102, 126, 234, 0);
  }
}
</style>
