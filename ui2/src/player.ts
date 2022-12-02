// import Hls from 'hls.js/dist/hls';
import dashjs, { MediaPlayer, MediaPlayerSettingClass } from "dashjs";
import ws from "./ws";

// play video when ready and restore and save volume
function autoplay(video: HTMLVideoElement) {
  video.onvolumechange = null;
  video.muted = true;
  let forcedMute = false;
  video.onplay = function () {
    // update saved volume once playing, after we're done testing whether we can unmute
    video.onvolumechange = () => {
      localStorage.setItem("unmute", String(!video.muted));
      localStorage.setItem("volume", String(Math.round(video.volume * 100)));
    };
    if (!forcedMute) {
      video.onclick = null;
    }
    forcedMute = false;
  };
  video.onclick = function () {
    // unmute and play on first click, if autoplay failed
    video.muted = false;
    video.play();
  };
  video.onpause = function () {
    // just play on subsequent clicks when paused
    video.onclick = function () {
      video.play();
    };
  };
  video.oncanplay = function () {
    video.oncanplay = null;
    // restore saved volume
    const vol = Number(localStorage.getItem("volume"));
    if (vol) {
      video.volume = vol / 100;
    }
    if (localStorage.getItem("unmute") !== "false") {
      video.muted = false;
    }
    video.play().catch(() => {
      // autoplay not allowed with sound, mute and try again
      video.muted = true;
      video.onclick = function () {
        // unmute on click
        video.muted = false;
      };
      forcedMute = true;
      return video.play();
    });
  };
}

export interface PlayerProvider {
  destroy(): void;
  seekLive(): void;
  latencyTo(): number;
}

// export class HLSPlayer {
//   constructor (video, webURL, lowLatencyMode) {
//     autoplay(video);
//     this.video = video;
//     this.stream = null;
//     if (!Hls.isSupported()) {
//       if (video.canPlayType('application/vnd.apple.mpegurl')) {
//         video.src = webURL;
//       }
//       return;
//     }
//     const conf = {
//       // debug: true,
//       bitrateTest: false,
//       liveDurationInfinity: true,
//       backBufferLength: 10,
//       lowLatencyMode: lowLatencyMode,
//       maxLiveSyncPlaybackRate: 1
//     };
//     if (lowLatencyMode) {
//       conf.backBufferLength = 0;
//       conf.liveSyncDuration = 2;
//       conf.maxLiveSyncPlaybackRate = 1.1;
//     } else {
//       conf.liveSyncDurationCount = 2;
//     }
//     this.stream = new Hls(conf);
//     this.stream.attachMedia(video);
//     this.stream.loadSource(webURL);
//     this.stream.on(Hls.Events.MANIFEST_LOADED, () => video.oncanplay());
//   }

//   destroy () {
//     this.video = null;
//     if (this.stream !== null) {
//       this.stream.destroy();
//       this.stream = null;
//     }
//   }

//   seekLive () {
//     this.video.currentTime = this.stream.liveSyncPosition;
//     this.video.play();
//   }

//   latencyTo () {
//     if (this.stream !== null) {
//       const latency = this.stream.latency;
//       return [latency, latency < 15];
//     }
//     return null;
//   }
// }

// test for iOS, which only supports RTC and native HLS
export function nativeRequired() {
  return (
    /iPad|iPhone|iPod/.test(navigator.userAgent) && !("MSStream" in window)
  );
}

export class NativePlayer {
  video: HTMLVideoElement;

  constructor(video: HTMLVideoElement, webURL: string) {
    autoplay(video);
    this.video = video;
    video.src = webURL;
  }

  destroy() {}

  seekLive() {
    this.video.play();
  }

  latencyTo() {
    return 0;
  }
}

export class DASHPlayer {
  video: HTMLVideoElement;
  stream: dashjs.MediaPlayerClass;

  constructor(
    video: HTMLVideoElement,
    webURL: string,
    lowLatencyMode: boolean
  ) {
    autoplay(video);
    this.video = video;
    this.stream = MediaPlayer().create();
    this.stream.initialize();
    const settings: MediaPlayerSettingClass = {
      streaming: {
        liveCatchup: {
          playbackRate: 0.1,
        },
        utcSynchronization: {
          timeBetweenSyncAttempts: 30,
        },
      },
    };
    if (lowLatencyMode) {
      settings.streaming!.delay = { liveDelay: 2 };
    } else {
      settings.streaming!.delay = { liveDelayFragmentCount: 2 };
    }
    this.stream.updateSettings(settings);
    this.stream.setAutoPlay(true);
    this.stream.attachSource(webURL);
    this.stream.attachView(video);
  }

  destroy() {
    this.stream.reset();
  }

  seekLive() {
    this.stream.seek(this.stream.duration());
    this.stream.play();
  }

  latencyTo() {
    return this.stream.getCurrentLiveLatency();
  }
}

export class RTCPlayer {
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
