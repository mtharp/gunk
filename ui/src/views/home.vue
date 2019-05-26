<template>
  <div class="home">
    <div
      v-for="ch in $root.channels" :key="ch.name"
      class="channel-card"
      >
      <router-link :to="$root.navChannel(ch)">
        <img :src="ch.thumb" />
        <div v-if="!ch.live" class="channel-shade">OFFLINE</div>
        <div class="channel-card-title">
          <h1>{{ch.name}}</h1>
          <div class="channel-status">
            <span v-if="ch.live" class="channel-live">LIVE</span>
            <timeago v-if="!ch.live" class="channel-notlive" :datetime="ch.last" :auto-update="60" />
          </div>
        </div>
      </router-link>
    </div>
  </div>
</template>

<script>
export default {
  name: 'home',
}
</script>
