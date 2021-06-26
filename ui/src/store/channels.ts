import { Module, VuexModule, Mutation, MutationAction, getModule } from 'vuex-module-decorators'
import axios from "axios";
import store from '.';

export interface ChannelInfo {
    name: string;
    live: boolean;
    last: number;
    thumb: string;
    live_url: string;
    web_url: string;
    native_url: string;
    viewers: number;
    rtc: boolean;
}

interface ChannelsResponse {
    channels: { [key: string]: ChannelInfo; };
    recent: string[];
}

@Module({name: "channels", store: store, dynamic: true})
export class ChannelsModule extends VuexModule {
    channels: { [key: string]: ChannelInfo; } = {};
    recentChannels: string[] = [];

    @MutationAction
    async refreshChannels() {
        const { data } = await axios.get<ChannelsResponse>('/channels.json');
        return {
            channels: data.channels,
            recentChannels: data.recent,
        }
    }

    @Mutation
    putChannel(ch: ChannelInfo) {
        const newch = Object.assign({}, this.channels);
        newch[ch.name] = ch;
        this.channels = newch;
    }
}

export default getModule(ChannelsModule);
