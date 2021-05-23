<template>
  <div id="app" :class="{ 'is-tabbing': $root.isTabbing }">
    <b-navbar type="dark" ref="nav" :class="$root.hiddenControlClasses">
      <b-navbar-brand to="/">
        <img src="/favicon96.png" alt="a dapper fellow" />
        gunk
      </b-navbar-brand>
      <b-navbar-nav>
        <b-nav-item to="/" v-b-tooltip title="Home"
          ><b-icon-house-fill />
        </b-nav-item>
        <b-nav-item
          to="/mychannels"
          v-if="$root.loggedIn"
          v-b-tooltip.hover
          title="Create a Channel"
          ><b-icon-pencil-fill
        /></b-nav-item>
        <b-nav-form class="ml-2">
          <b-avatar
            v-for="name in $root.liveChannels"
            button
            rounded
            @click="$root.pushChannel(name)"
            :key="name"
            :src="$root.channels[name].thumb"
            size="32px"
            class="ml-2"
            v-b-tooltip.hover
            :title="'Watch ' + name"
          ></b-avatar>
        </b-nav-form>
      </b-navbar-nav>
      <b-navbar-nav class="ml-auto">
        <b-nav-form v-if="!$root.loggedIn">
          <b-button size="sm" class="mr-2" @click.prevent="$root.doLogin"
            >Login</b-button
          >
        </b-nav-form>
        <b-nav-form v-if="$root.loggedIn">
          <b-avatar
            :src="$root.user.avatar"
            size="32px"
            alt="your avatar"
            class="mr-3"
            button
            v-b-tooltip.hover
            :title="
              'Logged in as ' +
              $root.user.username +
              '#' +
              $root.user.discriminator
            "
          />
          <b-button size="sm" class="mr-sm-2" @click.prevent="$root.doLogout"
            >Logout</b-button
          >
        </b-nav-form>
      </b-navbar-nav>
    </b-navbar>
    <router-view />
  </div>
</template>

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
