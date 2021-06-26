import Vue from 'vue';
import App from './app.vue';
import router from './router';
import ControlsHider from './components/controls-hider.vue';
import { APIMixin } from './api';
import axios from 'axios';

import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-vue/dist/bootstrap-vue.css';
import { BootstrapVue, BootstrapVueIcons } from 'bootstrap-vue';
import VueTimeago from 'vue-timeago';
import Component, { mixins } from 'vue-class-component';

Vue.use(BootstrapVue);
Vue.use(BootstrapVueIcons);
Vue.use(VueTimeago, { locale: 'en' });
Vue.config.productionTip = false;

@Component({
  watch: {
    rtcSelected (rtcSelected) {
      localStorage.setItem('playerType', rtcSelected ? 'RTC' : 'HLS');
    },
    lowLatency (lowLatency) {
      localStorage.setItem('lowLatency', lowLatency);
    }
  },
})
export default class Gunk extends mixins(ControlsHider, APIMixin) {
  rtcSelected = localStorage.getItem('playerType') === 'RTC';
  lowLatency = localStorage.getItem('lowLatency') !== 'false';
  readonly siteName = process.env.VUE_APP_SITE_NAME;

  mounted () {
    this.api.connect();
  }
  beforeDestroy () {
    this.api.disconnect();
  }
}

new Gunk({
  router,
  render: h => h(App),
}).$mount('#app');
