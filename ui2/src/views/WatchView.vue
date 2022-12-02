<template>
  <div class="player-box">
    <!-- ensure player gets re-rendered by using a key unique to channel and delivery type -->
    <player-box
      v-if="chInfo.live"
      :key="
        (rtcActive ? 'rtc.' : 'hls.') +
        (preferences.lowLatency ? 'll.' : '') +
        channel
      "
      :ch="chInfo"
      :rtcActive="rtcActive"
      :lowLatency="preferences.lowLatency"
    />
    <img
      v-if="!chInfo.live && chInfo.thumb"
      :src="chInfo.thumb"
      class="player-thumb"
    />
    <div v-if="!chInfo.live" class="player-shade">OFFLINE</div>
  </div>
</template>

<script setup lang="ts">
import { useChannelsStore, type ChannelInfo } from "@/stores/channels";
import { usePreferences } from "@/stores/preferences";
import { computed } from "vue";
import PlayerBox from "@/components/PlayerBox.vue";

const props = defineProps<{
  channel: string;
}>();
const channels = useChannelsStore();
const preferences = usePreferences();

const chInfo = computed((): ChannelInfo => {
  const ch = channels.channels[props.channel];
  if (ch) {
    return ch;
  }
  return {
    last: 0,
    live: false,
    live_url: "",
    name: "",
    native_url: "",
    rtc: false,
    thumb: "",
    viewers: 0,
    web_url: "",
  };
});
const rtcActive = computed(() => preferences.useRTC && chInfo.value.rtc);
</script>
