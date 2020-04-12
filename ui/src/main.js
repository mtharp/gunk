import Vue from 'vue';
import App from './app.vue';
import router from './router';
import ControlsHider from './components/controls-hider.vue';
import axios from 'axios';
import './assets/site.css';

import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-vue/dist/bootstrap-vue.css';
import BootstrapVue from 'bootstrap-vue';
import VueTimeago from 'vue-timeago';

Vue.use(BootstrapVue);
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
      serverTimeBase: null,
      rtcSelected: localStorage.getItem('playerType') === 'RTC',
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
          this.serverTimeBase = response.data.time - performance.now();
        });
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
    serverTime () {
      if (this.serverTimeBase === null) {
        return null;
      }
      return this.serverTimeBase + performance.now();
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
    }
  },
  mounted () {
    this.updateChannels();
    this.updateUser();
    this.chinterval = window.setInterval(this.updateChannels, 5000);
    this.userinterval = window.setInterval(this.updateUser, 300000);
  },
  beforeDestroy () {
    window.clearInterval(this.chinterval);
    window.clearInterval(this.userinterval);
  }
}).$mount('#app');
