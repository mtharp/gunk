import Vue from 'vue';
import App from './app.vue';
import router from './router';
import ControlsHider from './components/controls-hider.vue';
import axios from 'axios';
import WSSession from './ws.js';

import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-vue/dist/bootstrap-vue.css';
import { BootstrapVue, BootstrapVueIcons } from 'bootstrap-vue';
import VueTimeago from 'vue-timeago';

Vue.use(BootstrapVue);
Vue.use(BootstrapVueIcons);
Vue.use(VueTimeago, { locale: 'en' });
Vue.config.productionTip = false;

new Vue({
  router,
  render: h => h(App),
  name: 'gunk',
  mixins: [ControlsHider],
  data () {
    return {
      channels: {},
      recentChannels: [],
      rtcSelected: localStorage.getItem('playerType') === 'RTC',
      lowLatency: localStorage.getItem('lowLatency') !== 'false',
      user: {
        id: null,
        username: null,
        discriminator: null,
        avatar: null
      }
    };
  },
  methods: {
    updateChannels () {
      axios.get('/channels.json')
        .then(response => {
          this.channels = response.data.channels;
          this.recentChannels = response.data.recent;
        });
    },
    updateChannel (ch) {
      const newch = Object.assign({}, this.channels);
      newch[ch.name] = ch;
      this.channels = newch;
    },
    updateUser () {
      axios.get('/oauth2/user')
        .then(response => { this.user = response.data; });
    },
    doLogin () {
      window.location.href = '/oauth2/initiate';
    },
    doLogout () {
      axios.post('/oauth2/logout')
        .then(() => {
          this.user.id = '';
          this.updateUser();
        });
    },
    navChannel (name) {
      return { name: 'watch', params: { channel: name } };
    },
    pushChannel (name) {
      this.$router.push(this.navChannel(name));
    }
  },
  computed: {
    loggedIn () { return this.user.id !== null && this.user.id !== ''; },
    liveChannels () {
      const live = [];
      for (const ch of Object.values(this.channels)) {
        if (ch.live) {
          live.push(ch.name);
        }
      }
      return live.sort();
    }
  },
  watch: {
    rtcSelected (rtcSelected) {
      localStorage.setItem('playerType', rtcSelected ? 'RTC' : 'HLS');
    },
    lowLatency (lowLatency) {
      localStorage.setItem('lowLatency', lowLatency);
    }
  },
  mounted () {
    this.updateChannels();
    this.updateUser();
    // this.chinterval = window.setInterval(this.updateChannels, 5000);
    this.userinterval = window.setInterval(this.updateUser, 300000);
    this.ws = new WSSession(location);
    this.ws.onChannel = (ch) => this.updateChannel(ch);
  },
  beforeDestroy () {
    this.ws.close();
    // window.clearInterval(this.chinterval);
    window.clearInterval(this.userinterval);
  }
}).$mount('#app');
