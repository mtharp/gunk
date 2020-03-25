<template>
  <video class="w-100 h-100" muted :poster="poster"></video>
</template>

<script>
import axios from "axios";

export default {
  name: "rtc-player",
  props: ["channel"],
  data() {
    return { poster: this.$root.currentChannel().thumb };
  },
  methods: {
    autoplay() {
      let video = this.$el;
      video.play();
      let vol = localStorage.getItem("volume");
      if (vol) {
        video.volume = vol / 100;
      }
      if (localStorage.getItem("unmute") == "true") {
        video.muted = false;
      }
    },
    saveVolume() {
      let video = this.$el;
      localStorage.setItem("unmute", !video.muted);
      localStorage.setItem("volume", Math.round(video.volume * 100));
    }
  },
  mounted() {
    let video = this.$el;
    video.controls = true;
    var pc = new RTCPeerConnection({
      iceServers: [
        {
          urls: [
            "stun:stun1.l.google.com:19302",
            "stun:stun2.l.google.com:19302"
          ]
        }
      ]
    });
    this.pc = pc;
    this.ms = new MediaStream();
    pc.ontrack = ev => {
      this.ms.addTrack(ev.track);
      try {
        video.srcObject = this.ms;
      } catch (error) {
        // backwards compat
        video.src = URL.createObjectURL(this.ms);
      }
      this.autoplay();
      video.addEventListener("volumechange", this.saveVolume);
    };
    pc.onicecandidate = ev => {
      if (ev.candidate === null) {
        axios
          .post("/sdp/" + encodeURIComponent(this.channel), pc.localDescription)
          .then(d =>
            pc.setRemoteDescription(new RTCSessionDescription(d.data))
          );
      }
    };
    var offerArgs = {};
    try {
      pc.addTransceiver("audio");
      pc.addTransceiver("video");
    } catch (error) {
      // backwards compat
      offerArgs = { offerToReceiveVideo: true, offerToReceiveAudio: true };
    }
    pc.createOffer(offerArgs).then(d => this.pc.setLocalDescription(d));
  },
  beforeDestroy() {
    this.pc.close();
  }
};
</script>
