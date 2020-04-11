<script>
export default {
  name: "controls-hider",
  data() {
    return {
      mouseMoving: null,
      hidingControls: 0
    };
  },
  computed: {
    hiddenControlClasses() {
      return {
        "hidden-control": this.hidingControls,
        "show-control": !this.hidingControls || this.mouseMoving !== null
      };
    }
  },
  methods: {
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
    }
  },
  mounted() {
    this.mouseTimer = null;
    this.$el.addEventListener("mousemove", this.mouseMove);
    this.$el.addEventListener("mouseout", this.mouseOut);
  },
  beforeDestroy() {
    this.$el.removeEventListener("mousemove", this.mouseMove);
    this.$el.removeEventListener("mouseout", this.mouseOut);
    if (this.mouseTimer !== null) {
      window.clearInterval(this.mouseTimer);
    }
  }
};
</script>
