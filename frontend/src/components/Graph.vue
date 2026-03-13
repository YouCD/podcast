<template>
  <main>
    <div class="container-box shadow">
      <div class="header-container">
        <h2 class="card-title"><span>Graph</span></h2>
        <div style="padding-bottom: 10px">
          <div class="iconfont fullscreen-button" @click="toggleFullscreen">
            &#xeb99;
          </div>
        </div>
      </div>
      <VChart ref="chartRef" :style="props.style" :option="option" />
    </div>
  </main>
</template>

<script lang="ts" setup>
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { GraphChart } from "echarts/charts";
import { LegendComponent, ThumbnailComponent } from "echarts/components";
import VChart from "vue-echarts";
import {
  type CSSProperties,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
} from "vue"; // CSSProperties 可从 vue 直接导入
import type { DgraphResponse } from "@/types/types.ts";

use([ThumbnailComponent, LegendComponent, GraphChart, CanvasRenderer]);

const props = defineProps({
  dgraphResponse: {
    type: Object as () => DgraphResponse,
    required: true,
  },
  style: {
    type: Object as () => CSSProperties,
    default: () => ({}),
  },
  showThumbnail: {
    type: Boolean,
    default: true,
  },
});
const chartRef = ref();
const isFullscreen = ref(false);

// 计算 thumbnail 配置
function getThumbnail() {
  return props.showThumbnail
    ? {
        width: "15%",
        height: "15%",
        windowStyle: {
          color: "rgba(140, 212, 250, 0.5)",
          borderColor: "rgba(30, 64, 175, 0.7)",
          opacity: 1,
        },
      }
    : false;
}

const option = ref({
  backgroundColor: "white",
  legend: {
    data: [],
    orient: "vertical",
    right: 10, // ⭐贴右侧
    top: "middle", // 可改成 'top'
    itemGap: 12, // 每项间距
  },
  series: [
    {
      type: "graph",
      layout: "force",
      animation: false,
      roam: true,
      roamTrigger: "global",
      scaleLimit: {
        max: 8,
        min: 0.5,
      },
      // 节点标签
      label: {
        show: true,
        position: "right",
        formatter: "{b}",
        backgroundColor: "transparent", // 标签背景，实际好像是字的背景
      },
      draggable: true,
      data: [],
      categories: [],
      // 设置node的间距
      force: {
        edgeLength: [60, 100],
        repulsion: 300,
        gravity: 0.2,
      },
      relations: [],
      edges: [],
      // 👇 配置连线标签
      edgeLabel: {
        show: true,
        position: "middle",
        formatter: "{c}", // 使用 {b} 来显示 name 属性
        color: "#555",
        fontSize: 11,
      },
      //  连线为箭头
      edgeSymbol: ["none", "arrow"], // 👈 关键
      edgeSymbolSize: 10, // 箭头大小
      lineStyle: {
        width: 2,
        color: "#aaa",
        curveness: 0.2, // 让线条微弯更好看
      },
    },
  ],
  thumbnail: getThumbnail(),
});

function resizeChart() {
  requestAnimationFrame(() => {
    const chart = chartRef.value?.chart;
    if (!chart) return;
    chart.resize();
    chart.setOption({
      series: [
        {
          zoom: 1,
          center: null, // ⭐自动居中
        },
      ],
    });

    chart.resize();
  });
}

async function toggleFullscreen() {
  const chartElement = chartRef.value?.$el || chartRef.value;
  if (!chartElement) return;

  if (!isFullscreen.value) {
    await chartElement.requestFullscreen?.();
    isFullscreen.value = true;
  } else {
    await document.exitFullscreen?.();
    isFullscreen.value = false;
  }

  // ⭐ 等浏览器完成布局后再 resize
  setTimeout(resizeChart, 300);
}

//  监听全屏切换
function handleFullscreenChange() {
  const isCurrentlyFullscreen = !!(
    document.fullscreenElement ||
    (document as any).webkitFullscreenElement ||
    (document as any).mozFullScreenElement ||
    (document as any).msFullscreenElement
  );

  isFullscreen.value = isCurrentlyFullscreen;
  setTimeout(resizeChart, 300);
}

function updateChart() {
  // 添加空值检查
  if (!props.dgraphResponse.nodes) {
    console.warn("props.nodes is undefined");
    return;
  }
  // console.log('props.dgraphResponse.nodes', props.dgraphResponse.nodes)
  const typeSet = [
    ...new Set(props.dgraphResponse.nodes.map((n) => n["dgraph.type"])),
  ];
  const typeList = [
    ...new Set(props.dgraphResponse.nodes.map((n) => n["dgraph.type"])),
  ];

  // 通过 ECharts 实例直接 setOption
  if (chartRef.value?.chart) {
    chartRef.value.chart.setOption(
      {
        legend: {
          data: typeList,
        },
        series: [
          {
            data: props.dgraphResponse.nodes.map((node) => ({
              id: node.id,
              name: node.name,
              category: typeSet.indexOf(node["dgraph.type"]),
            })),
            // 👇 处理 edges 数据，确保包含 name
            edges: props.dgraphResponse.edges.map((edge) => ({
              source: edge.source,
              target: edge.target,
              value: edge.value,
            })),

            categories: typeList.map((type) => ({
              name: type,
            })),
          },
        ],
        thumbnail: getThumbnail(),
      },
      { replaceMerge: ["thumbnail"] },
    );
  }
}

onMounted(async () => {
  // 等待 DOM 更新
  await nextTick();
  updateChart();

  // 添加全屏事件监听
  document.addEventListener("fullscreenchange", handleFullscreenChange);
});
onBeforeUnmount(() => {
  // 清理事件监听
  document.removeEventListener("fullscreenchange", handleFullscreenChange);
});
</script>

<style scoped>
.container-box {
  margin: 24px 0;
  padding: 16px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  border-right: 1px solid rgba(102, 126, 234, 0.3);
  border-top: 1px solid rgba(102, 126, 234, 0.3);
  border-bottom: 1px solid rgba(102, 126, 234, 0.3);
}

.shadow {
  /* 关键：一层柔和的阴影 */
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transition:
    transform 0.25s ease,
    box-shadow 0.25s ease;
}

/* 可选：鼠标悬停时再抬高一点，增强悬浮感 */
.shadow:hover {
  transform: translateY(-4px);
  /* 再抬高 4 px */
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.15);
}

.card-title {
  color: #4a5568;
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
}

.card-title span {
  border-left: 4px solid #21a640;
  padding-left: 15px;
}

.header-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.fullscreen-button {
  display: flex;
  color: black;
  cursor: pointer;
  font-size: 18px;
}

/* 全屏模式下的样式调整 */
/* 当图表元素全屏时的样式 */
.v-charts:fullscreen,
.vue-echarts:fullscreen {
  width: 100vw !important;
  height: 100vh !important;
  margin: 0;
  padding: 20px;
  box-sizing: border-box;
}

.v-charts:-webkit-full-screen,
.vue-echarts:-webkit-full-screen {
  width: 100vw !important;
  height: 100vh !important;
  margin: 0;
  padding: 20px;
  box-sizing: border-box;
}

.v-charts:-moz-full-screen,
.vue-echarts:-moz-full-screen {
  width: 100vw !important;
  height: 100vh !important;
  margin: 0;
  padding: 20px;
  box-sizing: border-box;
}
</style>
