import type {
  CreatePromptRequest,
  Prompt,
  ResponseData,
  UpdatePromptRequest,
} from "@/types/types.ts";
import {
  deleteWithAuth,
  getWithAuth,
  postWithAuth,
  putWithAuth,
} from "@/util/authInterceptor.ts";

const API_PROMPT_BASE_URL = "/api/prompt";

// 获取所有Prompts
export const fetchAllPrompts = async (): Promise<Prompt[] | null> => {
  try {
    const res = await getWithAuth(API_PROMPT_BASE_URL);
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error("获取Prompt列表失败:", e);
    return null;
  }
};

// 根据ID获取Prompt
export const fetchPromptById = async (id: number): Promise<Prompt | null> => {
  try {
    const res = await getWithAuth(`${API_PROMPT_BASE_URL}/${id}`);
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error(`获取Prompt ${id} 失败:`, e);
    return null;
  }
};

// 根据Keyname获取Prompt
export const fetchPromptByKeyname = async (
  keyname: string,
): Promise<Prompt | null> => {
  try {
    const res = await getWithAuth(`${API_PROMPT_BASE_URL}/keyname/${keyname}`);
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error(`获取Prompt ${keyname} 失败:`, e);
    return null;
  }
};

// 创建Prompt
export const createPrompt = async (
  promptData: CreatePromptRequest,
): Promise<Prompt | null> => {
  try {
    const res = await postWithAuth(API_PROMPT_BASE_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(promptData),
    });
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error("创建Prompt失败:", e);
    return null;
  }
};

// 更新Prompt
export const updatePrompt = async (
  id: number,
  promptData: UpdatePromptRequest,
): Promise<Prompt | null> => {
  try {
    const res = await putWithAuth(`${API_PROMPT_BASE_URL}/${id}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(promptData),
    });
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error(`更新Prompt ${id} 失败:`, e);
    return null;
  }
};

// 删除Prompt
export const deletePrompt = async (id: number): Promise<boolean> => {
  try {
    const res = await deleteWithAuth(`${API_PROMPT_BASE_URL}/${id}`, {
      method: "DELETE",
    });
    return res.ok;
  } catch (e) {
    console.error(`删除Prompt ${id} 失败:`, e);
    return false;
  }
};
