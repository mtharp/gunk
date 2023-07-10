import autoplay from "./autoplay";
import ws from "../ws";

export default class RTCPlayer {
  video: HTMLVideoElement;
  pc: RTCPeerConnection;

  constructor(video: HTMLVideoElement, channel: string) {
    autoplay(video);
    this.video = video;
    this.pc = new RTCPeerConnection({
      iceServers: [
        {
          urls: [
            "stun:stun1.l.google.com:19302",
            "stun:stun2.l.google.com:19302",
          ],
        },
      ],
    });
    // accept audio and video
    let offerArgs = {};
    try {
      this.pc.addTransceiver("audio", { direction: "recvonly" });
      this.pc.addTransceiver("video", { direction: "recvonly" });
    } catch {
      // backwards compat
      offerArgs = { offerToReceiveVideo: true, offerToReceiveAudio: true };
    }
    // as the RTC session sets up tracks, attach them to a media stream that will feed the player
    const ms = new MediaStream();
    this.pc.ontrack = function (ev) {
      ms.addTrack(ev.track);

      if ("srcObject" in video) {
        video.srcObject = ms;
      } else {
        // @ts-ignore-next-line
        video.src = URL.createObjectURL(ms);
      }
    };
    this.pc.addEventListener("icecandidate", (ev) => {
      if (ev.candidate) {
        ws.candidate(ev.candidate);
      }
    });
    ws.onCandidate = (cand: RTCIceCandidateInit) =>
      this.pc.addIceCandidate(cand);
    // request an offer
    ws.play(channel)
      .then((offer: RTCSessionDescriptionInit) => {
        this.pc.setRemoteDescription(new RTCSessionDescription(offer));
      })
      .then(() => this.pc.createAnswer(offerArgs))
      .then((answer: RTCSessionDescriptionInit) => {
        ws.answer(answer);
        this.pc.setLocalDescription(answer);
      });
  }

  destroy() {
    ws.stop();
    this.pc.close();
  }

  seekLive() {
    this.video.play();
  }

  latencyTo() {
    return 0;
  }
}
