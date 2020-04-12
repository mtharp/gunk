import Hls from 'hls.js/dist/hls.js';
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
  constructor (video, hlsURL) {
    autoplay(video);
    this.video = video;
    this.hls = null;
    // how far behind the latest feasible point to sit
    this.targetBuffer = 1;
    if (Hls.isSupported()) {
      this.hls = new Hls({
        bitrateTest: false,
        liveDurationInfinity: true,
        liveBackBufferLength: 30,
        liveSyncDurationCount: 2,
      });
      this.hls.attachMedia(video);
      this.hls.loadSource(hlsURL);
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      video.src = hlsURL;
    }
  }

  destroy () {
    this.video = null;
    if (this.hls !== null) {
      this.hls.destroy();
      this.hls = null;
    }
  }

  details () {
    if (!this.hls || !this.hls.levels) {
      return null;
    }
    const lev = this.hls.levels[this.hls.currentLevel];
    if (!lev) {
      return null;
    }
    return lev.details;
  }

  seekLive () {
    // seek to end and play
    const details = this.details();
    if (details === null || details.fragments.length < 3) {
      this.video.play();
      return;
    }
    for (let i = details.fragments.length - 1; i >= 0; i--) {
      const f = details.fragments[i];
      if (f.appendedPTS) {
        // streaming this segment and chunks are ready to play
        this.video.currentTime = f.appendedPTS - this.targetBuffer;
        // console.log('s1', details.fragments.length - i, f);
        return;
      } else if (f.endPTS) {
        // segment is fully processed, start from here
        if ('appendedPTS' in f) {
          // the next segment is streaming but hasn't appended yet, still we can start near the end of this one and hopefully it will be ready
          // console.log('s2', details.fragments.length - i, f);
          this.video.currentTime = f.endPTS - this.targetBuffer;
        } else {
          // must wait an additional segment length because the next one isn't streaming
          // console.log('s3', details.fragments.length - i, f);
          this.video.currentTime = f.start - this.targetBuffer;
        }
        return;
      }
    }
  }

  latencyTo (serverTime) {
    // return how far behind the player is from the given server time, and whether it's close enough to live
    const details = this.details();
    if (serverTime === null || details === null || details.fragments.length < 2) {
      return null;
    }
    let targetDuration = details.targetduration;
    if ('appendedPTS' in details.fragments[0]) {
      // LHLS, can play segments that are still downloading
      targetDuration = 0;
    }
    const maxLatency = 5 * this.targetBuffer + 2 * targetDuration;
    for (let i = details.fragments.length - 2; i >= 0; i--) {
      const f = details.fragments[i];
      if (f.programDateTime && !f.prefetch) {
        const deltaDTS = this.video.currentTime - f.start;
        const deltaDate = serverTime - f.programDateTime;
        const latency = deltaDate / 1000 - deltaDTS;
        return [latency, latency < maxLatency];
      }
    }
    return null;
  }
}

export function attachRTCPlay (video, ws, channel) {
  video.controls = true;
  video.autoplay = true;
  video.addEventListener('canplay', function () { video.play(); });
  const ms = new MediaStream();
  if ('srcObject' in video) {
    video.srcObject = ms;
  } else {
    // backwards compat
    video.src = URL.createObjectURL(ms);
  }
  const pc = new RTCPeerConnection({
    iceServers: [{
      urls: [
        'stun:stun1.l.google.com:19302',
        'stun:stun2.l.google.com:19302'
      ]
    }]
  });
  // as the RTC session sets up tracks, attach them to a media stream that will feed the player
  pc.addEventListener('track', (ev) => ms.addTrack(ev.track));
  pc.addEventListener('icecandidate', (ev) => ws.candidate(ev.candidate));
  ws.onCandidate = (cand) => pc.addIceCandidate(cand);
  // ask for an offer
  // request an offer from the server
  ws.play(channel)
    .then((offer) => pc.setRemoteDescription(new RTCSessionDescription(offer)))
    .then(() => pc.createAnswer())
    .then((answer) => {
      ws.answer(answer);
      pc.setLocalDescription(answer);
    });
  return function () {
    ws.stop();
    pc.close();
  };
}

export function attachRTCOffer (video, ws, channel) {
  video.controls = true;
  video.autoplay = true;
  video.addEventListener('canplay', function () { video.play(); });
  const ms = new MediaStream();
  if ('srcObject' in video) {
    video.srcObject = ms;
  } else {
    // backwards compat
    video.src = URL.createObjectURL(ms);
  }
  const pc = new RTCPeerConnection({
    iceServers: [{
      urls: [
        'stun:stun1.l.google.com:19302',
        'stun:stun2.l.google.com:19302'
      ]
    }]
  });
  // as the RTC session sets up tracks, attach them to a media stream that will feed the player
  pc.addEventListener('track', (ev) => ms.addTrack(ev.track));
  pc.addEventListener('icecandidate', (ev) => ws.candidate(ev.candidate));
  let savedCandidates = [];
  ws.onCandidate = (cand) => {
    if (savedCandidates === null) {
      console.log('applying', cand);
      pc.addIceCandidate(cand);
    } else {
      console.log('saving', cand);
      savedCandidates.push(cand);
    }
  };
  // offer to receive
  var offerArgs = {};
  try {
    pc.addTransceiver('audio', { direction: 'recvonly' });
    pc.addTransceiver('video', { direction: 'recvonly' });
  } catch (error) {
    // backwards compat
    offerArgs = { offerToReceiveVideo: true, offerToReceiveAudio: true };
  }
  pc.createOffer(offerArgs)
    .then(offer => {
      pc.setLocalDescription(offer);
      return ws.offer(channel, offer);
    }).then(answer => {
      pc.setRemoteDescription(new RTCSessionDescription(answer));
      for (const cand of savedCandidates) {
        console.log('applying deferred', cand);
        pc.addIceCandidate(cand);
      }
      savedCandidates = null;
    });
  return function () {
    ws.stop();
    pc.close();
  };
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
    // rtc is always live
  }

  latencyTo () {
    return [0, true];
  }
}
export function attachRTC (video, sdpURL) {
  video.controls = true;
  video.autoplay = true;
  video.addEventListener('canplay', function () { video.play(); });
  const pc = new RTCPeerConnection({
    iceServers: [{
      urls: [
        'stun:stun1.l.google.com:19302',
        'stun:stun2.l.google.com:19302'
      ]
    }]
  });
  // as the RTC session sets up tracks, attach them to a media stream that will feed the player
  const ms = new MediaStream();
  pc.addEventListener('track', function (ev) {
    ms.addTrack(ev.track);
    if ('srcObject' in video) {
      video.srcObject = ms;
    } else {
      video.src = URL.createObjectURL(ms);
    }
  });
  pc.addEventListener('icecandidate', function (ev) {
    if (ev.candidate !== null) {
      // still gathering candidates
      return;
    }
    // full set of candidates is done, send the offer
    Axios.post(sdpURL, pc.localDescription)
      .then(d => pc.setRemoteDescription(new RTCSessionDescription(d.data)));
  });
  // offer to receive
  let offerArgs = {};
  try {
    pc.addTransceiver('audio', { direction: 'recvonly' });
    pc.addTransceiver('video', { direction: 'recvonly' });
  } catch (error) {
    // backwards compat
    offerArgs = { offerToReceiveVideo: true, offerToReceiveAudio: true };
  }
  pc.createOffer(offerArgs).then(d => pc.setLocalDescription(d));
  // icecandidate gets called a bunch and then eventually with null, at which point the offer will be sent
  return function () { pc.close(); };
}
