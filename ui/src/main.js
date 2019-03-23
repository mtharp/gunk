import Vue from 'vue'
import App from './app.vue'
import router from './router'
import axios from 'axios';

import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-vue/dist/bootstrap-vue.css'
import BootstrapVue from 'bootstrap-vue'
Vue.use(BootstrapVue)

import HLSPlayer from './components/hlsplayer.vue'
Vue.component('hls-player', HLSPlayer)
import RTCPlayer from './components/rtcplayer.vue'
Vue.component('rtc-player', RTCPlayer)

Vue.config.productionTip = false
Vue.config.ignoredElements = ["video-js"]

new Vue({
  router,
  render: h => h(App),
  data: {
    channels: null,
    useRTC: false,
  },
  methods: {
    updateChannels() {
      axios.get("/channels.json")
        .then(response => this.channels = response.data);
    },
  },
  mounted() {
    this.updateChannels();
    this.chinterval = window.setInterval(this.updateChannels, 5000);
  },
}).$mount('#app')
