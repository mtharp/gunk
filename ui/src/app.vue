<template>
  <div id="app" :class="{'is-tabbing': $root.isTabbing}">
    <b-navbar toggleable="lg" type="dark" ref="nav" :class="$root.hiddenControlClasses">
      <b-navbar-brand to="/">
        <img src="/cheese.png" width="58" height="40" alt="cheese" />
        gunk
      </b-navbar-brand>
      <b-navbar-toggle target="nav_collapse" />
      <b-collapse is-nav id="nav_collapse">
        <b-navbar-nav>
          <b-nav-item to="/">Home</b-nav-item>
          <b-nav-item to="/mychannels" v-if="$root.loggedIn">My Channels</b-nav-item>
          <b-nav-text>&bull; Watch:</b-nav-text>
          <b-nav-item
            v-for="name in $root.liveChannels"
            :key="name"
            :to="$root.navChannel(name)"
          >{{name}}</b-nav-item>
        </b-navbar-nav>
        <b-navbar-nav class="ml-auto">
          <b-nav-form v-if="!$root.loggedIn">
            <b-button size="sm" class="mr-2" @click.prevent="$root.doLogin">Login</b-button>
          </b-nav-form>
          <b-nav-form v-if="$root.loggedIn">
            <b-img
              :src="$root.user.avatar"
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
  width: 100%;
  height: 48px;
  z-index: 1;
  top: 0;
}
nav .show,
nav .collapsing {
  background-color: #002218;
  padding: 1rem;
}
nav .btn-sm img {
  height: 1rem;
  width: 1rem;
  margin-right: 0.25rem;
}
.navbar-brand img {
  margin-right: 0.5rem;
}
</style>
