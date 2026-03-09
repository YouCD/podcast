<!-- NetworkGraph.vue -->
<template>
  <div class="network-container">
    <div class="control-panel">
      <div class="stats">
        <span class="badge">企业: {{ enterpriseCount }}</span>
        <span class="badge">关系: {{ relationshipCount }}</span>
        <span class="badge">节点: {{ nodes.length }}</span>
      </div>

      <div class="filters">
        <label v-for="type in relationTypes" :key="type">
          <input type="checkbox" v-model="activeRelations" :value="type"/>
          <span :style="{ color: getRelationColor(type) }">{{ type }}</span>
        </label>
      </div>

      <button @click="resetZoom" class="btn">重置视图</button>
      <button @click="restartSimulation" class="btn">重新布局</button>
    </div>

    <div ref="container" class="graph-canvas"></div>
  </div>
</template>

<script setup>
import {computed, onBeforeUnmount, onMounted, ref, watch} from "vue";
import * as d3 from "d3";

const props = defineProps({
  data: {type: Array, default: () => []},
  loading: Boolean,
});

const container = ref(null);
const svg = ref(null);
const simulation = ref(null);
const g = ref(null);
const zoomBehavior = ref(null);

const activeRelations = ref([
  "研发",
  "投资",
  "收购",
  "竞争",
  "合作",
  "雇佣",
  "裁员",
  "掌管",
  "所属行业",
]);

const relationTypes = [
  "研发",
  "投资",
  "收购",
  "竞争",
  "合作",
  "雇佣",
  "裁员",
  "掌管",
  "所属行业",
];

const relationColors = {
  研发: "#ef4444",
  投资: "#22c55e",
  收购: "#f97316",
  竞争: "#eab308",
  合作: "#3b82f6",
  雇佣: "#8b5cf6",
  裁员: "#ec4899",
  掌管: "#14b8a6",
  所属行业: "#64748b",
  基于: "#94a3b8",
};

const getRelationColor = (type) => relationColors[type] || "#94a3b8";

/* ===============================
   正确解析 Dgraph JSON
================================ */
const graphData = computed(() => {
  const nodes = new Map();
  const links = [];

  props.data.forEach((entity) => {
    if (!entity.uid) return;

    // 主企业节点
    if (!nodes.has(entity.uid)) {
      nodes.set(entity.uid, {
        id: entity.uid,
        name: entity.name || "未知企业",
        type: "企业",
        radius: 25,
      });
    }

    /* ===== 反向关系：~研发 ===== */
    if (entity["~研发"]) {
      entity["~研发"].forEach((dev) => {
        if (!dev.uid) return;

        if (!nodes.has(dev.uid)) {
          nodes.set(dev.uid, {
            id: dev.uid,
            name: dev.name || "未知技术",
            type: "技术",
            radius: 18,
          });
        }

        links.push({
          source: dev.uid,
          target: entity.uid,
          type: "研发",
        });

        // 基于关系
        if (dev["基于"]) {
          dev["基于"].forEach((base) => {
            if (!base.uid) return;

            if (!nodes.has(base.uid)) {
              nodes.set(base.uid, {
                id: base.uid,
                name: base.name || "基础技术",
                type: "基础",
                radius: 15,
              });
            }

            links.push({
              source: dev.uid,
              target: base.uid,
              type: "基于",
            });
          });
        }
      });
    }

    /* ===== 正向关系 ===== */
    activeRelations.value.forEach((rel) => {
      if (!entity[rel]) return;

      entity[rel].forEach((target) => {
        if (!target.uid) return;

        if (!nodes.has(target.uid)) {
          nodes.set(target.uid, {
            id: target.uid,
            name: target.name || "未知",
            type: rel === "所属行业" ? "行业" : "实体",
            radius: rel === "所属行业" ? 20 : 15,
          });
        }

        links.push({
          source: entity.uid,
          target: target.uid,
          type: rel,
        });
      });
    });
  });

  return {
    nodes: Array.from(nodes.values()),
    links,
  };
});

