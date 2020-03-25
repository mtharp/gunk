import Vue from 'vue'
import Router from 'vue-router'
import Home from './views/home.vue'
import Watch from './views/watch.vue'

Vue.use(Router)

export default new Router({
  mode: 'history',
  base: process.env.BASE_URL,
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home
    },
    {
      path: '/mychannels',
      name: 'mychannels',
      component: () => import('./views/mychannels.vue')
    },
    {
      path: '/watch/:channel',
      name: 'watch',
      component: Watch,
      props: true,
    },
  ]
})
