import axios from "axios";
import Hls from "hls.js/dist/hls.js";

export function restoreVolume(video) {
    let vol = localStorage.getItem("volume");
    if (vol) {
        video.volume = vol / 100;
    }
    if (localStorage.getItem("unmute") == "true") {
        video.muted = false;
    }
    video.addEventListener("volumechange", function () {
        localStorage.setItem("unmute", !this.muted);
        localStorage.setItem("volume", Math.round(this.volume * 100));
    });
}

export function attachHLS(video, hlsURL) {
    video.controls = true;
    video.autoplay = true;
    if (Hls.isSupported()) {
        let hls = new Hls({
            bitrateTest: false
            // debug: true
        });
        hls.attachMedia(video);
        hls.loadSource(hlsURL);
        return function () { hls.destroy() };
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
        video.src = hlsURL;
    }
    return function () { };
}

export function attachRTC(video, sdpURL) {
    video.controls = true;
    video.autoplay = true;
    let pc = new RTCPeerConnection({
        iceServers: [{
            urls: [
                "stun:stun1.l.google.com:19302",
                "stun:stun2.l.google.com:19302"
            ]
        }]
    });
    // as the RTC session sets up tracks, attach them to a media stream that will feed the player
    let ms = new MediaStream();
    pc.addEventListener("track", function (ev) {
        ms.addTrack(ev.track);
        try {
            video.srcObject = ms;
        } catch (error) {
            // backwards compat
            video.src = URL.createObjectURL(ms);
        }
    });
    pc.addEventListener("icecandidate", function (ev) {
        if (ev.candidate !== null) {
            // still gathering candidates
        }
        axios.post(sdpURL, pc.localDescription).then(function (d) {
            pc.setRemoteDescription(new RTCSessionDescription(d.data))
        });
    });
    var offerArgs = {};
    try {
        pc.addTransceiver("audio");
        pc.addTransceiver("video");
    } catch (error) {
        // backwards compat
        offerArgs = { offerToReceiveVideo: true, offerToReceiveAudio: true };
    }
    pc.createOffer(offerArgs).then(d => pc.setLocalDescription(d));
    return function () { pc.close() };
}
