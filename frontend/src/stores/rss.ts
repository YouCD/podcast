import {defineStore} from "pinia";
import {computed, ref} from "vue";
import type {Rss} from "@/types/types.ts";
import {fetchCategories, fetchCategoryRss, fetchNotRead} from "@/api/rss.ts";

const useCategoryStore = defineStore("category", () => {
    /* 状态 */
    const categoryMap = ref(new Map<string, Rss[]>()); // 目标数据结构
    const loaded = ref(false); // 是否已加载过
    const loading = ref(false); // 加载中
    const notRead = ref(false); // 未读标志

    /* 只读计算属性：外部直接当 Map 用 */
    const rssMap = computed(() => categoryMap.value);
    const hasRss = ref(false);

    /* 核心动作：一次性拉全量数据 */
    async function loadAll() {
        if (loaded.value || loading.value) return;
        loading.value = true;
        try {
            const cats = await fetchCategories(); // 1. 先拿所有分类
            const tasks = cats.map(async (c) => {
                const list = await fetchCategoryRss(c); // 2. 并发拿每类 RSS
                if (list) {
                    if (list.length > 0) {
                        hasRss.value = true;
                    }
                    categoryMap.value.set(c, list);
                }
            });
            await Promise.all(tasks);
            loaded.value = true;
        } finally {
            loading.value = false;
        }
    }

    /* 按需刷新单个分类 */
    async function refreshCategory(c: string) {
        const list = await fetchCategoryRss(c);
        if (!list) return;
        categoryMap.value.set(c, list);
    }

    /* 删除指定分类下的某一条 RSS */
    function removeRssItem(category: string, id: number) {
        const list = categoryMap.value.get(category) ?? [];
        const filtered = list.filter((rss) => rss.id != id); // 用 link 当唯一 id
        if (filtered.length !== list.length) {
            categoryMap.value.set(category, filtered); // 重新写入，触发响应式
        }
    }

    async function notReadHandler() {
        let rss = await fetchNotRead();
        loading.value = true;
        try {
            rss.forEach((rss) => {
                let rssList = categoryMap.value.get(rss.categories);
                if (rssList) {
                    rssList.push(rss);
                    categoryMap.value.set(rss.categories, rssList);
                }
            });
            loaded.value = true;
            hasRss.value = true;
            notRead.value = true;
        } finally {
            loading.value = false;
        }
    }

    return {
        rssMap,
        loaded,
        loading,
        loadAll,
        refreshCategory,
        removeRssItem,
        hasRss,
        notRead,
        notReadHandler,
    };
});
export default useCategoryStore;
