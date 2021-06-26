<template>
  <div id="app" :class="{ 'is-tabbing': $root.isTabbing }">
    <b-navbar type="dark" ref="nav" :class="$root.hiddenControlClasses">
      <b-navbar-brand to="/">
        <img src="/favicon96.png" alt="a dapper fellow" />
        {{ $root.siteName }}
      </b-navbar-brand>
      <b-navbar-nav>
        <b-nav-item to="/" v-b-tooltip title="Home"
          ><b-icon-house-fill />
        </b-nav-item>
        <b-nav-item
          to="/mychannels"
          v-if="loggedIn"
          v-b-tooltip.hover
          title="Create a Channel"
          ><b-icon-pencil-fill
        /></b-nav-item>
        <b-nav-form class="ml-2">
          <b-avatar
            v-for="name in liveChannels"
            button
            rounded
            @click="pushChannel(name)"
            :key="name"
            :src="thumb(name)"
            size="32px"
            class="ml-2"
            v-b-tooltip.hover
            :title="'Watch ' + name"
          ></b-avatar>
        </b-nav-form>
      </b-navbar-nav>
      <b-navbar-nav class="ml-auto">
        <b-nav-form v-if="!loggedIn">
          <b-button size="sm" class="mr-2" @click.prevent="userinfo.login"
            >Login</b-button
          >
        </b-nav-form>
        <b-nav-form v-if="loggedIn">
          <b-avatar
            :src="userinfo.avatar"
            size="32px"
            alt="your avatar"
            class="mr-3"
            button
            v-b-tooltip.hover
            :title="'Logged in as ' + userinfo.account"
          />
          <b-button size="sm" class="mr-sm-2" @click.prevent="userinfo.logout"
            >Logout</b-button
          >
        </b-nav-form>
      </b-navbar-nav>
    </b-navbar>
    <router-view />
  </div>
</template>

<script lang="ts">
import Component, { mixins } from 'vue-class-component';
import { APIMixin } from './api';
import Gunk from "./main";
import channels from './store/channels';
import userinfo from './store/userinfo';

@Component
export default class App extends mixins(APIMixin) {
  $root!: Gunk;

  navChannel (name: string) {
    return { name: 'watch', params: { channel: name } };
  }

  pushChannel (name: string) {
    this.$router.push(this.navChannel(name));
  }

  get userinfo() { return userinfo; }
  get loggedIn () { return !!userinfo.user.id }
    get liveChannels () {
        const live = [];
        for (const ch of Object.values(channels.channels)) {
            if (ch.live) {
                live.push(ch.name);
            }
        }
        return live.sort();
    }

  thumb(name: string) {
    return channels.channels[name].thumb
  }
}
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
