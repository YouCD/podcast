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
import type {changeTitleReq, messageInfo, msgRequest, sessionItem, sessionResponse, PlanData, StepInfo,} from "@/types/types.ts";

// 解析后的消息类型（plan 和 steps 为对象）
export interface ParsedMessage {
    id?: number;
    created_at?: string;
    session_id: string;
    role: string;
    content?: string;
    uuid: string;
    reasoning_content: string;
    showReasoningContent?: boolean;
    reasoning_typing?: boolean;
    typing?: boolean;
    loading?: boolean;
    // 解析后为对象
    plan?: PlanData;
    steps?: StepInfo[];
    think_content?: string;
    message_type?: 'normal' | 'plan' | 'step';
}

// 解析后的会话响应类型
export interface ParsedSessionResponse {
    messages: ParsedMessage[];
    session: sessionItem;
}

interface State {
    sessionList: sessionItem[];
    sessionData: ParsedSessionResponse | null;
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
            
            // 解析消息中的 JSON 字符串字段
            const parsedMessages: ParsedMessage[] = resp.messages.map((message) => {
                // 解析 plan 字段（如果是 JSON 字符串）
                let plan: PlanData | undefined = undefined;
                if (typeof message.plan === 'string' && message.plan && message.plan.trim() !== '') {
                    try {
                        plan = JSON.parse(message.plan) as PlanData;
                    } catch (e) {
                        console.error('Failed to parse plan:', e);
                        plan = undefined;
                    }
                }
                
                // 解析 steps 字段（如果是 JSON 字符串）
                let steps: StepInfo[] | undefined = undefined;
                const stepsRaw = message.steps;
                console.log('Raw steps for message:', message.uuid, 'type:', typeof stepsRaw, 'value:', stepsRaw);
                
                if (typeof stepsRaw === 'string' && stepsRaw) {
                    const trimmed = stepsRaw.trim();
                    console.log('Steps is string, trimmed length:', trimmed.length, 'first 50 chars:', trimmed.substring(0, 50));
                    
                    if (trimmed !== '') {
                        try {
                            const parsed = JSON.parse(trimmed);
                            console.log('JSON.parse succeeded, type:', typeof parsed, 'isArray:', Array.isArray(parsed));
                            if (Array.isArray(parsed)) {
                                steps = parsed as StepInfo[];
                                console.log('Parsed steps for message:', message.uuid, 'steps count:', steps.length);
                                if (steps.length > 0 && steps[0]) {
                                    console.log('First step has result:', !!steps[0].result);
                                }
                            } else {
                                console.error('Parsed steps is not an array:', typeof parsed);
                            }
                        } catch (e) {
                            console.error('Failed to parse steps:', e);
                            steps = undefined;
                        }
                    } else {
                        console.log('Steps string is empty after trim');
                    }
                } else {
                    console.log('Steps is not a string or is falsy:', typeof stepsRaw, !!stepsRaw);
                }
                
                console.log('Final parsed message:', message.uuid, {
                    hasPlan: !!plan,
                    hasSteps: !!steps,
                    stepsCount: steps ? steps.length : 0
                });
                
                return {
                    id: message.id,
                    created_at: message.created_at,
                    session_id: message.session_id,
                    role: message.role,
                    content: message.content,
                    uuid: message.uuid,
                    reasoning_content: message.reasoning_content,
                    think_content: message.think_content,
                    message_type: message.message_type,
                    // 解析后的对象
                    plan: plan,
                    steps: steps,
                    // UI 状态
                    showReasoningContent: false,
                    typing: false,
                } as ParsedMessage;
            });
            
            state.value.sessionData = {
                messages: parsedMessages,
                session: resp.session,
            };
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
    
    // 新增：设置计划数据
    const setPlan = (uuid: string, plan: any) => {
        if (state.value.sessionData) {
            const message = state.value.sessionData.messages.find((m) => m.uuid === uuid);
            if (message) {
                message.loading = false;
                message.plan = plan;
            }
        }
    };
    
    // 新增：添加或更新步骤
    const setStep = (uuid: string, step: any) => {
        if (state.value.sessionData) {
            const message = state.value.sessionData.messages.find((m) => m.uuid === uuid);
            if (message) {
                if (!message.steps) {
                    message.steps = [];
                }
                const steps = message.steps;
                const existingIndex = steps.findIndex((s: any) => s.step_id === step.step_id);
                if (existingIndex >= 0) {
                    // 更新现有步骤
                    steps[existingIndex] = { ...steps[existingIndex], ...step };
                } else {
                    // 添加新步骤
                    steps.push({ ...step, expanded: false });
                }
                console.log('setStep: steps updated', JSON.stringify(message.steps));
            }
        }
    };
    
    // 新增：更新步骤结果
    const setStepResult = (uuid: string, stepId: number, result: string, status: 'pending' | 'running' | 'completed' | 'failed', toolArgs?: any) => {
        console.log('setStepResult called:', { uuid, stepId, result: result?.substring(0, 100), status, toolArgs });
        if (state.value.sessionData) {
            const message = state.value.sessionData.messages.find((m) => m.uuid === uuid);
            console.log('setStepResult: message found?', !!message);
            console.log('setStepResult: message.steps?', message?.steps ? JSON.stringify(message.steps).substring(0, 200) : 'undefined');
            if (message && message.steps) {
                const steps = message.steps;
                const step = steps.find((s: any) => s.step_id === stepId);
                console.log('setStepResult: step found?', !!step, 'step_id:', stepId);
                if (step) {
                    step.result = result;
                    step.status = status;
                    if (toolArgs) {
                        step.tool_args = toolArgs;
                    }
                    console.log('setStepResult: step updated', JSON.stringify(step).substring(0, 200));
                }
            }
        }
    };
    
    // 新增：设置思考内容
    const setThinkContent = (uuid: string, content: string) => {
        if (state.value.sessionData) {
            const message = state.value.sessionData.messages.find((m) => m.uuid === uuid);
            if (message) {
                if (!message.think_content) {
                    message.think_content = '';
                }
                message.think_content += content;
            }
        }
    };
    
    // 新增：切换步骤展开状态
    const toggleStepExpand = (uuid: string, stepId: number) => {
        if (state.value.sessionData) {
            const message = state.value.sessionData.messages.find((m) => m.uuid === uuid);
            if (message && message.steps) {
                const steps = message.steps;
                const step = steps.find((s: any) => s.step_id === stepId);
                if (step) {
                    step.expanded = !step.expanded;
                }
            }
        }
    };
    
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
        // 新增方法
        setPlan,
        setStep,
        setStepResult,
        setThinkContent,
        toggleStepExpand,
    };
});
