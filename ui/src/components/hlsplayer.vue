<template>
  <video class="w-100 h-100" muted></video>
</template>

<script>
import { restoreVolume, attachHLS } from "../player.js";

export default {
  name: "hls-player",
  props: ["channel"],
  mounted() {
    let video = this.$el;
    video.poster = this.$root.currentChannel().thumb;
    this.$nextTick(() => restoreVolume(video));
    let hlsURL = "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8";
    this.destroyPlayer = attachHLS(video, hlsURL);
  },
  beforeDestroy() {
    this.destroyPlayer();
  }
};
</script>
