import Hls from "hls.js/dist/hls.js";
import Axios from "axios";

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
        hls.on(Hls.Events.MEDIA_ATTACHED, function () { video.play() });
        return function () { hls.destroy() };
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
        video.src = hlsURL;
        video.addEventListener("canplay", function () { video.play() });
    }
    return function () { };
}

export function attachRTCPlay(video, ws, channel) {
    video.controls = true;
    video.autoplay = true;
    video.addEventListener("canplay", function () { video.play() });
    let ms = new MediaStream();
    if ("srcObject" in video) {
        video.srcObject = ms;
    } else {
        // backwards compat
        video.src = URL.createObjectURL(ms);
    }
    let pc = new RTCPeerConnection({
        iceServers: [{
            urls: [
                "stun:stun1.l.google.com:19302",
                "stun:stun2.l.google.com:19302"
            ]
        }]
    });
    // as the RTC session sets up tracks, attach them to a media stream that will feed the player
    pc.addEventListener("track", (ev) => ms.addTrack(ev.track));
    pc.addEventListener("icecandidate", (ev) => ws.candidate(ev.candidate));
    ws.onCandidate = (cand) => pc.addIceCandidate(cand);
    // ask for an offer
    // request an offer from the server
    ws.play(channel)
        .then((offer) => pc.setRemoteDescription(new RTCSessionDescription(offer)))
        .then(() => pc.createAnswer())
        .then((answer) => {
            ws.answer(answer);
            pc.setLocalDescription(answer)
        });
    return function () {
        ws.stop();
        pc.close();
    };
}

export function attachRTCOffer(video, ws, channel) {
    video.controls = true;
    video.autoplay = true;
    video.addEventListener("canplay", function () { video.play() });
    let ms = new MediaStream();
    if ("srcObject" in video) {
        video.srcObject = ms;
    } else {
        // backwards compat
        video.src = URL.createObjectURL(ms);
    }
    let pc = new RTCPeerConnection({
        iceServers: [{
            urls: [
                "stun:stun1.l.google.com:19302",
                "stun:stun2.l.google.com:19302"
            ]
        }]
    });
    // as the RTC session sets up tracks, attach them to a media stream that will feed the player
    pc.addEventListener("track", (ev) => ms.addTrack(ev.track));
    pc.addEventListener("icecandidate", (ev) => ws.candidate(ev.candidate));
    let savedCandidates = new Array();
    ws.onCandidate = (cand) => {
        if (savedCandidates === null) {
            console.log("applying", cand);
            pc.addIceCandidate(cand);
        } else {
            console.log("saving", cand);
            savedCandidates.push(cand);
        }
    };
    // offer to receive
    var offerArgs = {};
    try {
        pc.addTransceiver("audio", { direction: "recvonly" });
        pc.addTransceiver("video", { direction: "recvonly" });
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
            for (let cand of savedCandidates) {
                console.log("applying deferred", cand);
                pc.addIceCandidate(cand);
            }
            savedCandidates = null;
        });
    return function () {
        ws.stop();
        pc.close();
    };
}

export function attachRTC(video, sdpURL) {
    video.controls = true;
    video.autoplay = true;
    video.addEventListener("canplay", function () { video.play() });
    // attach tracks to a shared MediaStream.
    // note that safari does not support srcObject, nor does the fallback work if it's placed here, so handle that case when tracks are added.
    // setting srcObject here results in a smoother experience for other browsers.
    let ms = new MediaStream();
    video.srcObject = ms;
    let pc = new RTCPeerConnection({
        iceServers: [{
            urls: [
                "stun:stun1.l.google.com:19302",
                "stun:stun2.l.google.com:19302"
            ]
        }]
    });
    // as the RTC session sets up tracks, attach them to a media stream that will feed the player
    pc.addEventListener("track", function (ev) {
        ms.addTrack(ev.track);
        if ("srcObject" in video === false) {
            video.src = URL.createObjectURL(ms);
        }
    });
    pc.addEventListener("icecandidate", function (ev) {
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
        pc.addTransceiver("audio", { direction: "recvonly" });
        pc.addTransceiver("video", { direction: "recvonly" });
    } catch (error) {
        // backwards compat
        offerArgs = { offerToReceiveVideo: true, offerToReceiveAudio: true };
    }
    pc.createOffer(offerArgs).then(d => pc.setLocalDescription(d));
    // icecandidate gets called a bunch and then eventually with null, at which point the offer will be sent
    return function () { pc.close(); }
}
