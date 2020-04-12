<template>
  <div class="player-box">
    <!-- ensure player gets re-rendered by using a key unique to channel and delivery type -->
    <player
      v-if="chInfo.live && !rtcActive"
      :key="'hls' + channel"
      :poster="chInfo.thumb"
      :hlsURL="hlsURL"
      :liveURL="liveURL"
      :rtcEnabled="chInfo.rtc"
    />
    <player
      v-if="chInfo.live && rtcActive"
      :key="'rtc'+channel"
      :poster="chInfo.thumb"
      :sdpURL="sdpURL"
      :liveURL="liveURL"
      :rtcEnabled="chInfo.rtc"
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
  data() {
    return { rtcSelected: false };
  },
  computed: {
    chInfo() {
      const ch = this.$root.channels[this.channel];
      if (ch) {
        return ch;
      }
      return {
        live: false,
        rtc: false,
        thumb: null
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
      return this.rtcSelected && this.chInfo.rtc;
    }
  }
};
</script>
