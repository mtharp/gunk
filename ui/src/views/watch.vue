<template>
  <div class="player-box">
    <!-- ensure player gets re-rendered by using a key unique to channel and delivery type -->
    <player
      v-if="chInfo.live"
      :key="(rtcActive ? 'rtc.' : 'hls.')+channel"
      :ch="chInfo"
      :rtcActive="rtcActive"
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
    rtcActive() {
      return this.$root.rtcSelected && this.chInfo.rtc;
    }
  }
};
</script>
