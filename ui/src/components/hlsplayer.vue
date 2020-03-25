<template>
  <video class="w-100 h-100" muted :poster="poster"></video>
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
  methods: {
    autoplay() {
      let video = this.$el;
      video.play();
      let vol = localStorage.getItem("volume");
      if (vol) {
        video.volume = vol / 100;
      }
      if (localStorage.getItem("unmute") == "true") {
        video.muted = false;
      }
    },
    saveVolume() {
      let video = this.$el;
      localStorage.setItem("unmute", !video.muted);
      localStorage.setItem("volume", Math.round(video.volume * 100));
    }
  },
  mounted() {
    let video = this.$el;
    video.controls = true;
    if (Hls.isSupported()) {
      this.hls = new Hls({
        bitrateTest: false
        // debug: true
      });
      this.hls.attachMedia(video);
      this.hls.loadSource(this.hlsURL);
      this.hls.on(Hls.Events.MANIFEST_PARSED, this.autoplay);
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = this.hlsURL;
      video.addEventListener("loadedmetadata", this.autoplay);
    }
    video.addEventListener("volumechange", this.saveVolume);
  },
  beforeDestroy() {
    if (this.hls !== null) {
      this.hls.destroy();
      this.hls = null;
    }
  }
};
</script>
