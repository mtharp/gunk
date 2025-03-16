import axios, { type AxiosHeaderValue } from "axios";
import autoplay from "./autoplay";

function unquoteCredential(v: string): string {
  return JSON.parse(`"${v}"`);
}

function linkToIceServers(links: AxiosHeaderValue): RTCIceServer[] {
  if (typeof links !== "string") {
    return [];
  }
  const ret: RTCIceServer[] = [];
  for (const link of links.split(",")) {
    const m = link.match(
      /^ *<(.+?)>; rel="ice-server"(; username="(.*?)"; credential="(.*?)"; credential-type="password")?/i,
    );
    if (!m) {
      continue;
    }
    const srv: RTCIceServer = {
      urls: [m[1]],
    };

    if (m[3] !== undefined) {
      srv.username = unquoteCredential(m[3]);
      srv.credential = unquoteCredential(m[4]);
    }
    ret.push(srv);
  }
  return ret;
}

interface OfferData {
  medias: string[];
  iceUfrag: string;
  icePwd: string;
}

function parseOffer(offer: string | undefined): OfferData {
  const ret: OfferData = {
    iceUfrag: "",
    icePwd: "",
    medias: [],
  };
  if (!offer) {
    return ret;
  }

  for (const line of offer.split("\r\n")) {
    if (line.startsWith("m=")) {
      ret.medias.push(line.slice("m=".length));
    } else if (ret.iceUfrag === "" && line.startsWith("a=ice-ufrag:")) {
      ret.iceUfrag = line.slice("a=ice-ufrag:".length);
    } else if (ret.icePwd === "" && line.startsWith("a=ice-pwd:")) {
      ret.icePwd = line.slice("a=ice-pwd:".length);
    }
  }

  return ret;
}

function enableStereoOpus(section: string): string {
  let opusPayloadFormat = "";
  const lines = section.split("\r\n");

  for (let i = 0; i < lines.length; i++) {
    if (
      lines[i].startsWith("a=rtpmap:") &&
      lines[i].toLowerCase().includes("opus/")
    ) {
      opusPayloadFormat = lines[i].slice("a=rtpmap:".length).split(" ")[0];
      break;
    }
  }

  if (opusPayloadFormat === "") {
    return section;
  }

  for (let i = 0; i < lines.length; i++) {
    if (lines[i].startsWith("a=fmtp:" + opusPayloadFormat + " ")) {
      if (!lines[i].includes("stereo")) {
        lines[i] += ";stereo=1";
      }
      if (!lines[i].includes("sprop-stereo")) {
        lines[i] += ";sprop-stereo=1";
      }
    }
  }

  return lines.join("\r\n");
}

function editOffer(
  offer: RTCSessionDescriptionInit,
): RTCSessionDescriptionInit {
  if (!offer.sdp) {
    return offer;
  }
  const sections = offer.sdp.split("m=");

  for (let i = 0; i < sections.length; i++) {
    const section = sections[i];
    if (section.startsWith("audio")) {
      sections[i] = enableStereoOpus(section);
    }
  }

  offer.sdp = sections.join("m=");
  return offer;
}

function generateSdpFragment(
  od: OfferData,
  candidates: RTCIceCandidate[],
): string {
  const candidatesByMedia: Record<number, RTCIceCandidate[]> = {};
  for (const candidate of candidates) {
    const mid = candidate.sdpMLineIndex;
    if (mid === null) {
      continue;
    }
    if (candidatesByMedia[mid] === undefined) {
      candidatesByMedia[mid] = [];
    }
    candidatesByMedia[mid].push(candidate);
  }

  let frag =
    "a=ice-ufrag:" + od.iceUfrag + "\r\n" + "a=ice-pwd:" + od.icePwd + "\r\n";

  let mid = 0;

  for (const media of od.medias) {
    if (candidatesByMedia[mid] !== undefined) {
      frag += "m=" + media + "\r\n" + "a=mid:" + mid + "\r\n";

      for (const candidate of candidatesByMedia[mid]) {
        frag += "a=" + candidate.candidate + "\r\n";
      }
    }
    mid++;
  }

  return frag;
}

