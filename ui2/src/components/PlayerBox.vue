<template>
  <div class="w-100 h-100" ref="container">
    <video
      ref="video"
      class="w-100 h-100"
      :poster="state.poster"
      @play="onPlay"
      @pause="state.playing = false"
      @ended="state.playing = false"
      @volumechange="onVolumeChange"
      @click="controlsHider.controlsTouched"
    ></video>
    <div :class="classes">
      <div class="d-flex justify-content-start align-items-center">
        <!-- play/pause -->
        <button
          @click="video!.play()"
          class="big-button"
          v-if="!state.playing"
          data-bs-toggle="tooltip"
          title="Play (k)"
        >
          <b-icon-play-fill />
        </button>
        <button
          @click="video!.pause()"
          class="big-button"
          v-else
          data-bs-toggle="tooltip"
          title="Pause (k)"
        >
          <b-icon-pause-fill />
        </button>
        <!-- mute and volume -->
        <div class="mute-and-vol d-flex align-items-center">
          <button
            @click="video!.muted = true"
            class="big-button"
            v-if="!state.muted"
            data-bs-toggle="tooltip"
            title="Mute (m)"
          >
            <b-icon-volume-up-fill />
          </button>
          <button
            @click="video!.muted = false"
            class="big-button"
            v-else
            data-bs-toggle="tooltip"
            title="Unmute (m)"
          >
            <b-icon-volume-mute-fill />
          </button>
          <div :class="volumeClasses">
            <vue-slider
              v-model="state.volume"
              drag-on-click
              tooltip="none"
              :min="0"
              :max="1"
              :interval="0.01"
              data-bs-toggle="tooltip"
              title="Volume (up/down)"
              @change="
                video!.volume = state.volume;
                video!.muted = false;
              "
            />
          </div>
        </div>
      </div>
      <div class="d-flex justify-content-end align-items-center">
        <!-- latency and seek to live -->
        <template v-if="!rtcActive">
          <div
            v-if="state.playing"
            class="controls-latency"
            data-bs-toggle="tooltip"
            title="Current delay to live stream"
          >
            <b-icon-skip-forward-fill v-if="state.catchingUp" />
            <b-icon-clock-history v-else />
            {{ state.latency ? state.latency.toFixed(1) + "s" : "-" }}
          </div>
          <button
            @click="seekLive"
            v-if="state.atTail"
            class="controls-live text-primary"
            data-bs-toggle="tooltip"
            title="Jump to the latest point in the stream (j)"
          >
            <b-icon-lightning-fill />
            <span>LIVE</span>
          </button>
          <button
            @click="seekLive"
            v-else
            class="controls-live text-secondary"
            data-bs-toggle="tooltip"
            title="Jump to the latest point in the stream (j)"
          >
            <b-icon-lightning />
            <small>Skip to Live</small>
          </button>
        </template>
        <div
          v-if="state.playing && rtcActive"
          class="controls-rtclabel text-success"
          data-bs-toggle="tooltip"
          title="Real-time stream has near-zero latency"
        >
          <b-icon-soundwave />WebRTC
        </div>
        <!-- viewers -->
        <div class="controls-viewers">
          <b-icon-eye-fill />
          {{ ch.viewers }}
        </div>
        <!-- settings menu -->
        <div class="dropdown dropup no-caret">
          <button
            class="btn btn-secondary"
            type="button"
            data-bs-toggle="dropdown"
            aria-expanded="false"
          >
            <b-icon-gear-fill />
          </button>
          <form class="dropdown-menu dropdown-menu-end">
            <!-- for some reason chrome won't open a .m3u8 file directly so don't show the playlist link -->
            <div v-if="!isWebKit">
              <a
                class="dropdown-item"
                :href="playlistURL"
                @click="video!.pause()"
              >
                <img src="/vlc.png" />
                Watch in VLC
              </a>
            </div>
            <div>
              <a
                class="dropdown-item"
                :href="ch.live_url"
                @click.prevent="copyVLC"
              >
                <b-icon-clipboard-data />Copy VLC URL
              </a>
              <input
                v-if="state.showCopyVLC"
                ref="copyVLCInput"
                :value="ch.live_url"
                readonly
              />
            </div>
            <div class="dropdown-divider"></div>
            <div class="ps-3">
              <div class="form-check">
                <input
                  type="checkbox"
                  class="form-check-input"
                  id="rtc"
                  v-model="preferences.useRTC"
                />
                <label class="form-check-label" for="rtc">Use WebRTC</label>
              </div>
              <div class="form-check">
                <input
                  type="checkbox"
                  class="form-check-input"
                  id="ll"
                  v-model="preferences.lowLatency"
                />
                <label class="form-check-label" for="ll">Low Latency</label>
              </div>
            </div>
          </form>
        </div>
        <!-- fullscreen -->
        <button
          v-if="hasFullscreen"
          @click="toggleFullscreen"
          data-bs-toggle="tooltip"
          :title="state.isFullscreen ? 'Exit Fullscreen (f)' : 'Fullscreen (f)'"
        >
          <b-icon-fullscreen v-if="!state.isFullscreen" />
          <b-icon-fullscreen-exit v-else />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import VueSlider from "vue-slider-component";
import "vue-slider-component/theme/antd.css";

import {
  computed,
  nextTick,
  onBeforeMount,
  onMounted,
  onUnmounted,
  reactive,
  ref,
} from "vue";