const nodes = computed(() => graphData.value.nodes);
const enterpriseCount = computed(() => props.data.length);
const relationshipCount = computed(() => graphData.value.links.length);

/* ===============================
   D3 初始化
================================ */
const initGraph = () => {
  if (!container.value || !nodes.value.length) return;

  const width = container.value.clientWidth;
  const height = container.value.clientHeight || 700;

  d3.select(container.value).selectAll("*").remove();

  svg.value = d3
      .select(container.value)
      .append("svg")
      .attr("width", width)
      .attr("height", height);

  g.value = svg.value.append("g");

  zoomBehavior.value = d3
      .zoom()
      .scaleExtent([0.1, 4])
      .on("zoom", (event) => {
        g.value.attr("transform", event.transform);
      });

  svg.value.call(zoomBehavior.value);

  // simulation.value = d3.forceSimulation(nodes.value)
  //     .force('link', d3.forceLink(graphData.value.links)
  //         .id(d=>d.id)
  //         .distance(120)
  //     )
  //     .force('charge', d3.forceManyBody().strength(-400))
  //     .force('center', d3.forceCenter(width/2, height/2))
  //     .force('collision', d3.forceCollide().radius(d=>d.radius+8))
  simulation.value = d3
      .forceSimulation(nodes.value)
      .force(
          "link",
          d3
              .forceLink(graphData.value.links)
              .id((d) => d.id)
              .distance((d) => {
                if (d.type === "基于") return 50;
                if (d.type === "所属行业") return 70;
                return 80;
              })
              .strength(0.8),
      )
      .force(
          "charge",
          d3.forceManyBody().strength((d) => (d.type === "企业" ? -300 : -120)),
      )
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force(
          "collision",
          d3
              .forceCollide()
              .radius((d) => d.radius + 4)
              .strength(1),
      )
      .force("x", d3.forceX(width / 2).strength(0.1))
      .force("y", d3.forceY(height / 2).strength(0.1));

  //  企业固定中心，其他围绕
  // simulation.value.force('radial',
  //     d3.forceRadial(d => {
  //       if (d.type === '企业') return 0
  //       if (d.type === '行业') return 120
  //       return 80
  //     }, width / 2, height / 2).strength(0.6)
  // )
  // 箭头
  const defs = svg.value.append("defs");
  Object.keys(relationColors).forEach((type) => {
    defs
        .append("marker")
        .attr("id", `arrow-${type}`)
        .attr("viewBox", "0 -5 10 10")
        .attr("refX", 20)
        .attr("refY", 0)
        .attr("markerWidth", 6)
        .attr("markerHeight", 6)
        .attr("orient", "auto")
        .append("path")
        .attr("d", "M0,-5L10,0L0,5")
        .attr("fill", relationColors[type]);
  });

  const link = g.value
      .append("g")
      .selectAll("line")
      .data(graphData.value.links)
      .join("line")
      .attr("stroke", (d) => getRelationColor(d.type))
      .attr("stroke-width", 1.8)
      .attr("stroke-opacity", 0.6)
      .attr("marker-end", (d) => `url(#arrow-${d.type})`);

  const node = g.value
      .append("g")
      .selectAll("g")
      .data(nodes.value)
      .join("g")
      .call(
          d3
              .drag()
              .on("start", (e, d) => {
                if (!e.active) simulation.value.alphaTarget(0.3).restart();
                d.fx = d.x;
                d.fy = d.y;
              })
              .on("drag", (e, d) => {
                d.fx = e.x;
                d.fy = e.y;
              })
              .on("end", (e, d) => {
                if (!e.active) simulation.value.alphaTarget(0);
                d.fx = null;
                d.fy = null;
              }),
      );

  node
      .append("circle")
      .attr("r", (d) => d.radius)
      .attr("fill", (d) => {
        if (d.type === "企业") return "#0f172a";
        if (d.type === "行业") return "#f8fafc";
        return getRelationColor(d.type);
      })
      .attr("stroke", (d) => {
        if (d.type === "企业") return "#3b82f6";
        if (d.type === "行业") return "#cbd5e1";
        return "white";
      })
      .attr("stroke-width", 2)
      .style("filter", (d) =>
          d.type === "企业"
              ? "drop-shadow(0 0 8px rgba(59,130,246,0.4))"
              : "drop-shadow(0 2px 4px rgba(0,0,0,0.1))",
      );
  //  hover 高亮：
  node
      .on("mouseover", function () {
        d3.select(this)
            .select("circle")
            .transition()
            .duration(200)
            .attr("r", (d) => d.radius + 3);
      })
      .on("mouseout", function () {
        d3.select(this)
            .select("circle")
            .transition()
            .duration(200)
            .attr("r", (d) => d.radius);
      });

  node
      .append("text")
      .attr("dy", 4)
      .attr("text-anchor", "middle")
      .attr("fill", "#fff")
      .style("font-size", "10px")
      .text((d) => d.name.slice(0, 6));

  simulation.value.on("tick", () => {
    link
        .attr("x1", (d) => d.source.x)
        .attr("y1", (d) => d.source.y)
        .attr("x2", (d) => d.target.x)
        .attr("y2", (d) => d.target.y);

    node.attr("transform", (d) => `translate(${d.x},${d.y})`);
  });
};

