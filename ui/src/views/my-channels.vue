<template>
  <div class="container mt-3">
    <div class="bg-white">
      <h1>My Channels</h1>
      <b-alert :model-value="state.alert != ''" variant="danger">{{
        state.alert
      }}</b-alert>
      <b-form @submit.prevent="doCreate">
        <b-form-group label="Channel Name">
          <b-form-input v-model="state.newName" required class="mt-1" />
        </b-form-group>

        <b-button type="submit" variant="primary" class="mt-3">Create</b-button>
      </b-form>
      <b-list-group class="mt-5">
        <b-list-group-item v-for="def in state.defs" :key="def.name">
          <h4>{{ def.name }}</h4>
          <b-form-group>
            <b-form-checkbox
              v-model="def.announce"
              switch
              @change="doUpdate(def)"
              >Announce to Discord:
              {{ def.announce ? "Enabled" : "Disabled" }}</b-form-checkbox
            >
          </b-form-group>
          <b-button
            class="my-2"
            size="sm"
            variant="danger"
            @click="doShowDelete(def)"
            >Delete</b-button
          >
          <b-button class="m-2" size="sm" @click="doShow(def)"
            >Configure</b-button
          >
        </b-list-group-item>
      </b-list-group>
    </div>
    <!-- deletion modal -->
    <b-modal
      title="Delete Channel"
      id="confirmDelete"
      v-model="state.showDelete"
      ok-variant="danger"
      ok-title="Delete"
      @ok="doFinishDelete"
    >
      Delete channel {{ state.selected.name }}?
    </b-modal>
    <!-- stream key modal -->
    <b-modal
      title="Stream Key"
      id="keymodal"
      v-model="state.showKey"
      size="lg"
      ok-only
      @hide="state.revealKey = false"
    >
      <h4>OBS Stream Settings</h4>
      <b-card no-body>
        <b-tabs card>
          <b-tab title="WHIP (preferred)">
            <b-card-text>
              <b-form v-if="state.selected">
                <p>WHIP requires OBS 30.0.0 or later.</p>
                <p>Service: <strong>WHIP</strong></p>
                <b-form-group label="Server">
                  <b-form-input readonly :value="state.selected.whip_url" />
                </b-form-group>
                <b-form-group label="Bearer Token" class="mt-3">
                  <b-form-input
                    v-show="state.revealKey"
                    readonly
                    :value="state.selected.key"
                  />
                  <b-button
                    v-show="!state.revealKey"
                    @click="state.revealKey = true"
                    >Reveal Key</b-button
                  >
                </b-form-group>
              </b-form>
            </b-card-text>
          </b-tab>

          <b-tab title="RTMP (legacy)">
            <b-card-text>
              <b-form v-if="state.selected">
                <p>Service: <strong>Custom...</strong></p>
                <b-form-group label="Server">
                  <b-form-input readonly :value="state.selected.rtmp_dir" />
                </b-form-group>
                <b-form-group label="Stream Key" class="mt-3">
                  <b-form-input
                    v-show="state.revealKey"
                    readonly
                    :value="state.selected.rtmp_base"
                  />
                  <b-button
                    v-show="!state.revealKey"
                    @click="state.revealKey = true"
                    >Reveal Key</b-button
                  >
                </b-form-group>
              </b-form>
            </b-card-text>
          </b-tab>
        </b-tabs>
      </b-card>
      <h4 class="mt-3">OBS Recomended Output Settings</h4>
      <b-card no-body>
        <b-tabs card>
          <b-tab title="NVENC">
            <b-card-text>
              <ul>
                <li>Output Mode: Advanced</li>
                <li>Keyframe Interval: 5 seconds</li>
                <li>Preset: Low Latency</li>
                <li>Profile: high</li>
                <li><strong>Max B-frames: 0</strong></li>
              </ul>
            </b-card-text>
          </b-tab>
          <b-tab title="x264">
            <b-card-text>
              <ul>
                <li>Output Mode: Advanced</li>
                <li>Keyframe Interval: 5 seconds</li>
                <li>Preset: veryfast</li>
                <li>Profile: high</li>
                <li>Tune: lowlatency</li>
                <li><strong>x264 options: bframes=0</strong></li>
              </ul>
            </b-card-text>
          </b-tab>
        </b-tabs>
      </b-card>
      <p class="mt-3">
        Make sure B-frames are disabled (set to 0) otherwise WebRTC will not be
        available.
      </p>
    </b-modal>
  </div>
</template>

<script setup lang="ts">
import axios from "axios";
import { onMounted, reactive } from "vue";

interface ChannelDef {
  name: string;
  key?: string;
  announce?: boolean;
  rist_url?: string;
  rtmp_dir?: string;
  rtmp_base?: string;
  whip_url?: string;
}

interface ChannelsResponse {
  channels: ChannelDef[];
}

const state = reactive({
  defs: [] as ChannelDef[],
  selected: { name: "" } as ChannelDef,
  showKey: false,
  revealKey: false,
  showDelete: false,
  newName: "",
  alert: "",
});

onMounted(async () => {
  const { data } = await axios.get<ChannelsResponse>("/api/mychannels");
  state.defs = data.channels;
});

async function doCreate() {
  state.alert = "";
  try {
    const { data } = await axios.post<ChannelDef>("/api/mychannels", {
      name: state.newName,
    });
    if (state.defs) {
      state.defs.unshift(data);
    } else {
      state.defs = [data];
    }
    state.newName = "";
  } catch (err) {
    if (axios.isAxiosError(err) && err.response?.status == 409) {
      state.alert = "Channel name is already in use";
    } else {
      state.alert = "HTTP error while creating channel";
    }
  }
}

function doUpdate(def: ChannelDef) {
  axios.put("/api/mychannels/" + encodeURIComponent(def.name), def);
}

function doShowDelete(def: ChannelDef) {
  state.selected = def;
  state.showDelete = true;
}

async function doFinishDelete() {
  await axios.delete(
    "/api/mychannels/" + encodeURIComponent(state.selected.name),
  );
  state.newName = state.selected.name;
  state.defs.splice(state.defs.indexOf(state.selected), 1);
}

function doShow(def: ChannelDef) {
  state.selected = def;
  state.showKey = true;
}
</script>
