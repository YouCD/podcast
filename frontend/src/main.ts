import "./assets/base.css";

import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import router from "./router";
import "./assets/iconfont/iconfont.css";

//  语法高亮
import "highlight.js/styles/stackoverflow-light.css";
import hljs from "highlight.js/lib/core";
import markdown from "highlight.js/lib/languages/markdown";
import xml from "highlight.js/lib/languages/xml";
import hljsVuePlugin from "@highlightjs/vue-plugin";

hljs.registerLanguage("markdown", markdown);
hljs.registerLanguage("xml", xml);

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(hljsVuePlugin);

app.mount("#app");
