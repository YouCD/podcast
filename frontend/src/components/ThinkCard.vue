<template>
  <div class="think-card">
    <div class="think-header" @click="toggleExpand">
      <div class="think-icon">
        <span class="icon">🤔</span>
      </div>
      <div class="think-title">
        <span class="title-text">思考过程</span>
        <span class="think-preview" v-if="!expanded">{{ previewText }}</span>
      </div>
      <div class="think-toggle">
        <i :class="['toggle-arrow', { expanded }]">▶</i>
      </div>
    </div>
    
    <div class="think-content" v-if="expanded">
      <Typing :enableTyping="enableTyping" :msg="content"></Typing>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import Typing from './Typing.vue';

const props = defineProps<{
  content: string;
  enableTyping?: boolean;
}>();

const expanded = ref(false);

const toggleExpand = () => {
  expanded.value = !expanded.value;
};

const previewText = computed(() => {
  if (!props.content) return '';
  const text = props.content.replace(/\n/g, ' ');
  return text.length > 50 ? text.substring(0, 50) + '...' : text;
});
</script>

<style scoped>
.think-card {
  background: linear-gradient(135deg, #fff9e6 0%, #fff3cd 100%);
  border: 1px solid #ffc107;
  border-radius: 10px;
  margin-bottom: 12px;
  overflow: hidden;
}

.think-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.think-header:hover {
  background: rgba(255, 193, 7, 0.1);
}

.think-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #ffc107;
  border-radius: 8px;
  margin-right: 12px;
}

.think-icon .icon {
  font-size: 16px;
}

.think-title {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.title-text {
  font-size: 14px;
  font-weight: 600;
  color: #856404;
}

.think-preview {
  font-size: 12px;
  color: #997a00;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.think-toggle {
  display: flex;
  align-items: center;
}

.toggle-arrow {
  font-size: 10px;
  color: #856404;
  transition: transform 0.3s;
}

.toggle-arrow.expanded {
  transform: rotate(90deg);
}

.think-content {
  border-top: 1px solid #ffc107;
  padding: 16px;
  background: rgba(255, 255, 255, 0.5);
  font-size: 14px;
  line-height: 1.6;
  color: #666;
}
</style>
