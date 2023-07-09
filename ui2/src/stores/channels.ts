import { defineStore } from "pinia";
import axios from "axios";

export interface ChannelInfo {
  name: string;
  live: boolean;
  pending: boolean;
  last: number;
  thumb: string;
  live_url: string;
  web_url: string;
  native_url: string;
  viewers: number;
  rtc: boolean;
}

export function nullChannelInfo(): ChannelInfo {
  return {
    name: "",
    live: false,
    pending: false,
    last: 0,
    thumb: "",
    live_url: "",
    web_url: "",
    native_url: "",
    viewers: 0,
    rtc: false,
  };
}

interface ChannelState {
  channels: { [key: string]: ChannelInfo };
  recent: string[];
}

export const useChannelsStore = defineStore("channels", {
  state: (): ChannelState => ({
    channels: {},
    recent: [],
  }),
  actions: {
    async refreshChannels() {
      const { data } = await axios.get<ChannelState>("/channels.json");
      this.$patch(data);
    },
    putChannel(ch: ChannelInfo) {
      this.channels[ch.name] = ch;
    },
  },
});
