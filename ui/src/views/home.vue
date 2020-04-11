<template>
  <div class="home">
    <div v-for="name in $root.recentChannels" :key="name" class="channel-card">
      <router-link :to="$root.navChannel(name)">
        <img :src="$root.channels[name].thumb" />
        <div v-if="!$root.channels[name].live" class="channel-shade">OFFLINE</div>
        <div class="channel-card-title">
          <h1>{{name}}</h1>
          <div class="channel-status">
            <span v-if="$root.channels[name].live" class="channel-live">
              LIVE
              <img src="/eye-solid.svg" />
              {{$root.channels[name].viewers}}
            </span>
            <timeago
              v-if="!$root.channels[name].live"
              class="channel-notlive"
              :datetime="$root.channels[name].last"
              :auto-update="60"
            />
          </div>
        </div>
      </router-link>
    </div>
  </div>
</template>

<script>
export default {
  name: "home"
};
</script>
