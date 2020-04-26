<template>
  <div class="w-100 h-100">
    <video
      ref="video"
      class="w-100 h-100"
      :poster="initPoster"
      @play="playing = true; volume = $refs.video.volume"
      @pause="playing = false"
      @ended="playing = false"
      @volumechange="muted = $refs.video.muted"
      @click="$root.controlsTouched"
    ></video>
    <div :class="classes">
      <div class="d-flex justify-content-start align-items-center">
        <!-- play/pause -->
        <button
          @click="$refs.video.play()"
          class="big-button"
          v-if="!playing"
          v-b-tooltip.hover
          title="Play (k)"
        >
          <b-icon-play-fill />
        </button>
        <button
          @click="$refs.video.pause()"
          class="big-button"
          v-else
          v-b-tooltip.hover
          title="Pause (k)"
        >
          <b-icon-pause-fill />
        </button>
        <!-- mute and volume -->
        <div class="mute-and-vol d-flex align-items-center">
          <button
            @click="$refs.video.muted = true"
            class="big-button"
            v-if="!muted"
            v-b-tooltip.hover
            title="Mute (m)"
          >
            <b-icon-volume-up-fill />
          </button>
          <button
            @click="$refs.video.muted = false"
            class="big-button"
            v-else
            v-b-tooltip.hover
            title="Unmute (m)"
          >
            <b-icon-volume-mute-fill />
          </button>
          <div :class="volumeClasses">
            <vue-slider
              v-model="volume"
              drag-on-click
              tooltip="none"
              :min="0"
              :max="1"
              :interval="0.01"
              v-b-tooltip.hover
              title="Volume (up/down)"
              @change="$refs.video.volume = volume; $refs.video.muted = false"
            />
          </div>
        </div>
      </div>
      <div class="d-flex justify-content-end align-items-center">
        <!-- latency and seek to live -->
        <template v-if="!$parent.rtcActive">
          <div
            v-if="playing"
            class="controls-latency"
            v-b-tooltip.hover
            title="Current delay to live stream"
          >
            <b-icon-clock-history />
            {{latency ? latency.toFixed(1) + 's' : '-'}}
          </div>
          <button
            @click="seekLive"
            v-if="atTail"
            class="controls-live text-primary"
            v-b-tooltip.hover
            title="Jump to the latest point in the stream (j)"
          >
            <b-icon-lightning-fill />
            <span>LIVE</span>
          </button>
          <button
            @click="seekLive"
            v-else
            class="controls-live text-secondary"
            v-b-tooltip.hover
            title="Jump to the latest point in the stream (j)"
          >
            <b-icon-lightning />
            <small>Skip to Live</small>
          </button>
        </template>
        <div
          v-if="playing && $parent.rtcActive"
          class="controls-rtclabel text-success"
          v-b-tooltip.hover
          title="Real-time stream has near-zero latency"
        >
          <b-icon-soundwave />WebRTC
        </div>
        <!-- viewers -->
        <div class="controls-viewers">
          <b-icon-eye-fill />
          {{ch.viewers}}
        </div>
        <!-- settings menu -->
        <b-dropdown dropup right no-caret>
          <template v-slot:button-content>
            <b-icon-gear-fill />
          </template>
          <!-- for some reason chrome won't open a .m3u8 file directly so don't show the playlist link -->
          <b-dropdown-item :href="playlistURL" @click="$refs.video.pause()" v-if="!isWebKit">
            <img src="/vlc.png" />
            Watch in VLC
          </b-dropdown-item>
          <b-dropdown-item :href="ch.live_url" @click.prevent="copyVLC">
            <b-icon-clipboard-data />Copy VLC URL
          </b-dropdown-item>
          <b-dropdown-form v-if="showCopyVLC">
            <b-form-input ref="copyVLCInput" :value="ch.live_url" readonly />
          </b-dropdown-form>

          <b-dropdown-form>
            <b-form-checkbox v-model="$root.rtcSelected" :disabled="!ch.rtc">Use WebRTC</b-form-checkbox>
          </b-dropdown-form>
        </b-dropdown>
        <!-- fullscreen -->
        <button
          v-if="hasFullscreen"
          @click="toggleFullscreen"
          v-b-tooltip.hover
          :title="isFullscreen ? 'Exit Fullscreen (f)' : 'Fullscreen (f)'"
        >
          <b-icon-fullscreen v-if="!isFullscreen" />
          <b-icon-fullscreen-exit v-if="isFullscreen" />
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import VueSlider from "vue-slider-component";
import "vue-slider-component/theme/default.css";
import {
  BIconClipboardData,
  BIconClockHistory,
  BIconEyeFill,
  BIconFullscreen,
  BIconFullscreenExit,
  BIconGearFill,
  BIconLightning,
  BIconLightningFill,
  BIconPauseFill,
  BIconPlayFill,
  BIconSoundwave,
  BIconVolumeMuteFill,
  BIconVolumeUpFill
} from "bootstrap-vue";

import { HLSPlayer, RTCPlayer } from "../player.js";

