<template>
  <div class="player-box">
    <hls-player :channel="channel" v-if="$root.playerType == 'HLS' || !$root.playerType" />
    <rtc-player :channel="channel" v-if="$root.playerType == 'RTC'" />
    <b-modal
      v-model="$root.showStreamInfo"
      title="Stream Info"
      ok-only
      >
      <p>HLS URL:</p>
      <p><strong>{{hlsURL}}</strong></p>
      <p>Live URL (for VLC)</p>
      <p><strong>{{liveURL}}</strong></p>
    </b-modal>
  </div>
</template>

<script>
export default {
  name: 'watch',
  props: [
    'channel',
  ],
  computed: {
    ch() {
      for (let ch of Object.values(this.$root.channels)) {
        if (ch.name == this.channel) {
          return ch
        }
      }
      return null
    },
    baseURL() {
      let base = window.location.protocol + "//" + window.location.hostname
      if (window.location.port != "") {
        base += ":" + window.location.port
      }
      return base
    },
    hlsURL() { return this.baseURL + "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8" },
    liveURL() {
      let ch = this.ch
      if (ch === null) {
        return null
      }
      let u = ch.live_url
      if (u[0] == '/') {
        return this.baseURL + u
      }
      return u
    }
  },
}
</script>
