import Vue from 'vue'
import App from './app.vue'
import router from './router'
import axios from 'axios';

import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-vue/dist/bootstrap-vue.css'
import BootstrapVue from 'bootstrap-vue'
Vue.use(BootstrapVue)

import './assets/site.css'

import VueTimeago from 'vue-timeago'
Vue.use(VueTimeago, {locale: 'en'})

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
    liveChannels: null,
    user: {
      id: null,
      username: null,
      discriminator: null,
      avatar: null,
    },
    showStreamInfo: false,
    playerType: "HLS",
  },
  methods: {
    updateChannels() {
      axios.get("/channels.json")
        .then(response => {
          let live = new Array();
          for (let ch of response.data) {
            if (ch.live) {
              live.push(ch)
            }
          }
          this.channels = response.data
          this.liveChannels = live
        })
    },
    updateUser() {
      axios.get("/oauth2/user")
        .then(response => this.user = response.data)
    },
    doLogin() {
      window.location.href = "/oauth2/initiate"
    },
    doLogout() {
      axios.post("/oauth2/logout")
        .then(() => {
          this.user.id = ""
          this.updateUser()
        })
    },
    navChannel(ch) {
      return {name: 'watch', params: {channel: ch.name}}
    },
  },
  computed: {
    loggedIn() { return this.user.id !== null && this.user.id !== "" },
    avatarURL() {
      return "/avatars/" + this.user.id + "/" + this.user.avatar + ".png?size=32";
    },
  },
  mounted() {
    this.updateChannels();
    this.updateUser();
    this.chinterval = window.setInterval(this.updateChannels, 5000)
    this.userinterval = window.setInterval(this.updateUser, 300000)
  },
  beforeDestroy() {
    window.clearInterval(this.chinterval)
    window.clearInterval(this.userinterval)
  }
}).$mount('#app')
