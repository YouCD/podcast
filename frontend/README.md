# podcast

This template should help get you started developing with Vue 3 in Vite.

## Recommended IDE Setup

[VS Code](https://codevisualstudio.com/) + [Vue (Official)](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (and disable Vetur).

## Recommended Browser Setup

- Chromium-based browsers (Chrome, Edge, Brave, etc.):
  - [Vue.js devtools](https://chromewebstore.google.com/detail/vuejs-devtools/nhdogjmejiglipccpnnnanhbledajbpd) 
  - [Turn on Custom Object Formatter in Chrome DevTools](http://bit.ly/object-formatters)
- Firefox:
  - [Vue.js devtools](https://addons.mozilla.org/en-US/firefox/addon/vue-js-devtools/)
  - [Turn on Custom Object Formatter in Firefox DevTools](https://fxdx.dev/firefox-devtools-custom-object-formatters/)

## Type Support for `.vue` Imports in TS

TypeScript cannot handle type information for `.vue` imports by default, so we replace the `tsc` CLI with `vue-tsc` for type checking. In editors, we need [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) to make the TypeScript language service aware of `.vue` types.

## Customize configuration

See [Vite Configuration Reference](https://vite.dev/config/).

## Project Setup

```sh
npm install
```

### Compile and Hot-Reload for Development

```sh
npm run dev
```

### Type-Check, Compile and Minify for Production

```sh
npm run build
```

### Lint with [ESLint](https://eslint.org/)

```sh
npm run lint
```

## API 接口说明

项目提供了以下 API 接口用于获取社区帖子数据：

### 社区帖子接口

- `GET /api/posts` - 获取所有社区帖子
- `GET /api/posts/:id` - 根据 ID 获取特定社区帖子
- `GET /api/posts/:id/llm-html` - 根据 ID 获取特定社区帖子的 LLM HTML 内容
- `POST /api/posts` - 创建新的社区帖子
- `PUT /api/posts/:id` - 更新特定社区帖子
- `DELETE /api/posts/:id` - 删除特定社区帖子
- `GET /api/posts/categories` - 获取所有分类
- `GET /api/posts/categories/:category/24h` - 获取指定分类下最近24小时的内容

### 使用示例

要在组件中获取特定 ID 的社区帖子的 LLM HTML 内容，可以使用以下方法：

```typescript
import { getLLMHtmlById } from '@/api/post'

// 获取 ID 为 1 的帖子的 LLM HTML 内容
const fetchData = async () => {
  try {
    const response = await getLLMHtmlById(1)
    console.log(response.llm_html)
  } catch (error) {
    console.error('获取数据失败:', error)
  }
}
```