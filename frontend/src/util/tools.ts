import type { Report, Rss } from "@/types/types.ts";

export function iconHandel(p: Rss | Report): string {
  // 可以这样判断是否为 Report 类型
  if ("podcast_mp3_url" in p) {
    // 处理 Report 类型逻辑
    if (p.podcast_mp3_url != "") {
      return "\ue6ec";
    }
    return "\ue6c1";
  }

  // if ("score" in p) {
  //     // 处理 Report 类型逻辑
  //     if (p.score >= 80) {
  //         return "#icon-aixin1";
  //     }
  // }

  return "\ue634";
}

// id转字母
export function indexParser(index: number): string {
  switch (index) {
    case 0:
      return "A";
    case 1:
      return "B";
    case 2:
      return "C";
    case 3:
      return "D";
    case 4:
      return "E";
    case 5:
      return "F";
    case 6:
      return "G";
    case 7:
      return "H";
    case 8:
      return "I";
  }
  return "A";
}

export function loaderStorage(): number[] {
  const stored = localStorage.getItem("readList");
  if (stored) {
    try {
      const parsed = JSON.parse(stored);
      if (Array.isArray(parsed)) {
        return Array.from(new Set(parsed));
      }
    } catch (e) {
      console.error("解析已播放播客数据失败:", e);
      return []; // 默认返回空数组
    }
  }
  return []; // 默认返回空数组
}

// saveStorage
export function saveStorage(ids: number[] | number) {
  try {
    const idsArray = Array.isArray(ids) ? ids : [ids];

    loaderStorage().forEach((id) => {
      idsArray.push(id);
    });

    localStorage.setItem("readList", JSON.stringify(Array.from(idsArray)));
  } catch (e) {
    console.error("保存已播放播客数据失败:", e);
  }
}

// 获取截断的数据显示
export function getTruncatedData(data: string, maxLength = 100) {
  if (data.length <= maxLength) return data;
  return data.substring(0, maxLength) + "...";
}
