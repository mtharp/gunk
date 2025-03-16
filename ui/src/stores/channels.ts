import { defineStore } from "pinia";
import axios from "axios";

export interface ChannelInfo {
  name: string;
  live: boolean;
  last: number;
  thumb: string;
  hls_url: string;
  live_url: string;
  whep_url: string;
  viewers: number;
}

export function nullChannelInfo(): ChannelInfo {
  return {
    name: "",
    live: false,
    last: 0,
    thumb: "",
    hls_url: "",
    live_url: "",
    whep_url: "",
    viewers: 0,
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
  getters: {
    live(): string[] {
      const names = [];
      for (const name in this.channels) {
        const ch = this.channels[name];
        if (ch.live) {
          names.push(ch.name);
        }
      }
      return names.sort();
    },
  },
});
