<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useTemplateStore } from "@/stores/template";
import type {
  CreateTemplateRequest,
  UpdateTemplateRequest,
} from "@/types/types";
import notificationService from "@/components/Notification/notificationService";
import { getTruncatedData } from "@/util/tools.ts";

// 使用store
const templateStore = useTemplateStore();

// 表单相关
const showCreateForm = ref(false);
const showEditForm = ref(false);
const editingId = ref<number | null>(null);

// 折叠状态管理
const expandedTemplates = ref<Set<number>>(new Set());

const formData = ref({
  key_name: "",
  data: "",
});

// 加载数据
const loadData = async () => {
  const success = await templateStore.loadTemplates();
  if (!success) {
    notificationService.error("加载Template列表失败");
  }
};

// 创建Template
const handleCreate = async () => {
  if (!formData.value.key_name.trim() || !formData.value.data.trim()) {
    notificationService.warning("请输入完整的Template信息");
    return;
  }

  const requestData: CreateTemplateRequest = {
    key_name: formData.value.key_name,
    data: formData.value.data,
  };

  const result = await templateStore.createNewTemplate(requestData);
  if (result) {
    notificationService.success("Template创建成功");
    showCreateForm.value = false;
    resetForm();
  } else {
    notificationService.error("Template创建失败");
  }
};

// 编辑Template
const handleEdit = (template: any) => {
  editingId.value = template.id;
  formData.value.key_name = template.key_name;
  formData.value.data = template.data;
  showEditForm.value = true;
};

// 更新Template
const handleUpdate = async () => {
  if (!editingId.value) return;

  if (!formData.value.key_name.trim() || !formData.value.data.trim()) {
    notificationService.warning("请输入完整的Template信息");
    return;
  }

  const requestData: UpdateTemplateRequest = {
    key_name: formData.value.key_name,
    data: formData.value.data,
  };

  const result = await templateStore.updateExistingTemplate(
    editingId.value,
    requestData,
  );
  if (result) {
    notificationService.success("Template更新成功");
    showEditForm.value = false;
    resetForm();
  } else {
    notificationService.error("Template更新失败");
  }
};

// 删除Template
const handleDelete = async (id: number) => {
  if (!confirm("确定要删除这个Template吗？")) return;

  const success = await templateStore.deleteExistingTemplate(id);
  if (success) {
    notificationService.success("Template删除成功");
  } else {
    notificationService.error("Template删除失败");
  }
};

// 重置表单
const resetForm = () => {
  formData.value = {
    key_name: "",
    data: "",
  };
  editingId.value = null;
};

// 取消操作
const handleCancel = () => {
  showCreateForm.value = false;
  showEditForm.value = false;
  resetForm();
};

// 切换Template展开/折叠状态
const toggleExpand = (templateId: number) => {
  if (expandedTemplates.value.has(templateId)) {
    expandedTemplates.value.delete(templateId);
  } else {
    expandedTemplates.value.add(templateId);
  }
};

// 检查Template是否展开
const isExpanded = (templateId: number) => {
  return expandedTemplates.value.has(templateId);
};

// 格式化时间
const formatTime = (timeString: string) => {
  return new Date(timeString).toLocaleString("zh-CN");
};

