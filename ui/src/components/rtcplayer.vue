<template>
  <video class="w-100 h-100" muted></video>
</template>

<script>
import { restoreVolume, attachRTC } from "../player.js";

export default {
  name: "rtc-player",
  props: ["channel"],
  mounted() {
    let video = this.$el;
    video.poster = this.$root.currentChannel().thumb;
    this.$nextTick(() => restoreVolume(video));
    let sdpURL = "/sdp/" + encodeURIComponent(this.channel);
    this.destroyPlayer = attachRTC(video, sdpURL);
  },
  beforeDestroy() {
    this.destroyPlayer();
  }
};
</script>
