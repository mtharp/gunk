import Vue from 'vue';
import Router from 'vue-router';

Vue.use(Router);

export default new Router({
  mode: 'history',
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('./views/home.vue')
    },
    {
      path: '/mychannels',
      name: 'mychannels',
      component: () => import('./views/mychannels.vue')
    },
    {
      path: '/watch/:channel',
      name: 'watch',
      component: () => import('./views/watch.vue'),
      props: true
    }
  ]
});
