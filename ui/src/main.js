import Vue from 'vue';
import App from './app.vue';
import router from './router';
import axios from 'axios';
import WSSession from './ws.js';

import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-vue/dist/bootstrap-vue.css';
import BootstrapVue from 'bootstrap-vue';

import './assets/site.css';

import VueTimeago from 'vue-timeago';

Vue.use(BootstrapVue);
Vue.use(VueTimeago, { locale: 'en' });
Vue.config.productionTip = false;

new Vue({
  router,
  render: h => h(App),
  data () {
    let initialPlayerType = window.localStorage.getItem('playerType');
    if (initialPlayerType !== 'RTC') {
      initialPlayerType = 'HLS';
    }
    return {
      channels: {},
      user: {
        id: null,
        username: null,
        discriminator: null,
        avatar: null
      },
      showStreamInfo: false,
      playerType: initialPlayerType
    };
  },
  methods: {
    updateChannels () {
      axios.get('/channels.json')
        .then(response => { this.channels = response.data; });
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
    popVLC () {
      window.location.href = '/live/' + encodeURIComponent(this.$route.params.channel) + '.m3u8';
    },
    currentChannel () {
      for (const ch of Object.values(this.channels)) {
        if (ch.name === this.$route.params.channel) {
          return ch;
        }
      }
      return null;
    },
    serverTime () {
      if (this.ws.tsBase === null) {
        return null;
      }
      return this.ws.tsBase + performance.now();
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
  mounted () {
    this.updateChannels();
    this.updateUser();
    this.chinterval = window.setInterval(this.updateChannels, 1000);
    this.userinterval = window.setInterval(this.updateUser, 300000);
    this.unwatch = this.$watch('playerType', v => window.localStorage.setItem('playerType', v));
    this.ws = new WSSession(location);
  },
  beforeDestroy () {
    this.ws.close();
    window.clearInterval(this.chinterval);
    window.clearInterval(this.userinterval);
    this.unwatch();
  }
}).$mount('#app');
