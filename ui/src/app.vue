<template>
  <div id="app">
    <b-navbar toggleable="lg" type="dark">
      <b-navbar-brand to="/">gunk</b-navbar-brand>
      <b-navbar-toggle target="nav_collapse" />
      <b-collapse is-nav id="nav_collapse">
        <b-navbar-nav>
          <b-nav-item to="/">Home</b-nav-item>
          <b-nav-item to="/mychannels" v-if="$root.loggedIn">My Channels</b-nav-item>
          <b-nav-text>&bull; Watch:</b-nav-text>
          <b-nav-item v-for="ch in $root.channels" :key="ch" :to="{name: 'watch', params: {channel: ch}}">{{ch}}</b-nav-item>
        </b-navbar-nav>
        <b-navbar-nav class="ml-auto">
          <b-nav-text v-if="$route.name == 'watch'">
            <b-button size="sm" class="mr-3" @click="$root.showStreamInfo = true">Info</b-button>
            <b-form-radio-group buttons size="sm" class="mr-3" v-model="$root.playerType">
              <b-form-radio value="HLS">HLS</b-form-radio>
              <b-form-radio value="RTC">RTC</b-form-radio>
            </b-form-radio-group>
          </b-nav-text>
          <b-nav-form v-if="!$root.loggedIn">
            <b-button size="sm" class="mr-2" @click.prevent="$root.doLogin">Login</b-button>
          </b-nav-form>
          <b-nav-form v-if="$root.loggedIn">
            <b-img
              :src="$root.avatarURL"
              width="32"
              height="32"
              rounded="circle"
              alt="your avatar"
              class="mr-3"
              />
            <b-button size="sm" class="mr-sm-2" @click.prevent="$root.doLogout">Logout</b-button>
          </b-nav-form>
        </b-navbar-nav>
      </b-collapse>
    </b-navbar>
    <router-view />
  </div>
</template>

<style>
nav {
  background-color: #043;
  height: 56px;
}
</style>