// 组件挂载时加载数据
onMounted(() => {
  loadData();
});
</script>
<script lang="ts">
import { defineComponent } from "vue";
import { useUserStore } from "@/stores/user.ts";

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
<template>
  <div class="template-container">
    <!-- 头部 -->
    <div class="header">
      <h1>HTML模板管理</h1>
      <button
        class="button primary"
        @click="showCreateForm = true"
        :disabled="templateStore.loading"
      >
        {{ templateStore.loading ? "加载中..." : "新建Template" }}
      </button>
    </div>

    <!-- 错误提示 -->
    <div v-if="templateStore.error" class="error-message">
      {{ templateStore.error }}
      <button @click="templateStore.clearError()" class="close-btn">×</button>
    </div>

    <!-- 加载状态 -->
    <div v-if="templateStore.loading" class="loading">加载中...</div>

    <!-- Template列表 -->
    <div v-else class="template-list">
      <div v-if="!templateStore.hasTemplates" class="empty-state">
        暂无Template数据
      </div>

      <div
        v-for="template in templateStore.templates"
        :key="template.id"
        class="template-item"
      >
        <div class="template-header">
          <h3>{{ template.key_name }}</h3>
          <div class="template-actions">
            <button
              class="button secondary small"
              @click="handleEdit(template)"
            >
              编辑
            </button>
            <button
              class="button danger small"
              @click="handleDelete(template.id)"
            >
              删除
            </button>
          </div>
        </div>
        <div class="template-content">
          <div class="content-display" @click="toggleExpand(template.id)">
            <highlightjs
              v-if="isExpanded(template.id)"
              language="xml"
              :code="template.data"
            />
            <highlightjs
              v-else
              language="xml"
              :code="getTruncatedData(template.data)"
            />
            <div class="expand-indicator">
              {{ isExpanded(template.id) ? "收起 ↑" : "展开 ↓" }}
            </div>
          </div>
        </div>
        <div class="template-meta">
          <span>创建时间: {{ formatTime(template.created_at) }}</span>
          <span>更新时间: {{ formatTime(template.updated_at) }}</span>
        </div>
      </div>
    </div>

    <!-- 创建Template模态框 -->
    <div v-if="showCreateForm" class="modal-overlay" @click="handleCancel">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>新建HTML模板</h2>
          <button class="close-btn" @click="handleCancel">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Key Name:</label>
            <input
              v-model="formData.key_name"
              type="text"
              placeholder="请输入Key Name"
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label>HTML模板内容:</label>
            <textarea
              v-model="formData.data"
              placeholder="请输入HTML模板内容"
              class="form-textarea"
              rows="15"
            ></textarea>
          </div>
        </div>
        <div class="modal-footer">
          <button class="button secondary" @click="handleCancel">取消</button>
          <button class="button primary" @click="handleCreate">创建</button>
        </div>
      </div>
    </div>

    <!-- 编辑Template模态框 -->
    <div v-if="showEditForm" class="modal-overlay" @click="handleCancel">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>编辑HTML模板</h2>
          <button class="close-btn" @click="handleCancel">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Key Name:</label>
            <input
              v-model="formData.key_name"
              type="text"
              placeholder="请输入Key Name"
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label>HTML模板内容:</label>
            <textarea
              v-model="formData.data"
              placeholder="请输入HTML模板内容"
              class="form-textarea"
              rows="15"
            ></textarea>
          </div>
        </div>
        <div class="modal-footer">
          <button class="button secondary" @click="handleCancel">取消</button>
          <button class="button primary" @click="handleUpdate">更新</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.template-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h1 {
  margin: 0;
  color: var(--color-heading);
}

.error-message {
  background-color: #fee;
  border: 1px solid #fcc;
  color: #c33;
  padding: 10px;
  border-radius: 4px;
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: #c33;
}

.loading {
  text-align: center;
  padding: 40px;
  color: var(--color-text);
}

.template-list {
  display: grid;
  gap: 20px;
}

.empty-state {
  text-align: center;
  padding: 60px;
  color: var(--color-text);
  font-size: 18px;
}

.template-item {
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 20px;
  background: var(--color-background-soft);
}

.template-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.template-header h3 {
  margin: 0;
  color: var(--color-heading);
}

.template-actions {
  display: flex;
  gap: 10px;
}

.template-content {
  margin-bottom: 15px;
}

.content-display {
  cursor: pointer;
  position: relative;
  transition: all 0.3s ease;
  color: gray;
}

.content-display:hover {
  background-color: #f0f0f0;
}

.template-content pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  background: #f5f5f5;
  padding: 15px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 14px;
  line-height: 1.5;
  margin: 0;
}

.expand-indicator {
  text-align: center;
  padding: 8px;
  color: #007bff;
  font-size: 12px;
  font-weight: bold;
  background: #e9f5ff;
  border-radius: 0 0 4px 4px;
  transition: all 0.3s ease;
}

.content-display:hover .expand-indicator {
  background: #d0e8ff;
}

.template-meta {
  display: flex;
  gap: 20px;
  font-size: 12px;
  color: var(--color-text);
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  width: 90%;
  max-width: 600px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #eee;
}

.modal-header h2 {
  margin: 0;
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.form-textarea {
  resize: vertical;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 20px;
  border-top: 1px solid #eee;
}

.button {
  padding: 10px 20px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s ease;
}

.button.primary {
  background: #007bff;
  color: white;
}

.button.primary:hover {
  background: #0056b3;
}

.button.secondary {
  background: #6c757d;
  color: white;
}

.button.secondary:hover {
  background: #545b62;
}

.button.danger {
  background: #dc3545;
  color: white;
}

.button.danger:hover {
  background: #c82333;
}

.button.small {
  padding: 5px 10px;
  font-size: 12px;
}

.button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .template-container {
    padding: 10px;
  }

  .header {
    flex-direction: column;
    gap: 15px;
    align-items: stretch;
  }

  .template-header {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }

  .template-actions {
    justify-content: flex-end;
  }

  .template-meta {
    flex-direction: column;
    gap: 5px;
  }
}
</style>
