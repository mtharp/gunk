<template>
  <div class="container mt-3">
    <div class="bg-white">
      <h1>My Channels</h1>
      <b-form @submit.prevent="doCreate">
        <b-form-group label="Channel Name">
          <b-form-input v-model="state.newName" required />
        </b-form-group>
        <b-alert :show="state.alert != ''" variant="danger">{{
          state.alert
        }}</b-alert>
        <b-button type="submit" variant="primary">Create</b-button>
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
            class="mr-2"
            size="sm"
            variant="danger"
            @click="doDelete(def)"
            >Delete</b-button
          >
          <b-button class="mr-2" size="sm" @click="doShow(def)"
            >Show Key</b-button
          >
        </b-list-group-item>
      </b-list-group>
    </div>
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
          <b-tab title="Standard">
            <b-card-text>
              <b-form v-if="state.selected">
                <b-form-group label="Server (custom service)">
                  <b-form-input readonly :value="state.selected.rtmp_dir" />
                </b-form-group>
                <b-form-group label="Stream Key">
                  <b-form-input
                    v-if="state.revealKey"
                    readonly
                    :value="state.selected.rtmp_base"
                  />
                  <b-button v-else @click="state.revealKey = true"
                    >Reveal Key</b-button
                  >
                </b-form-group>
              </b-form>
            </b-card-text>
          </b-tab>
          <b-tab title="FTL (alpha)">
            <b-card-text>
              <b-form v-if="state.selected">
                <b-form-group label="Stream Key">
                  <b-form-input
                    v-if="state.revealKey"
                    readonly
                    :value="state.selected.ftl_key"
                  />
                  <b-button v-else @click="state.revealKey = true"
                    >Reveal Key</b-button
                  >
                </b-form-group>
                <b-form-group
                  label="OBS Services Config Snippet"
                  description='Open C:\Users\YOURACCOUNT\AppData\Roaming\obs-studio\plugin_config\rtmp-services\services.json in Notepad and paste this block after the line: "services": [ and then restart OBS and pick the new entry from the services drop-down. This will probably break every time you update OBS.'
                >
                  <b-form-textarea
                    readonly
                    rows="18"
                    :value="state.ftlConfig"
                  />
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
                <li>Preset: Quality</li>
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
                <li>Tune: (none)</li>
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
  ftl_key?: string;
  rtmp_dir?: string;
  rtmp_base?: string;
}

interface ChannelsResponse {
  channels: ChannelDef[];
  ftl: string;
}

const state = reactive({
  defs: [] as ChannelDef[],
  selected: { name: "" } as ChannelDef,
  showKey: false,
  revealKey: false,
  newName: "",
  ftlConfig: "",
  alert: "",
});

onMounted(async () => {
  const { data } = await axios.get<ChannelsResponse>("/api/mychannels");
  state.defs = data.channels;
  state.ftlConfig = data.ftl;
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

async function doDelete(def: ChannelDef) {
  // const confirmed = await state.$bvModal.msgBoxConfirm(
  //   "Delete channel " + def.name + "?", {
  //     title: "Delete Channel",
  //     okVariant: "danger",
  //     okTitle: "Delete"
  //   });
  const confirmed = false;
  if (confirmed) {
    await axios.delete("/api/mychannels/" + encodeURIComponent(def.name));
    state.defs.splice(state.defs.indexOf(def), 1);
  }
}

function doShow(def: ChannelDef) {
  state.selected = def;
  state.showKey = true;
}
</script>
