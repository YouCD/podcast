import {createRouter, createWebHashHistory} from "vue-router";
import HomeView from "../views/HomeView.vue";
import LLMHtmlView from "@/views/LLMHtmlView.vue";
import ReadListView from "@/views/ReadListView.vue";
import LLMReportView from "@/views/LLMReportView.vue";
import ChatView from "@/views/ChatView.vue";
import PromptView from "@/views/PromptView.vue";
import TemplateView from "@/views/TemplateView.vue";

const router = createRouter({
    history: createWebHashHistory(),
    routes: [
        {
            path: "/",
            name: "home",
            component: HomeView,
        },
        {
            path: "/llm_html/:id?",
            name: "llm_html",
            component: LLMHtmlView,
        },
        {
            path: "/read_list",
            name: "read_list",
            component: ReadListView,
        },
        {
            path: "/llm_report",
            name: "llm_report",
            component: LLMReportView,
        },
        {
            path: "/chat",
            name: "chat",
            component: ChatView,
        },
        {
            path: "/prompt",
            name: "prompt",
            component: PromptView,
        },
        {
            path: "/template",
            name: "template",
            component: TemplateView,
        },
    ],
});

export default router;