export default class RTCPlayer {
  video: HTMLVideoElement;
  whepUrl: string;

  pc?: RTCPeerConnection;
  sessionUrl?: string;
  offerData?: OfferData;
  queuedCandidates: RTCIceCandidate[];

  constructor(video: HTMLVideoElement, whep_url: string) {
    autoplay(video);
    this.video = video;
    this.whepUrl = whep_url;
    this.queuedCandidates = [];
  }

  async start() {
    // request ICE servers
    console.log("requesting options");
    const resp = await axios.options(this.whepUrl);
    const iceServers = linkToIceServers(resp.headers["link"]);
    console.log("ice servers:", iceServers);

    // accept audio and video
    this.pc = new RTCPeerConnection({
      iceServers: iceServers,
    });
    const direction = "recvonly";
    this.pc.addTransceiver("audio", { direction: direction });
    this.pc.addTransceiver("video", { direction: direction });
    this.pc.onicecandidate = (evt) => this.onLocalCandidate(evt);
    this.pc.oniceconnectionstatechange = () => this.onConnectionState();
    this.pc.ontrack = (evt) => this.onTrack(evt);

    // request an offer
    let offer = await this.pc.createOffer();
    offer = editOffer(offer);
    this.offerData = parseOffer(offer.sdp);
    await this.pc.setLocalDescription(offer);
    await this.sendOffer(offer);
  }

  destroy() {
    this.pc?.close();
    if (this.sessionUrl) {
      axios.delete(this.sessionUrl);
    }
  }

  seekLive() {
    this.video.play();
  }

  latencyTo() {
    return 0;
  }

  private async onConnectionState() {
    if (this.pc) {
      console.log("connection state is", this.pc.iceConnectionState);
    }
  }

  private async onTrack(evt: RTCTrackEvent) {
    console.log("ontrack", evt.streams);
    if (evt.streams.length != 0) {
      this.video.srcObject = evt.streams[0];
    }
  }

  private async onLocalCandidate(evt: RTCPeerConnectionIceEvent) {
    if (evt.candidate === null) {
      return;
    }
    if (this.sessionUrl) {
      await this.sendLocalCandidates([evt.candidate]);
    } else {
      this.queuedCandidates.push(evt.candidate);
    }
  }

  private async sendLocalCandidates(candidates: RTCIceCandidate[]) {
    if (!this.sessionUrl || this.offerData === undefined) {
      return;
    }
    await axios.patch(
      this.sessionUrl,
      generateSdpFragment(this.offerData, candidates),
      {
        headers: {
          "Content-Type": "application/trickle-ice-sdpfrag",
          "If-Match": "*",
        },
      },
    );
  }

  private async sendOffer(offer: RTCSessionDescriptionInit) {
    if (!offer.sdp || !this.pc) {
      throw new Error("missing SDP");
    }

    // send offer
    console.log("sending offer");
    const resp = await axios.post(this.whepUrl, offer.sdp, {
      headers: { "Content-Type": "application/sdp" },
      responseType: "text",
    });

    // get session URL from headers
    const sessionUrl = resp.headers["location"];
    if (typeof sessionUrl !== "string") {
      throw new Error("missing session URL");
    }
    this.sessionUrl = new URL(sessionUrl, this.whepUrl).toString();
    console.log("session url:", this.sessionUrl);

    // handle response
    const sdp = resp.data;
    if (typeof sdp !== "string") {
      throw new Error("invalid session response");
    }
    await this.pc.setRemoteDescription(
      new RTCSessionDescription({
        type: "answer",
        sdp,
      }),
    );

    // send local candidates that were queued while exchanging offers
    const toSend = this.queuedCandidates;
    if (toSend.length != 0) {
      this.queuedCandidates = [];
      await this.sendLocalCandidates(toSend);
    }
  }
}