import {
  type PlayerProperties,
  type PlayerProvider,
  choosePlayer,
} from "../players/provider";
import { useControlsHider } from "@/stores/controls-hider";
import { usePreferences } from "@/stores/preferences";
import {
  BIconPlayFill,
  BIconPauseFill,
  BIconVolumeUpFill,
  BIconVolumeMuteFill,
  BIconSkipForwardFill,
  BIconClockHistory,
  BIconLightningFill,
  BIconLightning,
  BIconSoundwave,
  BIconEyeFill,
  BIconGearFill,
  BIconClipboardData,
  BIconFullscreen,
  BIconFullscreenExit,
} from "bootstrap-icons-vue";

const props = defineProps<PlayerProperties>();
const container = ref<HTMLElement | null>(null);
const video = ref<HTMLVideoElement | null>(null);
const copyVLCInput = ref<HTMLInputElement | null>(null);
const controlsHider = useControlsHider();
const preferences = usePreferences();

const hasFullscreen = document.fullscreenEnabled;
const isWebKit = navigator.userAgent.indexOf("WebKit") >= 0;

const state = reactive({
  playing: false,
  muted: true,
  volume: 0,
  poster: "",

  latency: 0,
  keyPressed: false,
  atTail: false,
  catchingUp: false,
  showCopyVLC: false,

  isFullscreen: document.fullscreenElement !== null,
});

let latencyTimer: number | null;
let keyTimer: number | null;
let player: PlayerProvider | null;

// lifecycle
onBeforeMount(() => {
  // the poster attr on the video element must not be updated reactively,
  // otherwise the element will be continually recreated. use a static copy
  // instead.
  state.poster = props.ch.thumb;
});
onMounted(() => {
  const v = video.value as HTMLVideoElement;
  controlsHider.attachPlayer(v);
  player = choosePlayer(v, props);
  document.addEventListener("fullscreenchange", onFullscreenChanged);
  document.addEventListener("keydown", onKey);
  latencyTimer = window.setInterval(updateLatency, 1000);
});
onUnmounted(() => {
  document.removeEventListener("keydown", onKey);
  document.removeEventListener("fullscreenchange", onFullscreenChanged);
  if (keyTimer) {
    window.clearTimeout(keyTimer);
    keyTimer = null;
  }
  if (latencyTimer) {
    window.clearInterval(latencyTimer);
    latencyTimer = null;
  }
  controlsHider.detachPlayer(video.value);
  player?.destroy();
});

// computed properties
const classes = computed(() => {
  return Object.assign({ controls: true }, controlsHider.hiddenControlClasses);
});
const volumeClasses = computed(() => {
  return { volume: true, "key-pressed": state.keyPressed };
});
const playlistURL = computed(() => {
  return "/live/" + encodeURIComponent(props.ch.name) + ".m3u8";
});

// methods
function toggleFullscreen() {
  if (state.isFullscreen) {
    document.exitFullscreen();
  } else if (container.value) {
    container.value.requestFullscreen();
  }
}
function seekLive() {
  player?.seekLive();
}
function updateLatency() {
  if (!state.playing || !player || !video.value) {
    state.latency = 0;
    return;
  }
  state.catchingUp = video.value.playbackRate != 1;
  state.latency = player.latencyTo();
  state.atTail = state.latency < 5;
}
// copy VLC URL to clipboard
async function copyVLC() {
  state.showCopyVLC = true;
  await nextTick();
  copyVLCInput.value?.select();
  document.execCommand("copy");
  await nextTick();
  state.showCopyVLC = false;
  // this.$bvToast.toast(
  //   "Open VLC, press Ctrl-N and paste to play the stream",
  //   {
  //     title: "Stream URL copied",
  //     isStatus: true,
  //     toaster: "b-toaster-bottom-right",
  //     autoHideDelay: 2000
  //   }
  // );
}

// events
function onFullscreenChanged() {
  state.isFullscreen = document.fullscreenElement !== null;
}
function onPlay() {
  state.playing = true;
  if (video.value) {
    state.volume = video.value.volume;
  }
}
function onVolumeChange() {
  if (video.value) {
    state.muted = video.value.muted;
  }
}
function onKey(ev: KeyboardEvent) {
  if (!video.value) {
    return;
  }
  controlsHider.controlsTouched();
  state.keyPressed = true;
  if (keyTimer) {
    window.clearTimeout(keyTimer);
  }
  keyTimer = window.setTimeout(() => {
    state.keyPressed = false;
    keyTimer = null;
  }, 3000);
  switch (ev.key) {
    case "f":
      toggleFullscreen();
      break;
    case "k":
      if (state.playing) {
        video.value.pause();
      } else {
        video.value.play();
      }
      break;
    case "j":
      seekLive();
      break;
    case "m":
      state.muted = !state.muted;
      video.value.muted = state.muted;
      break;
    case "ArrowDown":
      if (state.volume > 0.05) {
        state.volume -= 0.05;
      } else {
        state.volume = 0;
      }
      video.value.volume = state.volume;
      video.value.muted = false;
      break;
    case "ArrowUp":
      if (state.volume < 0.95) {
        state.volume += 0.05;
      } else {
        state.volume = 1;
      }
      video.value.volume = state.volume;
      video.value.muted = false;
      break;
  }
}
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
  margin-left: 1rem;
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

/* controls hider */
.hidden-control {
  position: absolute !important;
  opacity: 0;
  transition: opacity 0.5s;
  pointer-events: none;
}
.show-control,
.hidden-control:hover {
  opacity: 1 !important;
  pointer-events: auto !important;
}
.no-outline {
  outline: none;
}
.is-tabbing *:focus {
  outline: 2px solid #7aacfe !important;
  outline: 5px auto -webkit-focus-ring-color !important;
}
</style>
