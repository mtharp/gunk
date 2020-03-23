<template>
  <video-js id="player" autoplay controls class="video-js vjs-default-skin w-100 h-100">
    <source :src="hlsURL" type="application/x-mpegURL" />
  </video-js>
</template>

<script>
import "video.js/dist/video-js.css";
import videojs from "video.js/dist/video.js";

export default {
  name: "hls-player",
  props: ["channel"],
  computed: {
    hlsURL() {
      return "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8";
    }
  },
  mounted() {
    this.player = videojs("player", {
      responsive: true,
      controls: true,
      liveui: true,
      poster: this.$root.currentChannel().thumb
    });
    this.player.ready(() => this.player.play());
  },
  beforeDestroy() {
    this.player.dispose();
  }
};
</script>
