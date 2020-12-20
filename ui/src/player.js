import Hls from 'hls.js/dist/hls';
// import { MediaPlayer } from 'dashjs';
import Axios from 'axios';

// play video when ready and restore and save volume
function autoplay (video) {
  video.onvolumechange = null;
  video.muted = true;
  let forcedMute = false;
  video.onplay = function () {
    // update saved volume once playing, after we're done testing whether we can unmute
    video.onvolumechange = () => {
      localStorage.setItem('unmute', !video.muted);
      localStorage.setItem('volume', Math.round(video.volume * 100));
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
    // restore saved volume
    const vol = localStorage.getItem('volume');
    if (vol) {
      video.volume = vol / 100;
    }
    if (localStorage.getItem('unmute') !== 'false') {
      video.muted = false;
    }
    video.play()
      .catch(() => {
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

export class HLSPlayer {
  constructor (video, webURL, lowLatencyMode) {
    autoplay(video);
    this.video = video;
    this.stream = null;
    if (!Hls.isSupported()) {
      if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = webURL;
      }
      return;
    }
    const conf = {
      // debug: true,
      bitrateTest: false,
      liveDurationInfinity: true,
      liveBackBufferLength: 10,
      lowLatencyMode: lowLatencyMode,
      maxLiveSyncPlaybackRate: 1
    };
    if (lowLatencyMode) {
      conf.liveBackBufferLength = 0;
      conf.liveSyncDuration = 0.5;
      conf.maxLiveSyncPlaybackRate = 1.1;
    } else {
      conf.liveSyncDurationCount = 2;
    }
    this.stream = new Hls(conf);
    this.stream.attachMedia(video);
    this.stream.loadSource(webURL);
  }

  destroy () {
    this.video = null;
    if (this.stream !== null) {
      this.stream.destroy();
      this.stream = null;
    }
  }

  seekLive () {
    this.video.currentTime = this.stream.liveSyncPosition;
    this.video.play();
  }

  latencyTo () {
    if (this.stream !== null) {
      const latency = this.stream.latency;
      return [latency, latency < 15];
    }
    return null;
  }
}

// export class DASHPlayer {
//   constructor (video, webURL, lowLatencyMode) {
//     autoplay(video);
//     this.video = video;
//     this.stream = MediaPlayer().create();
//     this.stream.initialize();
//     this.stream.updateSettings({
//       streaming: { lowLatencyEnabled: lowLatencyMode }
//     });
//     this.stream.setAutoPlay(false);
//     this.stream.attachSource(webURL);
//     this.stream.attachView(video);
//   }

//   destroy () {
//     this.video = null;
//     if (this.stream !== null) {
//       this.stream.reset();
//     }
//   }

//   seekLive () {
//     this.video.play();
//   }

//   latencyTo () {
//     if (this.stream !== null) {
//       return [this.stream.getCurrentLiveLatency(), true];
//     }
//     return null;
//   }
// }

export class RTCPlayer {
  constructor (video, sdpURL) {
    autoplay(video);
    this.pc = new RTCPeerConnection({
      iceServers: [{
        urls: [
          'stun:stun1.l.google.com:19302',
          'stun:stun2.l.google.com:19302'
        ]
      }]
    });
    // as the RTC session sets up tracks, attach them to a media stream that will feed the player
    const ms = new MediaStream();
    this.pc.ontrack = function (ev) {
      ms.addTrack(ev.track);
      if ('srcObject' in video) {
        video.srcObject = ms;
      } else {
        video.src = URL.createObjectURL(ms);
      }
    };
    this.pc.onicecandidate = (ev) => {
      if (ev.candidate !== null) {
        // still gathering candidates
        return;
      }
      // full set of candidates is done, send the offer
      Axios.post(sdpURL, this.pc.localDescription)
        .then(d => this.pc.setRemoteDescription(new RTCSessionDescription(d.data)));
    };
    // offer to receive
    let offerArgs = {};
    try {
      this.pc.addTransceiver('audio', { direction: 'recvonly' });
      this.pc.addTransceiver('video', { direction: 'recvonly' });
    } catch (error) {
      // backwards compat
      offerArgs = { offerToReceiveVideo: true, offerToReceiveAudio: true };
    }
    this.pc.createOffer(offerArgs).then(d => this.pc.setLocalDescription(d));
    // icecandidate gets called a bunch and then eventually with null, at which point the offer will be sent
  }

  destroy () {
    this.pc.close();
  }

  seekLive () {
    this.player.play();
  }

  latencyTo () {
    return [0, true];
  }
}
