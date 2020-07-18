import { MediaPlayer, Debug } from 'dashjs';
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
  constructor (video, webBase) {
    autoplay(video);
    this.video = video;
    this.stream = null;
    this.stream = MediaPlayer().create();
    this.stream.initialize();
    this.stream.updateSettings({
      streaming: {
        // lowLatencyEnabled: true,
        liveDelay: 1,
        liveCatchUpMinDrift: 0.5,
        liveCatchUpPlaybackRate: 0.05
      },
      debug: {
        logLevel: Debug.LOG_LEVEL_DEBUG
      }
    });
    // don't autoplay, the handler here will take care of it and this way the poster will be displayed in the meantime
    this.stream.setAutoPlay(false);
    this.stream.attachSource(webBase + '.mpd');
    this.stream.attachView(video);
    // if (video.canPlayType('application/vnd.apple.mpegurl')) {
    //   video.src = hlsURL;
    // }
  }

  destroy () {
    this.video = null;
    if (this.stream !== null) {
      this.stream.reset();
      this.stream = null;
    }
  }

  seekLive () {
    this.video.play();
    // seek to end and play
    // const details = this.details();
    // if (details === null || details.fragments.length < 3) {
    //   this.video.play();
    //   return;
    // }
    // for (let i = details.fragments.length - 1; i >= 0; i--) {
    //   const f = details.fragments[i];
    //   if (f.appendedPTS) {
    //     // streaming this segment and chunks are ready to play
    //     this.video.currentTime = f.appendedPTS - this.targetBuffer;
    //     // console.log('s1', details.fragments.length - i, f);
    //     return;
    //   } else if (f.endPTS) {
    //     // segment is fully processed, start from here
    //     if ('appendedPTS' in f) {
    //       // the next segment is streaming but hasn't appended yet, still we can start near the end of this one and hopefully it will be ready
    //       // console.log('s2', details.fragments.length - i, f);
    //       this.video.currentTime = f.endPTS - this.targetBuffer;
    //     } else {
    //       // must wait an additional segment length because the next one isn't streaming
    //       // console.log('s3', details.fragments.length - i, f);
    //       this.video.currentTime = f.start - this.targetBuffer;
    //     }
    //     return;
    // }
  }

  latencyTo () {
    try {
      return [this.stream.getCurrentLiveLatency(), true];
    } catch (error) {
      return null;
    }
  }
}

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
