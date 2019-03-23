<template>
  <div class="player-box">
    <hls-player :channel="channel" v-if="!$root.useRTC" />
    <rtc-player :channel="channel" v-if="$root.useRTC" />
  </div>
</template>

<script>
export default {
  name: 'watch',
  props: [
    'channel',
  ],
}
</script>

<style>
.player-box {
  width: 100%;
  height: calc(100vh - 56px);
}
</style>
