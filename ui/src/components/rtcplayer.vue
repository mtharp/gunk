<template>
  <video
    autoplay
    muted
    controls
    class="w-100 h-100"
    />
</template>

<script>
import axios from 'axios'

export default {
  name: 'rtc-player',
  props: [
    'channel',
  ],
  mounted() {
    var pc = new RTCPeerConnection({
      iceServers: [{
        urls: [
          'stun:stun1.l.google.com:19302',
          'stun:stun2.l.google.com:19302',
          ],
      }],
    })
    this.pc = pc
    this.ms = new MediaStream()
    pc.ontrack = (ev) => {
      this.ms.addTrack(ev.track)
      try {
        this.$el.srcObject = this.ms
      } catch (error) {
        this.$el.src = URL.createObjectURL(this.ms)
      }
    }
    pc.onicecandidate = (ev) => {
      if (ev.candidate === null) {
        axios.post("/sdp/" + encodeURIComponent(this.channel), pc.localDescription)
          .then(d => pc.setRemoteDescription(new RTCSessionDescription(d.data)));
      }
    }
    pc.createOffer({offerToReceiveVideo: true, offerToReceiveAudio: true}).then(d => this.pc.setLocalDescription(d));
  },
  beforeDestroy() {
    this.pc.close();
  },
}
</script>
