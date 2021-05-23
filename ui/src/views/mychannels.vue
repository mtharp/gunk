<template>
  <div class="container mt-3">
    <div class="bg-white">
      <h1>My Channels</h1>
      <b-form @submit.prevent="doCreate">
        <b-form-group label="Channel Name">
          <b-form-input v-model="newName" required />
        </b-form-group>
        <b-alert :show="alert !== null" variant="danger">{{ alert }}</b-alert>
        <b-button type="submit" variant="primary">Create</b-button>
      </b-form>
      <b-list-group class="mt-5">
        <b-list-group-item v-for="def in defs" :key="def.name">
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
      v-model="showKey"
      size="lg"
      ok-only
      @hide="revealKey = false"
    >
      <h4>OBS Stream Settings</h4>
      <b-card no-body>
        <b-tabs card>
          <b-tab title="Standard">
            <b-card-text>
              <b-form v-if="selected">
                <b-form-group label="Server (custom service)">
                  <b-form-input readonly :value="selected.rtmp_dir" />
                </b-form-group>
                <b-form-group label="Stream Key">
                  <b-form-input
                    v-if="revealKey"
                    readonly
                    :value="selected.rtmp_base"
                  />
                  <b-button v-else @click="revealKey = true"
                    >Reveal Key</b-button
                  >
                </b-form-group>
              </b-form>
            </b-card-text>
          </b-tab>
          <b-tab title="FTL (alpha)">
            <b-card-text>
              <b-form v-if="selected">
                <b-form-group label="Stream Key">
                  <b-form-input
                    v-if="revealKey"
                    readonly
                    :value="selected.ftl_key"
                  />
                  <b-button v-else @click="revealKey = true"
                    >Reveal Key</b-button
                  >
                </b-form-group>
                <b-form-group
                  label="OBS Services Config Snippet"
                  description='Open C:\Users\YOURACCOUNT\AppData\Roaming\obs-studio\plugin_config\rtmp-services\services.json in Notepad and paste this block after the line: "services": [ and then restart OBS and pick the new entry from the services drop-down. This will probably break every time you update OBS.'
                >
                  <b-form-textarea readonly rows="18" :value="ftl_config" />
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

<script>
import axios from "axios";

export default {
  name: "mychannels",
  data() {
    return {
      defs: [],
      newName: null,
      selected: null,
      showKey: false,
      revealKey: false,
      ftl_config: "",
      alert: null
    };
  },
  mounted() {
    axios.get("/api/mychannels").then(response => {
      this.defs = response.data.channels;
      this.ftl_config = response.data.ftl;
    });
  },
  methods: {
    doCreate() {
      this.alert = null;
      axios
        .post("/api/mychannels", { name: this.newName })
        .then(response => {
          let def = response.data;
          if (this.defs === null) {
            this.defs = [def];
          } else {
            this.defs.unshift(def);
          }
          this.newName = "";
        })
        .catch(error => {
          if (error.response.status == 409) {
            this.alert = "Channel name is already in use";
          } else {
            this.alert = "HTTP error while creating channel";
          }
        });
    },
    doUpdate(def) {
      axios.put("/api/mychannels/" + encodeURIComponent(def.name), def);
    },
    doDelete(def) {
      this.$bvModal
        .msgBoxConfirm("Delete channel " + def.name + "?", {
          title: "Delete Channel",
          okVariant: "danger",
          okTitle: "Delete"
        })
        .then(confirmed => {
          if (confirmed) {
            axios
              .delete("/api/mychannels/" + encodeURIComponent(def.name))
              .then(() => this.defs.splice(this.defs.indexOf(def), 1));
          }
        });
    },
    doShow(def) {
      this.selected = def;
      this.showKey = true;
    }
  }
};
</script>
