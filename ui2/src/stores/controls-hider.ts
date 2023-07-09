import { defineStore } from "pinia";

interface ControlsState {
  configured: boolean;
  isTabbing: boolean;
  mouseMoving: number;
  mouseTimer?: number;
  tabTimer?: number;
  targetPlayer?: HTMLVideoElement;
}

export const useControlsHider = defineStore("controls-hider", {
  state: (): ControlsState => ({
    configured: false,
    isTabbing: false,
    mouseMoving: 0,
    mouseTimer: undefined,
    tabTimer: undefined,
    targetPlayer: undefined,
  }),
  actions: {
    // called by a player when it mounts, remember its unique key
    attachPlayer(key: any) {
      this.targetPlayer = key;
      if (!this.configured) {
        document.addEventListener("mousemove", this.mouseMove);
        document.addEventListener("mouseout", this.mouseOut);
        document.addEventListener("keyup", this.keyUp);
        this.configured = true;
      }
    },
    // when a player unmounts, stop hiding controls unless a different player
    // mounted first
    detachPlayer(key: any) {
      if (this.targetPlayer === key) {
        this.targetPlayer = undefined;
      }
    },
    controlsTouched() {
      if (!this.targetPlayer) {
        return;
      }
      this.mouseMoving = performance.now() + 3000;
      if (!this.mouseTimer) {
        this.mouseTimer = window.setInterval(this.mouseCheck, 500);
      }
    },
    // mouse events
    mouseMove(ev: Event) {
      if (!this.targetPlayer) {
        return;
      }
      this.mouseMoving = ev.timeStamp + 1000;
      if (!this.mouseTimer) {
        this.mouseTimer = window.setInterval(this.mouseCheck, 500);
      }
    },
    mouseOut() {
      this.mouseMoving = 0;
    },
    mouseCheck() {
      if (this.mouseMoving > performance.now()) {
        return;
      }
      this.mouseMoving = 0;
      if (this.mouseTimer) {
        window.clearInterval(this.mouseTimer);
        this.mouseTimer = undefined;
      }
    },
    // keyboard events
    keyUp(ev: KeyboardEvent) {
      switch (ev.key) {
        case "Tab":
          this.isTabbing = true;
          if (this.tabTimer) {
            window.clearTimeout(this.tabTimer);
          }
          this.tabTimer = window.setTimeout(() => {
            this.tabTimer = undefined;
            this.isTabbing = false;
          }, 10000);
          break;
        case "Escape":
          this.isTabbing = false;
          break;
        default:
          this.controlsTouched();
      }
    },
  },
  getters: {
    // dynamic classes
    rootClasses(state) {
      return {
        "is-tabbing": state.isTabbing,
      };
    },
    hiddenControlClasses(state) {
      return {
        "hidden-control": !!state.targetPlayer,
        "show-control": !!(
          !state.targetPlayer ||
          state.mouseMoving ||
          state.isTabbing
        ),
      };
    },
  },
});
