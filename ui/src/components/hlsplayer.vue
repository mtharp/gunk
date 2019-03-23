<template>
  <video-js
    id="player"
    class="video-js vjs-default-skin w-100 h-100"
    >
    <source :src="hlsURL" type="application/x-mpegURL" />
  </video-js>
</template>

<script>
import 'video.js/dist/video-js.css'
import videojs from 'video.js/dist/video.js'

export default {
  name: 'hls-player',
  props: [
    'channel',
  ],
  computed: {
    hlsURL() { return "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8" },
  },
  mounted() {
    this.player = videojs("player", {
      responsive: true,
      muted: true,
      controls: true,
    })
    this.player.ready(() => this.player.play())
  },
  beforeDestroy() {
    this.player.dispose();
  },
}
</script>
