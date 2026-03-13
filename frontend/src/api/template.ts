import type {
  CreateTemplateRequest,
  ResponseData,
  Template,
  UpdateTemplateRequest,
} from "@/types/types.ts";
import {
  deleteWithAuth,
  getWithAuth,
  postWithAuth,
  putWithAuth,
} from "@/util/authInterceptor.ts";

const API_TEMPLATE_BASE_URL = "/api/template";

// 获取所有Templates
export const fetchAllTemplates = async (): Promise<Template[] | null> => {
  try {
    const res = await getWithAuth(API_TEMPLATE_BASE_URL);
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error("获取Template列表失败:", e);
    return null;
  }
};

// 根据ID获取Template
export const fetchTemplateById = async (
  id: number,
): Promise<Template | null> => {
  try {
    const res = await getWithAuth(`${API_TEMPLATE_BASE_URL}/${id}`);
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error(`获取Template ${id} 失败:`, e);
    return null;
  }
};

// 根据Keyname获取Template
export const fetchTemplateByKeyname = async (
  keyname: string,
): Promise<Template | null> => {
  try {
    const res = await getWithAuth(
      `${API_TEMPLATE_BASE_URL}/keyname/${keyname}`,
    );
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error(`获取Template ${keyname} 失败:`, e);
    return null;
  }
};

// 创建Template
export const createTemplate = async (
  templateData: CreateTemplateRequest,
): Promise<Template | null> => {
  try {
    const res = await postWithAuth(API_TEMPLATE_BASE_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(templateData),
    });
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error("创建Template失败:", e);
    return null;
  }
};

// 更新Template
export const updateTemplate = async (
  id: number,
  templateData: UpdateTemplateRequest,
): Promise<Template | null> => {
  try {
    const res = await putWithAuth(`${API_TEMPLATE_BASE_URL}/${id}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(templateData),
    });
    const data: ResponseData = (await res.json()) as ResponseData;
    return data.data;
  } catch (e) {
    console.error(`更新Template ${id} 失败:`, e);
    return null;
  }
};

// 删除Template
export const deleteTemplate = async (id: number): Promise<boolean> => {
  try {
    const res = await deleteWithAuth(`${API_TEMPLATE_BASE_URL}/${id}`, {
      method: "DELETE",
    });
    return res.ok;
  } catch (e) {
    console.error(`删除Template ${id} 失败:`, e);
    return false;
  }
};
