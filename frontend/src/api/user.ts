import type { ResponseData, tokenInfo, user } from "@/types/types.ts";

const API_USER_BASE_URL = "/api/user";

export const userLogin = async (data: user): Promise<tokenInfo> => {
  try {
    const res = await fetch(`${API_USER_BASE_URL}/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    });
    let d: ResponseData = (await res.json()) as ResponseData;
    return d.data;
  } catch (e) {
    console.error("登录失败:", e);
    return { success: false, message: "登录失败", token: "" };
  }
};
