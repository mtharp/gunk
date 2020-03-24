<template>
  <video id="player" ref="player" class="w-100 h-100" controls muted autoplay :poster="poster"></video>
</template>

<script>
import Hls from "hls.js/dist/hls.js";

export default {
  name: "hls-player",
  props: ["channel"],
  data() {
    return {
      poster: this.$root.currentChannel().thumb
    };
  },
  computed: {
    hlsURL() {
      return "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8";
    }
  },
  mounted() {
    let video = this.$refs.player;
    if (Hls.isSupported()) {
      this.hls = new Hls({
        bitrateTest: false
        // debug: true
      });
      this.hls.attachMedia(video);
      this.hls.loadSource(this.hlsURL);
      this.hls.on(Hls.Events.MANIFEST_PARSED, () => video.play());
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = this.hlsURL;
      video.addEventListener("loadedmetadata", () => video.play());
    }
  },
  beforeDestroy() {
    if (this.hls !== null) {
      this.hls.destroy();
      this.hls = null;
    }
  }
};
</script>
