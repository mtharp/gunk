<template>
  <div class="player-box">
    <!-- ensure player gets re-rendered by using a key unique to channel and delivery type -->
    <player
      v-if="chInfo.live"
      :key="
        (rtcActive ? 'rtc.' : 'hls.') +
        ($root.lowLatency ? 'll.' : '') +
        channel
      "
      :ch="chInfo"
      :rtcActive="rtcActive"
      :lowLatency="$root.lowLatency"
    />
    <img
      v-if="!chInfo.live && chInfo.thumb"
      :src="chInfo.thumb"
      class="player-thumb"
    />
    <div v-if="!chInfo.live" class="player-shade">OFFLINE</div>
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import Component, { mixins } from 'vue-class-component';
import { APIMixin } from "../api";
import Player from "../components/player.vue";
import Gunk from "../main";

const WatchProps = Vue.extend({
  props: {
    channel: String,
  }
});

@Component({
  components: { Player },
})
export default class Watch extends mixins(WatchProps, APIMixin) {
  $root!: Gunk;
  
    get chInfo() {
      const ch = this.api.channels[this.channel];
      if (ch) {
        return ch;
      }
      return {
        live: false,
        rtc: false,
        thumb: null,
        viewers: 0
      };
    }
    get rtcActive() {
      return this.$root.rtcSelected && this.chInfo.rtc;
    }
}
</script>
