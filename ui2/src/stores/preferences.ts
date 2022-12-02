import { defineStore } from "pinia";

interface Preferences {
  useRTC: boolean;
  lowLatency: boolean;
}

export const usePreferences = defineStore("preferences", {
  state: (): Preferences => ({
    useRTC: false,
    lowLatency: true,
  }),
  actions: {
    fromLocalStorage() {
      this.$patch({
        useRTC: localStorage.getItem("playerType") === "RTC",
        lowLatency: localStorage.getItem("lowLatency") !== "false",
      });
      this.$subscribe((_mutation, state) => {
        localStorage.setItem("playerType", state.useRTC ? "RTC" : "HLS");
        localStorage.setItem("lowLatency", JSON.stringify(state.lowLatency));
      });
    },
  },
});
