import Vue, { VueConstructor } from 'vue';
import { getModule } from 'vuex-module-decorators';
import Component from 'vue-class-component';
import axios from "axios";
import WSSession from './ws';
import store from './store'
import Channels from './store/channels';



export class API {
    ws?: WSSession;

    connect() {
        this.ws = new WSSession(location);
        this.ws.onChannel = (ch) => Channels.putChannel(ch);
    }

    disconnect() {
        this.ws?.close();
    }
}

const APIInstance = new API();

@Component
export class APIMixin extends Vue {
    readonly api = APIInstance;
}
