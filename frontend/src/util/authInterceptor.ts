// 自定义请求函数，自动添加认证头
import notificationService from "@/components/Notification/notificationService.ts";
import router from "@/router";
import { useUserStore } from "@/stores/user.ts";

export const authenticatedFetch = async (
  url: string,
  options: RequestInit = {},
): Promise<Response> => {
  // 获取存储的令牌
  const token = localStorage.getItem("authToken");

  // 设置默认选项
  const defaultOptions: RequestInit = {
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  };

  // 合并选项
  const mergedOptions: RequestInit = {
    ...defaultOptions,
    ...options,
    headers: {
      ...defaultOptions.headers,
      ...options.headers,
    },
  };

  try {
    const response = await fetch(url, mergedOptions);

    // 检查是否返回了认证错误
    if (response.status === 401) {
      // 清除无效的令牌
      localStorage.removeItem("authToken");
      notificationService.error("请先登录后再使用此功能");
      const userStore = useUserStore();
      userStore.initializeAuth();
      await router.push("/");
      // 可以在这里触发登出逻辑
      console.warn("认证失败，令牌可能已过期");
    }

    return response;
  } catch (error) {
    console.error("请求失败:", error);
    throw error;
  }
};

// 便捷函数，用于发送带认证的GET请求
export const getWithAuth = async (url: string, options: RequestInit = {}) => {
  return authenticatedFetch(url, { ...options, method: "GET" });
};

// 便捷函数，用于发送带认证的POST请求
export const postWithAuth = async (url: string, options: RequestInit = {}) => {
  return authenticatedFetch(url, { ...options, method: "POST" });
};

// 便捷函数，用于发送带认证的PUT请求
export const putWithAuth = async (url: string, options: RequestInit = {}) => {
  return authenticatedFetch(url, { ...options, method: "PUT" });
};

// 便捷函数，用于发送带认证的DELETE请求
export const deleteWithAuth = async (
  url: string,
  options: RequestInit = {},
) => {
  return authenticatedFetch(url, { ...options, method: "DELETE" });
};
