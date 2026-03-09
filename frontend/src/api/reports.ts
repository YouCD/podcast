import type {Report, ResponseData} from "@/types/types.ts";
import {getWithAuth} from "@/util/authInterceptor.ts";

const API_REPORT_BASE_URL = "/api/reports";

// 获取报告列表（不包含LLMResult）
export const fetchReports = async (genre?: number): Promise<Report[]> => {
    try {
        let url = `${API_REPORT_BASE_URL}`;
        if (genre !== undefined) {
            url += `?genre=${genre}`;
        }
        const res = await getWithAuth(url);
        let data: ResponseData = (await res.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error("获取报告列表失败:", e);
        return [];
    }
};

// 根据 ID 重新生成日报失败
export const genDailyReport = async (
    id: number,
): Promise<string | undefined> => {
    try {
        const res = await getWithAuth(`${API_REPORT_BASE_URL}/${id}/daily_report`);
        if (res.status === 202) {
            // 202 状态码返回 HTML 文档，直接读取文本内容
            return await res.text();
        }
        const data = (await res.json()) as ResponseData;
        let d: { llm_result: string } = data.data;
        return d.llm_result;
    } catch (e) {
        console.error(`重新生成日报失败:`, e);
    }
};
