<template>
  <div class="home">
    <div v-for="name in channels.recent" :key="name" class="channel-card">
      <router-link :to="navChannel(name)">
        <img :src="channels.channels[name].thumb" />
        <div v-if="!channels.channels[name].live" class="channel-shade">
          OFFLINE
        </div>
        <div class="channel-card-title">
          <h1 :class="{ 'long-title': name.length > 20 }">{{ name }}</h1>
          <div class="channel-status">
            <span v-if="channels.channels[name].live" class="channel-live">
              LIVE
              <b-icon-eye-fill />
              {{ channels.channels[name].viewers }}
            </span>
            <time-ago
              v-else
              class="channel-notlive"
              :since="channels.channels[name].last"
              :auto-update="60"
            />
          </div>
        </div>
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useChannelsStore } from "@/stores/channels";
import { BIconEyeFill } from "bootstrap-icons-vue";
import TimeAgo from "@/components/TimeAgo";

const channels = useChannelsStore();

function navChannel(name: string) {
  return { name: "watch", params: { channel: name } };
}
</script>

<style>
.home {
  padding-top: 1rem;
  background: #111;
  min-height: calc(100vh - 48px);
}

/* home channel cards */
.channel-card {
  display: inline-block;
  position: relative;
  background: black;
  margin: 1rem 1rem 0;
  border: 1px solid black;
  width: 402px;
  color: white;
}
.channel-card a {
  color: white;
}
.channel-card a:hover {
  text-decoration: none;
}
.channel-card a:focus {
  outline: none;
}
.channel-card > img {
  width: 400px;
  min-height: 225px;
}

.channel-shade {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: #000c;
  color: white;
  overflow: hidden;

  padding: 100px 30px;
  font-size: 32px;
  letter-spacing: 35px;
}
.channel-card-title {
  position: relative;
  width: 100%;
  height: 1.5rem;
}
.channel-card-title h1 {
  margin: 0 0 0 0.5rem;
  font-size: 200%;
  font-family: "Montserrat", sans-serif;
  position: absolute;
  bottom: 0;
  width: 290px;
  max-height: 200px;
  overflow: hidden;
  text-shadow: 3px 3px 2px black;
}
.channel-card-title h1.long-title {
  font-size: 130%;
  width: 390px;
}
.channel-status {
  position: absolute;
  right: 0.2rem;
  bottom: 0;
  background-color: #000c;
}
.channel-live {
  font-weight: bold;
  color: red;
}
.channel-live img {
  margin-left: 0.3rem;
  width: 1rem;
  height: 1rem;
  vertical-align: -10%;
}
.channel-notlive {
  color: #777;
  font-style: italic;
  font-size: 90%;
}
</style>
