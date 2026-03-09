import {defineStore} from "pinia";
import {computed, ref} from "vue";
import {jwtDecode} from "jwt-decode";
import type {TokenPayload} from "@/types/types.ts";

export const useUserStore = defineStore("user", () => {
    // 存储认证令牌
    const authToken = ref<string | null>(localStorage.getItem("authToken"));

    // 存储解码后的令牌信息
    const tokenPayload = ref<TokenPayload | null>(null);

    // 检查令牌是否存在的计算属性
    const isAuthenticated = computed(() => {
        return authToken.value !== null && authToken.value !== undefined;
    });

    // 检查令牌是否过期的计算属性
    const isTokenExpired = computed(() => {
        if (!tokenPayload.value || !tokenPayload.value.exp) {
            return true; // 如果没有有效载荷或没有过期时间，则认为已过期
        }

        const currentTime = Math.floor(Date.now() / 1000); // 当前时间戳（秒）
        return tokenPayload.value.exp < currentTime;
    });

    // 综合认证状态：既有令牌又未过期
    const isValidAuth = computed(() => {
        return isAuthenticated.value && !isTokenExpired.value;
    });

    // 更新令牌
    const setAuthToken = (token: string | null) => {
        if (token) {
            localStorage.setItem("authToken", token);
            authToken.value = token;
            console.log("Token updated:", token);
            console.log("isAuthenticated:", isAuthenticated.value);
            console.log("isValidAuth:", isValidAuth.value);
            // 解码新令牌
            try {
                tokenPayload.value = jwtDecode<TokenPayload>(token);
                console.log("B Token updated:", token);
                console.log("B isAuthenticated:", isAuthenticated.value);
                console.log("B isValidAuth:", isValidAuth.value);
            } catch (error) {
                console.error("Error decoding token:", error);
                tokenPayload.value = null;
            }
        } else {
            // 清除令牌
            localStorage.removeItem("authToken");
            authToken.value = null;
            tokenPayload.value = null;
        }
    };

    // 手动检查并更新令牌状态（例如页面加载时调用）
    const initializeAuth = () => {
        const storedToken = localStorage.getItem("authToken");
        if (storedToken) {
            setAuthToken(storedToken);
        } else {
            setAuthToken(null);
        }
    };

    // 刷新令牌（如果令牌来自API响应）
    const refreshToken = (newToken: string) => {
        setAuthToken(newToken);
    };

    // 登出
    const logout = () => {
        setAuthToken(null);
    };

    // 初始化时尝试解析现有令牌
    if (authToken.value) {
        try {
            tokenPayload.value = jwtDecode<TokenPayload>(authToken.value);
        } catch (error) {
            console.error("Error decoding token on initialization:", error);
            tokenPayload.value = null;
            localStorage.removeItem("authToken");
            authToken.value = null;
        }
    }

    return {
        authToken,
        tokenPayload,
        isAuthenticated,
        isTokenExpired,
        isValidAuth,
        setAuthToken,
        initializeAuth,
        refreshToken,
        logout,
    };
});
