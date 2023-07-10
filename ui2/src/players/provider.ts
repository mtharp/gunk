import DASHPlayer from "./dashplayer";
// import HLSPlayer from "./hlsplayer";
import RTCPlayer from "./rtcplayer";
import NativePlayer, { nativeRequired } from "./nativeplayer";
import type { ChannelInfo } from "@/stores/channels";

export interface PlayerProvider {
  destroy(): void;
  seekLive(): void;
  latencyTo(): number;
}

export interface PlayerProperties {
  ch: ChannelInfo;
  rtcActive: boolean;
  lowLatency: boolean;
}

export function choosePlayer(
  video: HTMLVideoElement,
  props: PlayerProperties
): PlayerProvider {
  if (props.rtcActive) {
    return new RTCPlayer(video, props.ch.name);
  } else if (nativeRequired()) {
    return new NativePlayer(video, props.ch.native_url);
  } /*if (props.ch.web_url.endsWith(".mpd"))*/ else {
    return new DASHPlayer(video, props.ch.web_url, props.lowLatency);
  } /* else {
    return new HLSPlayer(video, props.ch.web_url, props.lowLatency);
  }*/
}
