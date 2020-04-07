<template>
  <div class="w-100 h-100">
    <video
      ref="video"
      class="w-100 h-100"
      muted
      @click="videoClicked"
      @play="playing = true"
      @pause="playing = false; isLive = false"
      @mouseover="mouseMovedAt = $event.timeStamp"
      @mousemove="mouseMovedAt = $event.timeStamp"
      @mouseout="mouseMovedAt = null"
    ></video>
    <div :class="{'controls-region':true, 'controls-show': mouseMovedAt !== null}">
      <div class="controls">
        <div class="d-flex justify-content-start">
          <button
            @click="playPause"
            class="big-button"
            v-b-tooltip.hover
            :title="playing ? 'Pause' : 'Resume playing'"
          >
            <b-icon-play-fill v-if="!playing" />
            <b-icon-pause-fill v-if="playing" />
          </button>
          <div class="mute-and-vol d-flex">
            <button
              @click="toggleMute"
              class="big-button"
              v-b-tooltip.hover
              :title="isMuted ? 'Unmute' : 'Mute'"
            >
              <b-icon-volume-mute-fill v-if="isMuted" />
              <b-icon-volume-up-fill v-if="!isMuted" />
            </button>
            <div class="volume">
              <vue-slider
                v-model="volume"
                drag-on-click
                tooltip="none"
                :min="0"
                :max="1"
                :interval="0.01"
                @change="setVolume"
              />
            </div>
          </div>
        </div>
        <div class="d-flex justify-content-end">
          <div
            v-show="isLive && latency > 0"
            class="latency"
            v-b-tooltip.hover
            title="Current delay to live stream"
          >
            <b-icon-clock-history />
            {{latency.toFixed(1)}}s
          </div>
          <button
            @click="seekLive"
            :class="['live-button', isLive ? 'text-primary' : 'text-secondary']"
            v-b-tooltip.hover
            title="Jump to the latest point in the stream"
          >
            <b-icon-lightning-fill v-show="isLive" />
            <b-icon-lightning v-show="!isLive" />
            {{isLive ? "LIVE": "Not Live"}}
          </button>

          <b-dropdown dropup right no-caret>
            <template v-slot:button-content>
              <b-icon-gear-fill />
            </template>
            <b-dropdown-item :href="hlsURL">
              <img src="/vlc.png" />
              Watch in VLC
            </b-dropdown-item>
            <!-- <b-dropdown-form>
              <b-form-checkbox v-model="useRTC">Use WebRTC</b-form-checkbox>
            </b-dropdown-form>-->
          </b-dropdown>
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
import VueSlider from "vue-slider-component";
import "vue-slider-component/theme/default.css";
import { restoreVolume, HLSPlayer } from "../player.js";

export default {
  name: "hls-player",
  props: ["channel"],
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
      playing: false,
      isLive: true,
      hasFullscreen: document.fullscreenEnabled,
      isFullscreen: document.fullscreen,
      isMuted: true,
      volume: 0,
      mouseMovedAt: null,
      timer: null,
      latency: -1,
      targetLatency: 1,
      useRTC: false,
      hlsURL: null
    };
  },
  methods: {
    playPause(ev) {
      // toggle play or pause
      ev.target.blur();
      const v = this.$refs.video;
      if (this.playing) {
        v.pause();
        this.isLive = false;
        this.playing = false;
      } else {
        v.playbackRate = 1;
        v.play();
      }
    },
    videoClicked() {
      if (!this.playing) {
        // seek to end and play
        return this.seekLive();
      } else {
        // give touch devices a way to access controls by tapping player
        this.mouseMovedAt = performance.now() + 2000;
      }
    },
    fragments() {
      const hls = this.player.hls;
      if (!hls || !hls.levels) {
        return null;
      }
      const lev = hls.levels[hls.currentLevel];
      if (!lev) {
        return null;
      }
      return lev.details.fragments;
    },
    seekLive(ev) {
      // seek to end and play
      if (ev) {
        ev.target.blur();
      }
      const v = this.$refs.video;
      const fragments = this.fragments();
      if (fragments === null) {
        return;
      }
      if (fragments.length >= 3) {
        const tail = fragments[fragments.length - 1];
        if (tail.prefetch) {
          v.currentTime = tail.appendedPTS - this.targetLatency;
        } else {
          v.currentTime = tail.start - this.targetLatency;
        }
      }
      v.playbackRate = 1;
      v.play().then(() => {
        this.isLive = true;
      });
    },
    updateLatency() {
      const v = this.$refs.video;
      const fragments = this.fragments();
      const ts = this.$root.serverTime();
      if (ts === null || fragments === null || fragments.length < 2) {
        return;
      }
      for (let i = fragments.length - 2; i >= 0; i--) {
        let f = fragments[i];
        if (f.programDateTime && !f.prefetch) {
          const deltaDTS = v.currentTime - f.start;
          const deltaDate = ts - f.programDateTime;
          this.latency = deltaDate / 1000 - deltaDTS;
          break;
        }
      }
    },
    toggleFullscreen(ev) {
      ev.target.blur();
      if (this.isFullscreen) {
        document.exitFullscreen();
        this.isFullscreen = false;
      } else {
        this.$el.requestFullscreen().then(() => {
          this.isFullscreen = true;
        });
      }
    },
    toggleMute(ev) {
      ev.target.blur();
      this.isMuted = !this.isMuted;
      this.$refs.video.muted = this.isMuted;
    },
    setVolume() {
      const v = this.$refs.video;
      v.volume = this.volume;
      if (this.volume == 0) {
        this.isMuted = true;
        v.muted = true;
      } else if (this.isMuted || v.muted) {
        this.isMuted = false;
        v.muted = false;
      }
    },
    onTimer() {
      // hide controls if mouse isn't moving
      if (
        this.mouseMovedAt === null ||
        performance.now() - this.mouseMovedAt >= 1000
      ) {
        this.mouseMovedAt = null;
      }
      this.updateLatency();
    }
  },
  watch: {
    mouseMovedAt(v) {
      if (v !== null) {
        this.$root.$el.setAttribute("data-shownav", "show");
      } else {
        this.$root.$el.removeAttribute("data-shownav");
      }
    }
  },
  mounted() {
    this.$root.$el.setAttribute("data-hidenav", "hidenav");
    let video = this.$refs.video;
    video.poster = this.$root.currentChannel().thumb;
    this.$nextTick(() => {
      restoreVolume(video);
      this.isMuted = video.muted;
      this.volume = video.volume;
    });
    this.hlsURL = "/hls/" + encodeURIComponent(this.channel) + "/index.m3u8";
    this.player = new HLSPlayer(video, this.hlsURL);
    this.timer = window.setInterval(this.onTimer, 500);
  },
  beforeDestroy() {
    this.$root.$el.removeAttribute("data-hidenav");
    this.player.destroy();
    window.clearTimeout(this.timer);
  }
};
</script>
