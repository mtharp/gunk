<template>
  <div class="w-100 h-100">
    <video
      ref="video"
      class="w-100 h-100"
      :poster="poster"
      @play="playing = true; volume = $refs.video.volume"
      @pause="playing = false"
      @ended="playing = false"
      @volumechange="muted = $refs.video.muted"
      @click="$root.controlsTouched"
    ></video>
    <div :class="$root.hiddenControlClasses">
      <div class="controls">
        <div class="d-flex justify-content-start">
          <!-- play/pause -->
          <button
            @click="$refs.video.play()"
            class="big-button"
            v-show="!playing"
            v-b-tooltip.hover
            title="Play"
          >
            <b-icon-play-fill />
          </button>
          <button
            @click="$refs.video.pause()"
            class="big-button"
            v-show="playing"
            v-b-tooltip.hover
            title="Pause"
          >
            <b-icon-pause-fill />
          </button>
          <!-- mute and volume -->
          <div class="mute-and-vol d-flex">
            <button
              @click="$refs.video.muted = true"
              class="big-button"
              v-show="!muted"
              v-b-tooltip.hover
              title="Mute"
            >
              <b-icon-volume-up-fill />
            </button>
            <button
              @click="$refs.video.muted = false"
              class="big-button"
              v-show="muted"
              v-b-tooltip.hover
              title="Unmute"
            >
              <b-icon-volume-mute-fill />
            </button>
            <div class="volume">
              <vue-slider
                v-model="volume"
                drag-on-click
                tooltip="none"
                :min="0"
                :max="1"
                :interval="0.01"
                @change="$refs.video.volume = volume; $refs.video.muted = false"
              />
            </div>
          </div>
        </div>
        <div class="d-flex justify-content-end">
          <!-- latency and seek to live -->
          <div
            v-if="playing"
            class="latency"
            v-b-tooltip.hover
            title="Current delay to live stream"
          >
            <b-icon-clock-history />
            {{latency ? latency.toFixed(1) + 's' : '-'}}
          </div>
          <button
            @click="seekLive"
            v-show="atTail"
            class="live-button text-primary"
            v-b-tooltip.hover
            title="Jump to the latest point in the stream"
          >
            <b-icon-lightning-fill />
            <span>LIVE</span>
          </button>
          <button
            @click="seekLive"
            v-show="!atTail"
            class="live-button text-secondary"
            v-b-tooltip.hover
            title="Jump to the latest point in the stream"
          >
            <b-icon-lightning />
            <small>Skip to Live</small>
          </button>
          <!-- settings menu -->
          <b-dropdown dropup right no-caret>
            <template v-slot:button-content>
              <b-icon-gear-fill />
            </template>
            <b-dropdown-item :href="liveURL">
              <img src="/vlc.png" />
              Watch in VLC
            </b-dropdown-item>
            <b-dropdown-form>
              <b-form-checkbox v-model="$parent.useRTC">Use WebRTC</b-form-checkbox>
            </b-dropdown-form>
          </b-dropdown>
          <!-- fullscreen -->
          <button
            v-if="hasFullscreen"
            @click="toggleFullscreen"
            v-b-tooltip.hover
            :title="isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'"
          >
            <b-icon-fullscreen v-if="!isFullscreen" />
            <b-icon-fullscreen-exit v-if="isFullscreen" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import VueSlider from "vue-slider-component";
import "vue-slider-component/theme/default.css";
import {
  BIconPlayFill,
  BIconPauseFill,
  BIconLightning,
  BIconLightningFill,
  BIconClockHistory,
  BIconFullscreen,
  BIconFullscreenExit,
  BIconVolumeMuteFill,
  BIconVolumeUpFill,
  BIconGearFill
} from "bootstrap-vue";

import { HLSPlayer } from "../player.js";

export default {
  name: "player",
  props: ["poster", "liveURL", "hlsURL", "sdpURL"],
  components: {
    BIconPlayFill,
    BIconPauseFill,
    BIconLightning,
    BIconLightningFill,
    BIconClockHistory,
    BIconFullscreen,
    BIconFullscreenExit,
    BIconVolumeMuteFill,
    BIconVolumeUpFill,
    BIconGearFill,
    VueSlider
  },
  data() {
    return {
      hasFullscreen: document.fullscreenEnabled,
      isFullscreen: document.fullscreenElement !== null,
      playing: false,
      muted: true,
      volume: 0,

      latency: 0,
      latencyTimer: null,
      atTail: false
    };
  },
  created() {
    document.addEventListener("fullscreenchange", this.onFullscreen);
    this.$root.startHidingControls();
  },
  mounted() {
    let video = this.$refs.video;
    if (this.hlsURL) {
      this.player = new HLSPlayer(video, this.hlsURL);
      this.latencytimer = window.setInterval(this.updateLatency, 1000);
    }
  },
  beforeDestroy() {
    document.removeEventListener("fullscreenchange", this.onFullscreen);
    if (this.latencytimer) {
      window.clearInterval(this.latencytimer);
    }
    this.$root.stopHidingControls();
    if (this.player) {
      this.player.destroy();
    }
  },
  methods: {
    onFullscreen() {
      this.isFullscreen = document.fullscreenElement !== null;
    },
    toggleFullscreen() {
      if (this.isFullscreen) {
        document.exitFullscreen();
      } else {
        this.$el.requestFullscreen();
      }
    },
    seekLive() {
      if (this.hlsURL && this.player !== null) {
        this.player.seekLive();
      } else {
        this.$refs.video.play();
      }
    },
    updateLatency() {
      if (!this.playing) {
        this.latency = 0;
      }
      const ts = this.$root.serverTime();
      const latency = this.player.latencyTo(ts);
      if (latency !== null) {
        [this.latency, this.atTail] = latency;
      }
    }
  }
};
</script>
