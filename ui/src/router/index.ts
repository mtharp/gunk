import { createRouter, createWebHistory } from "vue-router";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  linkActiveClass: "active",
  routes: [
    {
      path: "/",
      name: "home",
      component: () => import("@/views/home-view.vue"),
    },
    {
      path: "/mychannels",
      name: "mychannels",
      component: () => import("@/views/my-channels.vue"),
    },
    {
      path: "/watch/:channel",
      name: "watch",
      component: () => import("@/views/watch-view.vue"),
      props: true,
    },
  ],
});

export default router;
