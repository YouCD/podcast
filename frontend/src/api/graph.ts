import type {DgraphResponse, ResponseData} from "@/types/types.ts";
import {getWithAuth} from "@/util/authInterceptor.ts";

const API_GRAPH_BASE_URL = "/api/graph";

// 根据ID获取聊天记录
export const getDgraph = async (): Promise<DgraphResponse> => {
    try {
        const response = await getWithAuth(`${API_GRAPH_BASE_URL}/`);

        if (!response.ok) {
            throw new Error(`获取聊天记录失败: ${response.status}`);
        }
        let data: ResponseData = (await response.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error(`获取失败:`, e);
        throw e;
    }
};
