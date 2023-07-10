<template>
  <div id="app" :class="controlsHider.rootClasses">
    <nav
      class="navbar navbar-expand-lg bg-dark navbar-dark text-light"
      :class="controlsHider.hiddenControlClasses"
    >
      <div class="container-fluid">
        <router-link class="navbar-brand" to="/">
          <img src="/favicon96.png" alt="a dapper fellow" />
          {{ siteName }}
        </router-link>
        <ul class="navbar-nav mr-auto flex-grow-1">
          <li class="nav-item">
            <router-link class="nav-link" to="/" title="Home"
              ><b-icon-house-fill
            /></router-link>
          </li>
          <li class="nav-item" v-if="loggedIn">
            <router-link
              class="nav-link"
              to="/mychannels"
              title="Create a channel"
              ><b-icon-pencil-fill
            /></router-link>
          </li>
        </ul>
        <ul class="navbar-nav ml-auto">
          <li class="nav-item" v-if="!loggedIn">
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              @click.prevent="userinfo.login"
            >
              Login
            </button>
          </li>
          <li class="nav-item" v-else>
            <img
              :src="userinfo.avatar"
              width="32"
              height="32"
              alt="your avatar"
              class="mr-3 rounded-circle"
              :title="'Logged in as ' + userinfo.account"
            />
            <button
              class="btn btn-secondary btn-sm mr-sm-2"
              @click.prevent="userinfo.logout"
            >
              Logout
            </button>
          </li>
        </ul>
      </div>
    </nav>
    <router-view />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { RouterView, RouterLink } from "vue-router";
import { BIconHouseFill, BIconPencilFill } from "bootstrap-icons-vue";
// import router from "./router";
import { useChannelsStore } from "./stores/channels";
import { useControlsHider } from "./stores/controls-hider";
import { usePreferences } from "./stores/preferences";
import { useUserInfoStore } from "./stores/userinfo";
import ws from "./ws";

const userinfo = useUserInfoStore();
const channels = useChannelsStore();
const controlsHider = useControlsHider();
const preferences = usePreferences();
const siteName = import.meta.env.VITE_APP_TITLE;

const loggedIn = computed(() => !!userinfo.id);
// const liveChannels = computed(() => {
//   const live = [];
//   for (const name in channels.channels) {
//     const ch = channels.channels[name];
//     if (ch.live) {
//       live.push(ch.name);
//     }
//   }
//   return live.sort();
// });

// function pushChannel(name: string) {
//   const route = { name: "watch", params: { channel: name } };
//   router.push(route);
// }

// function thumb(name: string) {
//   return channels.channels[name].thumb;
// }

onMounted(() => {
  preferences.fromLocalStorage();
  channels.refreshChannels();
  userinfo.refreshUserInfo();
  window.setInterval(() => userinfo.refreshUserInfo(), 300000);
  ws.onChannel = (ch) => channels.putChannel(ch);
  ws.open();
});
</script>

<style>
@import url("https://fonts.googleapis.com/css2?family=Montserrat&family=Special+Elite&display=swap");

nav {
  background-color: #133123;
  width: 100%;
  height: 48px;
  z-index: 1;
  top: 0;
}
nav .btn-sm img {
  height: 1rem;
  width: 1rem;
  margin-right: 0.25rem;
}
.navbar-brand {
  padding: 0 !important;
  font-family: "Special Elite", cursive;
}
.navbar-brand img {
  width: 64px;
  height: 64px;
  margin-left: -1rem;
}
@media (min-width: 992px) {
  .navbar-brand {
    padding: 0.3125rem 0;
  }
  .navbar-brand img {
    width: 96px;
    height: 96px;
    margin-top: 1.25rem;
  }
}
nav *:focus {
  outline: none;
}
</style>
