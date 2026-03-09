// stores/chat_records.ts
import {defineStore} from "pinia";
import {computed, ref} from "vue";
import {
    changeTitleHandler,
    createNewSession,
    deleteChatRecord,
    getChatRecordBySessionId,
    getChatRecordsByUserId,
    sendMsg,
} from "@/api/chat.ts";
import type {changeTitleReq, messageInfo, msgRequest, sessionItem, sessionResponse,} from "@/types/types.ts";

interface State {
    sessionList: sessionItem[];
    sessionData: sessionResponse | null;
    loading: boolean;
    error: string | null;
}

export const useChatRecordsStore = defineStore("chatRecords", () => {
    const state = ref<State>({
        sessionList: [],
        sessionData: null,
        loading: false,
        error: null,
    });
    // ✅ 使用 computed 保持响应性
    const sessionList = computed(() => state.value.sessionList);
    const sessionData = computed(() => state.value.sessionData);
    const loading = computed(() => state.value.loading);
    const error = computed(() => state.value.error);

    // Getters
    const getAllChatRecords = computed(() => state.value.sessionList);
    const isLoading = computed(() => state.value.loading);
    const getError = computed(() => state.value.error);

    // Actions
    const setLoading = (loading: boolean) => {
        state.value.loading = loading;
    };

    const setError = (error: string | null) => {
        state.value.error = error;
    };

    // 根据会话ID获取聊天记录
    const fetchSessionDataBySessionId = async (sessionId: string) => {
        try {
            setLoading(true);
            setError(null);

            const resp = await getChatRecordBySessionId(sessionId);
            state.value.sessionData = resp;
            state.value.sessionData!.session = resp.session;
            state.value.sessionData!.messages = resp.messages.map((message) => {
                return {
                    ...message,
                    showReasoningContent: false,
                    typing: false,
                }
            });
            return resp;
        } catch (err: any) {
            setError(err.message || "获取聊天记录失败");
            throw err;
        } finally {
            setLoading(false);
        }
    };
    const clearMsg = () => {
        if (state.value.sessionData) state.value.sessionData.messages = []
    }
    const SetReasoningTyping = (uuid: string, show: boolean) => {
        if (state.value.sessionData) state.value.sessionData.messages.forEach((message) => {
            if (message.uuid === uuid) {
                console.log("toggleReasoningContent", message.showReasoningContent)
                message.reasoning_typing = show;
            }
        })
    }
    const setMsg = async (sessionId: string, role: string, uuid: string, msg?: string, reasoning_content?: string, loading?: boolean) => {
        if (state.value.sessionData && role === "user") {
            let mm = {
                role: role,
                content: msg || "",
                session_id: sessionId,
                uuid: uuid,
                reasoning_content: reasoning_content || "",
            }
            state.value.sessionData.messages.push(mm);

            await sendMessageAction(mm)

            return
        }
        if (state.value.sessionData && role === "assistant") {
            let m = state.value.sessionData.messages.find((message) => {
                return message.uuid === uuid
            })

            if (!m) {
                let mm = {
                    role: role,
                    content: msg || "",
                    session_id: sessionId,
                    uuid: uuid,
                    typing: true,
                    loading: loading,
                    reasoning_content: "",
                    showReasoningContent: false,
                    reasoning_typing: false,
                }
                state.value.sessionData.messages.push(mm);
                return
            }
            state.value.sessionData.messages.forEach((message) => {
                if (message.uuid === uuid) {
                    message.loading=false
                    if (reasoning_content) {
                        message.reasoning_typing = true;
                        message.reasoning_content += reasoning_content;
                        message.showReasoningContent = true
                    }
                    if (msg) {
                        message.typing = true;
                        message.content += msg;
                    }
                }
            })
        }
    };

    // 筛选会话
    const filterSession = async (
        sessionId: string,
    ): Promise<sessionItem | null> => {
        return (
            sessionList.value.find(
                (record): record is sessionItem => record.session_id === sessionId,
            ) ?? null
        );
    };

    // 根据用户ID获取聊天记录列表
    const fetchSessionList = async () => {
        try {
            setLoading(true);
            setError(null);

            const chatRecords = await getChatRecordsByUserId();
            state.value.sessionList = chatRecords;

            return chatRecords;
        } catch (err: any) {
            setError(err.message || "获取聊天记录列表失败");
            throw err;
        } finally {
            setLoading(false);
        }
    };
    //  修改标题
    const changeTitleHandlerAction = async () => {
        if (!state.value.sessionData) return
        let msg: messageInfo[] = state.value.sessionData.messages.map((message) => {
            return {
                role: message.role,
                content: message.content || "",
            }
        })
        let req: changeTitleReq = {
            message: msg,
            session_id: state.value.sessionData.session.session_id,
        }
        await changeTitleHandler(req)
    };

    const createNewSessionAction = async (sessionId: string) => {
        try {
            setLoading(true);
            setError(null);

            const response = await createNewSession(sessionId);

            // 如果是创建新记录，将新记录添加到列表中
            if (response.message && response.message.includes("success")) {
                // 重新获取用户的所有聊天记录以更新列表
                await fetchSessionList();
            }
            await fetchSessionDataBySessionId(sessionId)
            return response;
        } catch (err: any) {
            setError(err.message || "创建或更新聊天记录失败");
            throw err;
        } finally {
            setLoading(false);
        }
    };

    // 删除聊天记录
    const deleteChatRecordAction = async (id: number) => {
        try {
            setLoading(true);
            setError(null);

            await deleteChatRecord(id);

            // 从本地状态移除
            state.value.sessionList = state.value.sessionList.filter(
                (record) => record.id !== id,
            );
        } catch (err: any) {
            setError(err.message || "删除聊天记录失败");
            throw err;
        } finally {
            setLoading(false);
        }
    };

    // 保存消息 todo 删除sessionId参数
    const sendMessageAction = async (messageData: msgRequest) => {
        try {
            setLoading(true);
            setError(null);

            const response = await sendMsg(messageData.session_id, messageData);
            return response;
        } catch (err: any) {
            setError(err.message || "发送RAG消息失败");
            throw err;
        } finally {
            setLoading(false);
        }
    };

    // 清空错误信息
    const clearError = () => {
        setError(null);
    };

    return {
        // State
        sessionList,
        sessionData,
        loading,
        error,

        // Getters
        getAllChatRecords,
        isLoading,
        getError,

        // Actions
        setLoading,
        setError,
        fetchSessionDataBySessionId,
        fetchSessionList,
        filterSession,
        createNewSessionAction,
        deleteChatRecordAction,
        sendMessageAction,
        clearError,
        setMsg,
        clearMsg,
        SetReasoningTyping,
        changeTitleHandlerAction,
    };
});
