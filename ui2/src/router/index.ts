import { createRouter, createWebHistory } from "vue-router";
import HomeView from "@/views/HomeView.vue";
import MyChannels from "@/views/MyChannels.vue";
import WatchView from "@/views/WatchView.vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  linkActiveClass: "active",
  routes: [
    {
      path: "/",
      name: "home",
      component: HomeView,
    },
    {
      path: "/mychannels",
      name: "mychannels",
      component: MyChannels,
    },
    {
      path: "/watch/:channel",
      name: "watch",
      component: WatchView,
      props: true,
    },
  ],
});

export default router;
