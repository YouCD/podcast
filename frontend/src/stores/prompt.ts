import {defineStore} from "pinia";
import {computed, ref} from "vue";
import type {CreatePromptRequest, Prompt, UpdatePromptRequest,} from "@/types/types.ts";
import {
    createPrompt,
    deletePrompt,
    fetchAllPrompts,
    fetchPromptById,
    fetchPromptByKeyname,
    updatePrompt,
} from "@/api/prompt.ts";

export const usePromptStore = defineStore("prompt", () => {
    // 状态
    const prompts = ref<Prompt[]>([]);
    const currentPrompt = ref<Prompt | null>(null);
    const loading = ref(false);
    const error = ref<string | null>(null);

    // 计算属性
    const promptCount = computed(() => prompts.value.length);
    const hasPrompts = computed(() => prompts.value.length > 0);

    // Actions
    // 加载所有Prompts
    const loadPrompts = async (): Promise<boolean> => {
        if (loading.value) return false;

        loading.value = true;
        error.value = null;

        try {
            const data = await fetchAllPrompts();
            if (data) {
                prompts.value = data;
                return true;
            }
            return false;
        } catch (err) {
            error.value = "加载Prompt列表失败";
            console.error("加载Prompt列表失败:", err);
            return false;
        } finally {
            loading.value = false;
        }
    };

    // 根据ID获取Prompt
    const getPromptById = async (id: number): Promise<Prompt | null> => {
        try {
            const prompt = await fetchPromptById(id);
            if (prompt) {
                currentPrompt.value = prompt;
                // 如果该Prompt不在列表中，则添加到列表
                const existingIndex = prompts.value.findIndex((p) => p.id === id);
                if (existingIndex === -1) {
                    prompts.value.push(prompt);
                } else {
                    prompts.value[existingIndex] = prompt;
                }
            }
            return prompt;
        } catch (err) {
            error.value = "获取Prompt失败";
            console.error("获取Prompt失败:", err);
            return null;
        }
    };

    // 根据Keyname获取Prompt
    const getPromptByKeyname = async (
        keyname: string,
    ): Promise<Prompt | null> => {
        try {
            return await fetchPromptByKeyname(keyname);
        } catch (err) {
            error.value = "获取Prompt失败";
            console.error("获取Prompt失败:", err);
            return null;
        }
    };

    // 创建Prompt
    const createNewPrompt = async (
        promptData: CreatePromptRequest,
    ): Promise<Prompt | null> => {
        loading.value = true;
        error.value = null;

        try {
            const newPrompt = await createPrompt(promptData);
            if (newPrompt) {
                prompts.value.push(newPrompt);
                currentPrompt.value = newPrompt;
            }
            return newPrompt;
        } catch (err) {
            error.value = "创建Prompt失败";
            console.error("创建Prompt失败:", err);
            return null;
        } finally {
            loading.value = false;
        }
    };

    // 更新Prompt
    const updateExistingPrompt = async (
        id: number,
        promptData: UpdatePromptRequest,
    ): Promise<Prompt | null> => {
        loading.value = true;
        error.value = null;

        try {
            const updatedPrompt = await updatePrompt(id, promptData);
            if (updatedPrompt) {
                const index = prompts.value.findIndex((p) => p.id === id);
                if (index !== -1) {
                    prompts.value[index] = updatedPrompt;
                }
                if (currentPrompt.value && currentPrompt.value.id === id) {
                    currentPrompt.value = updatedPrompt;
                }
            }
            return updatedPrompt;
        } catch (err) {
            error.value = "更新Prompt失败";
            console.error("更新Prompt失败:", err);
            return null;
        } finally {
            loading.value = false;
        }
    };

    // 删除Prompt
    const deleteExistingPrompt = async (id: number): Promise<boolean> => {
        loading.value = true;
        error.value = null;

        try {
            const success = await deletePrompt(id);
            if (success) {
                prompts.value = prompts.value.filter((p) => p.id !== id);
                if (currentPrompt.value && currentPrompt.value.id === id) {
                    currentPrompt.value = null;
                }
            }
            return success;
        } catch (err) {
            error.value = "删除Prompt失败";
            console.error("删除Prompt失败:", err);
            return false;
        } finally {
            loading.value = false;
        }
    };

    // 清除当前Prompt
    const clearCurrentPrompt = () => {
        currentPrompt.value = null;
    };

    // 清除错误
    const clearError = () => {
        error.value = null;
    };

    // 重置状态
    const reset = () => {
        prompts.value = [];
        currentPrompt.value = null;
        loading.value = false;
        error.value = null;
    };

    return {
        // 状态
        prompts,
        currentPrompt,
        loading,
        error,

        // 计算属性
        promptCount,
        hasPrompts,

        // Actions
        loadPrompts,
        getPromptById,
        getPromptByKeyname,
        createNewPrompt,
        updateExistingPrompt,
        deleteExistingPrompt,
        clearCurrentPrompt,
        clearError,
        reset,
    };
});