const resetZoom = () => {
  if (svg.value && zoomBehavior.value) {
    svg.value
        .transition()
        .duration(500)
        .call(zoomBehavior.value.transform, d3.zoomIdentity);
  }
};

const restartSimulation = () => {
  if (simulation.value) {
    simulation.value.alpha(1).restart();
  }
};

watch(
    () => props.data,
    () => {
      simulation.value?.stop();
      initGraph();
    },
    {deep: true, immediate: true},
);

watch(
    activeRelations,
    () => {
      simulation.value?.stop();
      initGraph();
    },
    {deep: true},
);

onMounted(() => {
  if (props.data.length) initGraph();
});

onBeforeUnmount(() => {
  simulation.value?.stop();
});
</script>

<style scoped>
.network-container {
  position: relative;
  width: 100%;
  height: 100vh;
  background: radial-gradient(circle at 30% 30%, #f1f5f9, #e2e8f0, #cbd5e1);
  overflow: hidden;
  font-family: "Inter",
  -apple-system,
  BlinkMacSystemFont,
  sans-serif;
}

.network-container::before {
  content: "";
  position: absolute;
  width: 600px;
  height: 600px;
  background: radial-gradient(circle, rgba(59, 130, 246, 0.2), transparent 70%);
  top: -200px;
  left: -200px;
  pointer-events: none;
}

.control-panel {
  position: absolute;
  top: 24px;
  left: 24px;
  z-index: 10;
  backdrop-filter: blur(16px);
  background: rgba(255, 255, 255, 0.75);
  padding: 20px;
  border-radius: 18px;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.08),
  inset 0 1px 0 rgba(255, 255, 255, 0.6);
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 240px;
  border: 1px solid rgba(255, 255, 255, 0.4);
}

.badge {
  background: linear-gradient(135deg, #e0f2fe, #dbeafe);
  padding: 6px 14px;
  border-radius: 30px;
  font-size: 12px;
  font-weight: 600;
  color: #1e40af;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.6);
}

.graph-canvas {
  width: 100%;
  height: 100%;
}

.btn {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: white;
  border: none;
  padding: 8px 14px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.25s ease;
  box-shadow: 0 4px 10px rgba(59, 130, 246, 0.25);
}

.btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(59, 130, 246, 0.35);
}

.btn:last-child {
  background: linear-gradient(135deg, #64748b, #475569);
}

::-webkit-scrollbar {
  width: 8px;
}

::-webkit-scrollbar-thumb {
  background: linear-gradient(#94a3b8, #64748b);
  border-radius: 6px;
}
</style>