export default {
  name: "player",
  props: ["ch", "rtcActive"],
  components: {
    BIconClipboardData,
    BIconClockHistory,
    BIconEyeFill,
    BIconFullscreen,
    BIconFullscreenExit,
    BIconGearFill,
    BIconLightning,
    BIconLightningFill,
    BIconPauseFill,
    BIconPlayFill,
    BIconSoundwave,
    BIconVolumeMuteFill,
    BIconVolumeUpFill,
    VueSlider
  },
  data() {
    return {
      hasFullscreen: document.fullscreenEnabled,
      isFullscreen: document.fullscreenElement !== null,
      isWebKit: navigator.userAgent.indexOf("WebKit") >= 0,
      playing: false,
      initPoster: this.ch.thumb,
      muted: true,
      volume: 0,

      latency: 0,
      latencyTimer: null,
      keyTimer: null,
      keyPressed: false,
      atTail: false,
      showCopyVLC: false
    };
  },
  created() {
    document.addEventListener("fullscreenchange", this.onFullscreen);
    this.$root.startHidingControls(this.$vnode.key);
  },
  mounted() {
    this.keyTimer = null;
    document.addEventListener("keydown", this.onKey);
    let video = this.$refs.video;
    if (this.rtcActive) {
      this.player = new RTCPlayer(video, this.sdpURL);
      this.atTail = true;
    } else {
      this.player = new HLSPlayer(video, this.hlsURL);
      this.latencyTimer = window.setInterval(this.updateLatency, 1000);
    }
  },
  beforeDestroy() {
    document.removeEventListener("keydown", this.onKey);
    document.removeEventListener("fullscreenchange", this.onFullscreen);
    if (this.keyTimer) {
      window.clearTimeout(this.keyTimer);
    }
    if (this.latencyTimer) {
      window.clearInterval(this.latencyTimer);
    }
    this.$root.stopHidingControls(this.$vnode.key);
    if (this.player) {
      this.player.destroy();
      this.player = null;
    }
  },
  computed: {
    classes() {
      return Object.assign({ controls: true }, this.$root.hiddenControlClasses);
    },
    volumeClasses() {
      return { volume: true, "key-pressed": this.keyPressed };
    },
    hlsURL() {
      return "/hls/" + encodeURIComponent(this.ch.name) + "/index.m3u8";
    },
    sdpURL() {
      return "/sdp/" + encodeURIComponent(this.ch.name);
    },
    playlistURL() {
      return "/live/" + encodeURIComponent(this.ch.name) + ".m3u8";
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
    onKey(ev) {
      this.$root.controlsTouched();
      this.keyPressed = true;
      if (this.keyTimer) {
        window.clearTimeout(this.keyTimer);
      }
      this.keyTimer = window.setTimeout(() => {
        this.keyPressed = false;
        this.keyTimer = null;
      }, 3000);
      switch (ev.key) {
        case "f":
          this.toggleFullscreen();
          break;
        case "k":
          if (this.playing) {
            this.$refs.video.pause();
          } else {
            this.$refs.video.play();
          }
          break;
        case "j":
          this.seekLive();
          break;
        case "m":
          this.muted = !this.muted;
          this.$refs.video.muted = this.muted;
          break;
        case "ArrowDown":
          if (this.volume > 0.05) {
            this.volume -= 0.05;
          } else {
            this.volume = 0;
          }
          this.$refs.video.volume = this.volume;
          this.$refs.video.muted = false;
          break;
        case "ArrowUp":
          if (this.volume < 0.95) {
            this.volume += 0.05;
          } else {
            this.volume = 1;
          }
          this.$refs.video.volume = this.volume;
          this.$refs.video.muted = false;
          break;
      }
    },
    seekLive() {
      this.player.seekLive();
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
    },
    // copy VLC URL to clipboard
    copyVLC() {
      this.showCopyVLC = true;
      this.$nextTick(() => {
        this.$refs.copyVLCInput.select();
        document.execCommand("copy");
        this.$nextTick(() => {
          this.showCopyVLC = false;
          this.$bvToast.toast(
            "Open VLC, press Ctrl-N and paste to play the stream",
            {
              title: "Stream URL copied",
              isStatus: true,
              toaster: "b-toaster-bottom-right",
              autoHideDelay: 2000
            }
          );
        });
      });
    }
  }
};
</script>

<style>
/* player */
.player-box {
  width: 100vw;
  height: 100vh;
  background-color: black;
  position: absolute;
  top: 0;
  overflow: hidden;
}
.player-thumb {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center center;
}
.player-shade {
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

  font-size: 10vh;
  letter-spacing: 10vh;
  text-align: center;
  padding-top: 40vh;
}
/* controls */
.controls {
  background: #000d;
  width: 100vw;
  height: 2.3rem;
  position: absolute;
  bottom: 0;
  display: flex;
  justify-content: space-between;
}
.controls button {
  height: 1.8rem;
  min-width: 1.6rem;
  border: 0;
  background: transparent;
  color: white;
}
.controls button:hover {
  border-radius: 0.3rem;
  background: #fff2;
}
.controls button:focus {
  outline: none;
}
/* controls dropdown */
.controls .dropdown-menu {
  min-width: 12rem;
}
.controls .dropdown button {
  font-size: 0.75rem;
}
.controls .dropdown-item img {
  margin-left: -2px;
  width: 20px;
  height: 20px;
}
.controls .dropdown svg {
  margin-left: -2px;
  margin-right: 0.3rem;
}
/* controls elements */
.big-button {
  padding-top: 0.15rem;
}
.big-button svg {
  font-size: 1.5rem;
}
.controls-live {
  margin: 0 0.5rem;
}
.controls-live svg {
  font-size: 1rem;
}
.controls-latency {
  font-size: 0.85rem;
  color: #888;
}
.controls-rtclabel {
  margin-right: 0.75rem;
}
.controls-viewers {
  color: #b00;
}
.mute-and-vol .volume {
  opacity: 0;
  transition: 0.25s;
  height: 100%;
}
.mute-and-vol:hover .volume,
.mute-and-vol *:focus ~ .volume,
.is-tabbing .volume,
.volume.key-pressed {
  opacity: 1;
}
.volume {
  margin-left: 0.5rem;
  width: 100px;
}
</style>
