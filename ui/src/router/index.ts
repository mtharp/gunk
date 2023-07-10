import { createRouter, createWebHistory } from "vue-router";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  linkActiveClass: "active",
  routes: [
    {
      path: "/",
      name: "home",
      component: () => import("@/views/HomeView.vue"),
    },
    {
      path: "/mychannels",
      name: "mychannels",
      component: () => import("@/views/MyChannels.vue"),
    },
    {
      path: "/watch/:channel",
      name: "watch",
      component: () => import("@/views/WatchView.vue"),
      props: true,
    },
  ],
});

export default router;
