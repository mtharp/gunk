<script>
export default {
  name: "controls-hider",
  data() {
    return {
      mouseMoving: null,
      hidingControls: 0,
      isTabbing: false
    };
  },
  computed: {
    rootClasses() {
      return {
        "is-tabbing": this.isTabbing
      };
    },
    hiddenControlClasses() {
      return {
        "hidden-control": this.hidingControls,
        "show-control":
          !this.hidingControls || this.mouseMoving !== null || this.isTabbing
      };
    }
  },
  methods: {
    // public methods
    controlsTouched() {
      if (!this.hidingControls) {
        return;
      }
      this.mouseMoving = performance.now() + 3000;
      if (this.mouseTimer === null) {
        this.mouseTimer = window.setInterval(this.mouseCheck, 500);
      }
    },
    startHidingControls() {
      this.hidingControls++;
    },
    stopHidingControls() {
      this.hidingControls--;
    },
    // mouse events
    mouseMove(ev) {
      if (!this.hidingControls) {
        return;
      }
      this.mouseMoving = ev.timeStamp + 1000;
      if (this.mouseTimer === null) {
        this.mouseTimer = window.setInterval(this.mouseCheck, 500);
      }
    },
    mouseOut() {
      this.mouseMoving = null;
    },
    mouseCheck() {
      if (this.mouseMoving !== null && this.mouseMoving > performance.now()) {
        return;
      }
      this.mouseMoving = null;
      if (this.mouseTimer !== null) {
        window.clearInterval(this.mouseTimer);
        this.mouseTimer = null;
      }
    },
    // keyboard events
    keyUp(ev) {
      switch (ev.key) {
        case "Tab":
          this.isTabbing = true;
          if (this.tabTimer !== null) {
            window.clearTimeout(this.tabTimer);
          }
          this.tabTimer = window.setTimeout(() => {
            this.tabTimer = null;
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
  },
  mounted() {
    this.mouseTimer = null;
    this.tabTimer = null;
    this.$el.addEventListener("mousemove", this.mouseMove);
    this.$el.addEventListener("mouseout", this.mouseOut);
    this.$el.addEventListener("keyup", this.keyUp);
  },
  beforeDestroy() {
    this.$el.removeEventListener("mousemove", this.mouseMove);
    this.$el.removeEventListener("mouseout", this.mouseOut);
    this.$el.removeEventListener("keyup", this.keyUp);
    if (this.mouseTimer !== null) {
      window.clearInterval(this.mouseTimer);
    }
    if (this.tabTimer !== null) {
      window.clearTimeout(this.tabTimer);
    }
  }
};
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
