<script lang="ts">
import Vue from "vue";
import Component from 'vue-class-component';

@Component
export default class ControlsHider extends Vue {
  isTabbing = false;
  mouseMoving = 0;
  mouseTimer?: number;
  tabTimer?: number;
  hidingControls: string | number | undefined;

  $el!: HTMLElement;

  mounted() {
    this.$el.addEventListener("mousemove", this.mouseMove);
    this.$el.addEventListener("mouseout", this.mouseOut);
    this.$el.addEventListener("keyup", this.keyUp);
  }

  beforeDestroy() {
    this.$el.removeEventListener("mousemove", this.mouseMove);
    this.$el.removeEventListener("mouseout", this.mouseOut);
    this.$el.removeEventListener("keyup", this.keyUp);
    if (this.mouseTimer) {
      window.clearInterval(this.mouseTimer);
    }
    if (this.tabTimer) {
      window.clearTimeout(this.tabTimer);
    }
  }

  controlsTouched() {
    if (!this.hidingControls) {
      return;
    }
    this.mouseMoving = performance.now() + 3000;
    if (this.mouseTimer) {
      this.mouseTimer = window.setInterval(this.mouseCheck, 500);
    }
  }
  // called by a player when it mounts, remember its unique key
  startHidingControls(key: string | number | undefined) {
    this.hidingControls = key;
  }
  // when a player unmounts, stop hiding controls unless a different player mounted first
  stopHidingControls(key: string | number | undefined) {
    if (this.hidingControls === key) {
      this.hidingControls = undefined;
    }
  }
  // mouse events
  mouseMove(ev: Event) {
    if (!this.hidingControls) {
      return;
    }
    this.mouseMoving = ev.timeStamp + 1000;
    if (this.mouseTimer) {
      this.mouseTimer = window.setInterval(this.mouseCheck, 500);
    }
  }
  mouseOut() {
    this.mouseMoving = 0;
  }
  mouseCheck() {
    if (this.mouseMoving > performance.now()) {
      return;
    }
    this.mouseMoving = 0;
    if (this.mouseTimer) {
      window.clearInterval(this.mouseTimer);
      this.mouseTimer = undefined;
    }
  }
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
  }

  // dynamic classes
  get rootClasses() {
    return {
      "is-tabbing": this.isTabbing
    };
  }
  get hiddenControlClasses() {
    return {
      "hidden-control": !!this.hidingControls,
      "show-control":
        !!(!this.hidingControls || this.mouseMoving || this.isTabbing)
    };
  }
}
</script>

<style>
.hidden-control {
  position: absolute !important;
  opacity: 0;
  transition: opacity 0.5s;
  pointer-events: none;
}
.show-control,
.hidden-control:hover {
  opacity: 1 !important;
  pointer-events: auto !important;
}
.no-outline {
  outline: none;
}
.is-tabbing *:focus {
  outline: 2px solid #7aacfe !important;
  outline: 5px auto -webkit-focus-ring-color !important;
}
</style>
