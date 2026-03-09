import type {NextLlmHtml, ResponseData, Rss, status, TimeStay,} from "@/types/types.ts";
import {getWithAuth, postWithAuth} from "@/util/authInterceptor.ts";

const API_FEED_BASE_URL = "/api/feed";

export const fetchCategories = async (): Promise<string[]> => {
    try {
        const res = await getWithAuth(`${API_FEED_BASE_URL}/categories`);
        let data: ResponseData = (await res.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error("获取分类失败:", e);
        return [];
    }
};

export const fetchCategoryRss = async (
    category: string,
): Promise<Rss[] | null> => {
    try {
        const res = await getWithAuth(
            `${API_FEED_BASE_URL}/categories/${category}/24h`,
        );
        let data: ResponseData = (await res.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error(`获取 ${category} 失败:`, e);
        return null;
    }
};

export const fetchLlmHtml = async (
    id: number,
    notRead: boolean | null,
): Promise<NextLlmHtml | undefined> => {
    try {
        let url = `${API_FEED_BASE_URL}/${id}/llm_html`;
        if (notRead) {
            url += "?not_read=true";
        }
        const res = await getWithAuth(url);
        let data: ResponseData = (await res.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error(`获取 失败:`, e);
    }
};

export const sendTimeStay = async (data: TimeStay): Promise<boolean> => {
    try {
        const res = await postWithAuth(`${API_FEED_BASE_URL}/time_stay`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(data),
        });
        return res.ok;
    } catch (e) {
        console.error("发送停留时间失败:", e);
        return false;
    }
};

export const fetchRead24hRss = async (): Promise<Rss[]> => {
    try {
        const res = await getWithAuth(`${API_FEED_BASE_URL}/read24h`);
        let data: ResponseData = (await res.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error(`获取失败:`, e);
        return [];
    }
};
export const fetchRssStatus = async (): Promise<status | undefined> => {
    try {
        const res = await getWithAuth(`${API_FEED_BASE_URL}/status`);
        let data: ResponseData = (await res.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error(`获取失败:`, e);
        return;
    }
};
export const fetchNotRead = async (): Promise<Rss[]> => {
    try {
        const res = await getWithAuth(`${API_FEED_BASE_URL}/not_read`);
        let data: ResponseData = (await res.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error(`获取失败:`, e);
        return [];
    }
};
