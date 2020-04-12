<template>
  <div class="player-box">
    <!-- ensure player gets re-rendered by using a key unique to channel and delivery type -->
    <player
      v-if="chInfo.live && !rtcActive"
      :key="'hls' + channel"
      :ch="chInfo"
      :hlsURL="hlsURL"
      :liveURL="liveURL"
    />
    <player
      v-if="chInfo.live && rtcActive"
      :key="'rtc'+channel"
      :ch="chInfo"
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
  name: "watch",
  props: ["channel"],
  components: { Player },
  computed: {
    chInfo() {
      const ch = this.$root.channels[this.channel];
      if (ch) {
        return ch;
      }
      return {
        live: false,
        rtc: false,
        thumb: null,
        viewers: 0
      };
    },
    liveURL() {
      return "/live/" + encodeURIComponent(this.channel) + ".m3u8";
    },
    hlsURL() {
      return "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8";
    },
    sdpURL() {
      return "/sdp/" + encodeURIComponent(this.channel);
    },
    rtcActive() {
      return this.$root.rtcSelected && this.chInfo.rtc;
    }
  }
};
</script>
