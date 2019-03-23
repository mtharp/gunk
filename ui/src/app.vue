<template>
  <div id="app">
    <b-navbar toggleable="lg" type="dark">
      <b-navbar-brand to="/">gunk</b-navbar-brand>
      <b-navbar-toggle target="nav_collapse" />
      <b-collapse is-nav id="nav_collapse">
        <b-navbar-nav>
          <b-nav-item to="/">Home</b-nav-item>
          <b-nav-text>&bull;</b-nav-text>
          <b-nav-item v-for="ch in $root.channels" :key="ch" :to="{name: 'watch', params: {channel: ch}}">{{ch}}</b-nav-item>
        </b-navbar-nav>
        <b-navbar-nav class="ml-auto">
          <b-form-checkbox switch v-model="$root.useRTC" class="navbar-text">Zero-Latency Mode (WebRTC)</b-form-checkbox>
        </b-navbar-nav>
      </b-collapse>
    </b-navbar>
    <router-view />
  </div>
</template>

<style>
nav {
  background-color: #043;
}

</style>
