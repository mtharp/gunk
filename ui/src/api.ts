import Vue, { VueConstructor } from 'vue';
import Component from 'vue-class-component';
import axios from "axios";
import WSSession from './ws';

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

interface UserInfo {
    id: string;
    username: string;
    discriminator: string;
    avatar: string;
}

interface ChannelsResponse {
    channels: { [key: string]: ChannelInfo; };
    recent: string[];
}

export class API {
    ws?: WSSession;
    channels: { [key: string]: ChannelInfo; } = {};
    recentChannels: string[] = [];
    user?: UserInfo;

    private userinterval?: number;

    connect() {
        this.updateChannels();
        this.updateUser();
        this.userinterval = window.setInterval(this.updateUser, 300000);
        this.ws = new WSSession(location);
        this.ws.onChannel = (ch) => this.updateChannelInfo(ch);
    }

    disconnect() {
        this.ws?.close();
        if (this.userinterval) {
            window.clearInterval(this.userinterval);
        }
    }

    doLogin () {
        window.location.href = '/oauth2/initiate';
    }
    async doLogout () {
        await axios.post('/oauth2/logout');
        this.user = undefined;
        await this.updateUser();
    }

    private updateChannelInfo(ch: ChannelInfo) {
        const newch = Object.assign({}, this.channels);
        newch[ch.name] = ch;
        this.channels = newch;
    }

    private async updateChannels () {
        const { data } = await axios.get<ChannelsResponse>('/channels.json');
        this.channels = data.channels;
        this.recentChannels = data.recent;
    }

    private async updateUser() {
        const { data } = await axios.get<UserInfo>('/oauth2/user');
        this.user = data;
    }
}

const APIInstance = new API();

@Component
export class APIMixin extends Vue {
    readonly api = APIInstance;
}
