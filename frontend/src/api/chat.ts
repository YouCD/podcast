// 聊天记录相关的API接口
import {deleteWithAuth, getWithAuth, postWithAuth, putWithAuth,} from "@/util/authInterceptor.ts";
import type {
    changeTitleReq,
    changeTitleResp,
    msgRequest,
    ResponseData,
    sessionItem,
    sessionResponse,
} from "@/types/types.ts";

const API_CHAT_BASE_URL = "/api/chat";

// 根据会话ID获取聊天记录
export const getChatRecordBySessionId = async (
    sessionId: string,
): Promise<sessionResponse> => {
    try {
        const response = await getWithAuth(
            `${API_CHAT_BASE_URL}/session/${sessionId}`,
        );
        let data: ResponseData = (await response.json()) as ResponseData;
        return data.data;
    } catch (e) {
        console.error(`获取会话ID为${sessionId}的聊天记录失败:`, e);
        throw e;
    }
};

// 根据用户ID获取聊天记录列表
export const getChatRecordsByUserId = async (): Promise<sessionItem[]> => {
    try {
        const response = await getWithAuth(`${API_CHAT_BASE_URL}/user`);
        let data: ResponseData = await response.json();

        return data.data;
    } catch (e) {
        console.error("获取聊天记录列表失败:", e);
        throw e;
    }
};

export const createNewSession = async (sessionId: string): Promise<any> => {
    try {
        const response = await postWithAuth(
            `${API_CHAT_BASE_URL}/session/${sessionId}`,
            {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
            },
        );

        if (!response.ok) {
            throw new Error(`创建新聊天记录失败: ${response.status}`);
        }

        return await response.json();
    } catch (e) {
        console.error(`通过会话ID ${sessionId} 创建聊天记录失败:`, e);
        throw e;
    }
};
export const sendMsg = async (
    sessionId: string,
    msg: msgRequest,
): Promise<any> => {
    try {
        const response = await postWithAuth(
            `${API_CHAT_BASE_URL}/session/${sessionId}/send_msg`,
            {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(msg),
            },
        );

        if (!response.ok) {
            throw new Error(`创建聊天记录失败: ${response.status}`);
        }

        return await response.json();
    } catch (e) {
        console.error(`通过会话ID ${sessionId} 创建聊天记录失败:`, e);
        throw e;
    }
};

// 更新聊天记录
export const updateChatRecord = async (
    id: number,
    chatRecord: Partial<
        Omit<sessionItem, "id" | "user_id" | "created_at" | "updated_at">
    >,
): Promise<void> => {
    try {
        const response = await putWithAuth(`${API_CHAT_BASE_URL}/${id}`, {
            method: "PUT",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(chatRecord),
        });

        if (!response.ok) {
            throw new Error(`更新聊天记录失败: ${response.status}`);
        }

        // 不需要返回响应体，只需确认成功即可
        return;
    } catch (e) {
        console.error(`更新ID为${id}的聊天记录失败:`, e);
        throw e;
    }
};

// 删除聊天记录
export const deleteChatRecord = async (id: number): Promise<void> => {
    try {
        const response = await deleteWithAuth(`${API_CHAT_BASE_URL}/${id}`, {
            method: "DELETE",
        });

        if (!response.ok) {
            throw new Error(`删除聊天记录失败: ${response.status}`);
        }

        // 不需要返回响应体，只需确认成功即可
        return;
    } catch (e) {
        console.error(`删除ID为${id}的聊天记录失败:`, e);
        throw e;
    }
};

// 发送消息到后端RAG API
export const sendRagMessage = async (messageData: any) => {
    try {
        const res = await postWithAuth("/api/chat", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(messageData),
        });
        return await res.json();
    } catch (e) {
        console.error("发送RAG消息失败:", e);
        throw e;
    }
};

export const changeTitleHandler = async (
    req: changeTitleReq,
): Promise<changeTitleResp> => {
    try {
        const response = await putWithAuth(
            `${API_CHAT_BASE_URL}/session/change_title`,
            {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(req),
            },
        );
        let data: ResponseData = await response.json();
        return data.data;
    } catch (e) {
        console.error(`更新会话标题失败:`, e);
        throw e;
    }
};
