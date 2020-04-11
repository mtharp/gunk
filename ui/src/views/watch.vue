<template>
  <div class="player-box">
    <player
      v-if="chInfo.live && !useRTC"
      :poster="chInfo.thumb"
      :hlsURL="hlsURL"
      :liveURL="liveURL"
    />
    <player
      v-if="chInfo.live && useRTC"
      :poster="chInfo.thumb"
      :sdpURL="sdpURL"
      :liveURL="liveURL"
    />
    <img v-if="!chInfo.live && chInfo.thumb" :src="chInfo.thumb" class="player-thumb" />
    <div v-if="!chInfo.live" class="player-shade">OFFLINE</div>
  </div>
</template>

<script>
import Player from "../components/player";

export default {
  name: "hls-player",
  props: ["channel"],
  components: { Player },
  data() {
    return {
      useRTC: false,
      liveURL: "/live/" + encodeURIComponent(this.channel) + ".m3u8",
      hlsURL: "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8",
      sdpURL: "/sdp/" + encodeURIComponent(this.channel)
    };
  },
  computed: {
    chInfo() {
      const ch = this.$root.channels[this.channel];
      if (ch) {
        return ch;
      }
      return {
        live: false,
        thumb: null
      };
    }
  }
};
</script>
