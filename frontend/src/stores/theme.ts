import { defineStore } from "pinia";

export interface ThemeStore {
  mode: "light" | "dark" | "auto";
}

export const useThemeStore = defineStore("theme", {
  state: (): ThemeStore => ({
    mode: "auto",
  }),

  getters: {
    isDark(): boolean {
      return this.mode === "dark";
    },
  },

  actions: {
    init() {
      const saved = localStorage.getItem("theme-mode");
      if (saved) {
        this.mode = saved as "light" | "dark" | "auto";
      }



      this.syncThemeClass();
    },

    toggle() {
      this.mode = this.isDark ? "light" : "dark";
      localStorage.setItem("theme-mode", this.mode);
      this.syncThemeClass();
    },



    syncThemeClass() {
      document.documentElement.classList.toggle(
        "dark",
        this.isDark
      );
    },
  },
});
