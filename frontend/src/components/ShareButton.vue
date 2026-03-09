<template>
  <div class="button" :disabled="loading" @click="shareScreen">
    {{ loading ? "…" : "分享" }}
  </div>
</template>

<script setup lang="ts">
import {defineProps, ref} from "vue";
import html2canvas from "html2canvas";

const props = defineProps({
  file_name: String,
});

const loading = ref(false);

async function shareScreen() {
  loading.value = true;
  try {
    const canvas = await html2canvas(document.body, {
      useCORS: true, // 跨域图片
      scale: window.devicePixelRatio || 2, // 清晰度
      backgroundColor: "#fff",
    });

    const blob = await new Promise<Blob | null>((res) =>
        canvas.toBlob(res, "image/png"),
    );
    if (!blob) throw new Error("canvas 转 blob 失败");

    const file = new File([blob], `${props.file_name}.png`, {
      type: "image/png",
    });

    /* 判断当前环境是否支持“带文件的分享” */
    if (navigator.canShare && navigator.canShare({files: [file]})) {
      await navigator.share({
        files: [file],
        title: "分享截图",
        text: "看看这张截图",
      });
    } else {
      /* 理论不会走到这里，Chrome 主进程都支持；以防万一给个提示 */
      alert("当前环境不支持直接分享，已自动下载");
      autoDownload(file);
    }
  } catch (e: any) {
    /* 用户点“取消”也会进 catch，不做提示即可 */
    if (e.name !== "AbortError") {
      alert("分享失败：" + e.message);
    }
  } finally {
    loading.value = false;
  }
}

/** 兜底：自动下载 */
function autoDownload(file: File) {
  const url = URL.createObjectURL(file);
  const a = document.createElement("a");
  a.href = url;
  a.download = file.name;
  a.style.display = "none";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
</script>

<style scoped></style>
