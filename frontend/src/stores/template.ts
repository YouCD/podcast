import { defineStore } from "pinia";
import { computed, ref } from "vue";
import type {
  CreateTemplateRequest,
  Template,
  UpdateTemplateRequest,
} from "@/types/types.ts";
import {
  createTemplate,
  deleteTemplate,
  fetchAllTemplates,
  fetchTemplateById,
  fetchTemplateByKeyname,
  updateTemplate,
} from "@/api/template.ts";

export const useTemplateStore = defineStore("template", () => {
  // 状态
  const templates = ref<Template[]>([]);
  const currentTemplate = ref<Template | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // 计算属性
  const templateCount = computed(() => templates.value.length);
  const hasTemplates = computed(() => templates.value.length > 0);

  // Actions
  // 加载所有Templates
  const loadTemplates = async (): Promise<boolean> => {
    if (loading.value) return false;

    loading.value = true;
    error.value = null;

    try {
      const data = await fetchAllTemplates();
      if (data) {
        templates.value = data;
        return true;
      }
      return false;
    } catch (err) {
      error.value = "加载Template列表失败";
      console.error("加载Template列表失败:", err);
      return false;
    } finally {
      loading.value = false;
    }
  };

  // 根据ID获取Template
  const getTemplateById = async (id: number): Promise<Template | null> => {
    try {
      const template = await fetchTemplateById(id);
      if (template) {
        currentTemplate.value = template;
        // 如果该Template不在列表中，则添加到列表
        const existingIndex = templates.value.findIndex((p) => p.id === id);
        if (existingIndex === -1) {
          templates.value.push(template);
        } else {
          templates.value[existingIndex] = template;
        }
      }
      return template;
    } catch (err) {
      error.value = "获取Template失败";
      console.error("获取Template失败:", err);
      return null;
    }
  };

  // 根据Keyname获取Template
  const getTemplateByKeyname = async (
    keyname: string,
  ): Promise<Template | null> => {
    try {
      return await fetchTemplateByKeyname(keyname);
    } catch (err) {
      error.value = "获取Template失败";
      console.error("获取Template失败:", err);
      return null;
    }
  };

  // 创建Template
  const createNewTemplate = async (
    templateData: CreateTemplateRequest,
  ): Promise<Template | null> => {
    loading.value = true;
    error.value = null;

    try {
      const newTemplate = await createTemplate(templateData);
      if (newTemplate) {
        templates.value.push(newTemplate);
        currentTemplate.value = newTemplate;
      }
      return newTemplate;
    } catch (err) {
      error.value = "创建Template失败";
      console.error("创建Template失败:", err);
      return null;
    } finally {
      loading.value = false;
    }
  };

  // 更新Template
  const updateExistingTemplate = async (
    id: number,
    templateData: UpdateTemplateRequest,
  ): Promise<Template | null> => {
    loading.value = true;
    error.value = null;

    try {
      const updatedTemplate = await updateTemplate(id, templateData);
      if (updatedTemplate) {
        const index = templates.value.findIndex((p) => p.id === id);
        if (index !== -1) {
          templates.value[index] = updatedTemplate;
        }
        if (currentTemplate.value && currentTemplate.value.id === id) {
          currentTemplate.value = updatedTemplate;
        }
      }
      return updatedTemplate;
    } catch (err) {
      error.value = "更新Template失败";
      console.error("更新Template失败:", err);
      return null;
    } finally {
      loading.value = false;
    }
  };

  // 删除Template
  const deleteExistingTemplate = async (id: number): Promise<boolean> => {
    loading.value = true;
    error.value = null;

    try {
      const success = await deleteTemplate(id);
      if (success) {
        templates.value = templates.value.filter((p) => p.id !== id);
        if (currentTemplate.value && currentTemplate.value.id === id) {
          currentTemplate.value = null;
        }
      }
      return success;
    } catch (err) {
      error.value = "删除Template失败";
      console.error("删除Template失败:", err);
      return false;
    } finally {
      loading.value = false;
    }
  };

  // 清除当前Template
  const clearCurrentTemplate = () => {
    currentTemplate.value = null;
  };

  // 清除错误
  const clearError = () => {
    error.value = null;
  };

  // 重置状态
  const reset = () => {
    templates.value = [];
    currentTemplate.value = null;
    loading.value = false;
    error.value = null;
  };

  return {
    // 状态
    templates,
    currentTemplate,
    loading,
    error,

    // 计算属性
    templateCount,
    hasTemplates,

    // Actions
    loadTemplates,
    getTemplateById,
    getTemplateByKeyname,
    createNewTemplate,
    updateExistingTemplate,
    deleteExistingTemplate,
    clearCurrentTemplate,
    clearError,
    reset,
  };
});
