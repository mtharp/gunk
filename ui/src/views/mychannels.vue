<template>
  <div class="container mt-3">
    <div class="col">
      <h1>My Channels</h1>
      <b-form @submit.prevent="doCreate">
        <b-form-group label="Channel Name">
          <b-form-input v-model="newName" required />
        </b-form-group>
        <b-alert :show="alert !== null" variant="danger">{{alert}}</b-alert>
        <b-button type="submit" variant="primary">Create</b-button>
      </b-form>
      <b-list-group class="mt-5">
        <b-list-group-item v-for="def in defs" :key="def.name">
          <h4>{{def.name}}</h4>
          <b-form-group>
            <b-form-checkbox v-model="def.announce" switch @change="doUpdate(def)">Announce {{def.announce ? "Enabled" : "Disabled"}}</b-form-checkbox>
          </b-form-group>
          <b-button class="mr-2" size="sm" variant="danger" @click="doDelete(def)">Delete</b-button>
          <b-button class="mr-2" size="sm" @click="doShow(def)">Show Key</b-button>
        </b-list-group-item>
      </b-list-group>
    </div>
    <b-modal
      title="Stream Key"
      id="keymodal"
      v-model="showKey"
      size="lg"
      ok-only
      >
      <b-form v-if="selected">
        <b-form-group label="Server">
          <b-form-input readonly :value="selected.rtmp_dir" />
        </b-form-group>
        <b-form-group label="Stream Key">
          <b-form-input readonly :value="selected.rtmp_base" />
        </b-form-group>
      </b-form>
    </b-modal>
  </div>
</template>

<script>
import axios from 'axios';

export default {
  name: 'mychannels',
  data() {
    return {
      defs: [],
      newName: null,
      selected: null,
      showKey: false,
      alert: null,
    }
  },
  mounted() {
    axios.get("/api/mychannels")
      .then(response => this.defs = response.data)
  },
  methods: {
    doCreate() {
      this.alert = null;
      axios.post("/api/mychannels", {name: this.newName})
        .then(response => {
          let def = response.data;
          if (this.defs === null) {
            this.defs = [def]
          } else {
            this.defs.unshift(def)
          }
          this.newName = ""
        }).catch(error => {
          if (error.response.status == 409) {
            this.alert = "Channel name is already in use"
          } else {
            this.alert = "HTTP error while creating channel"
          }
        })
    },
    doUpdate(def) {
      axios.put("/api/mychannels/" + encodeURIComponent(def.name), def)
    },
    doDelete(def) {
      axios.delete("/api/mychannels/" + encodeURIComponent(def.name))
        .then(() => this.defs.splice(this.defs.indexOf(def), 1))
    },
    doShow(def) {
      this.selected = def
      this.showKey = true
    },
  },
}
</script>
