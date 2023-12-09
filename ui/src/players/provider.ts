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

export async function choosePlayer(
  video: HTMLVideoElement,
  props: PlayerProperties,
): Promise<PlayerProvider> {
  if (props.rtcActive) {
    return new RTCPlayer(video, props.ch.name);
  } else if (nativeRequired()) {
    return new NativePlayer(video, props.ch.native_url);
  } else if (props.ch.web_url.endsWith(".mpd")) {
    const { DASHPlayer } = await import("./dashplayer");
    return new DASHPlayer(video, props.ch.web_url, props.lowLatency);
  } else {
    const { HLSPlayer } = await import("./hlsplayer");
    return new HLSPlayer(video, props.ch.web_url, props.lowLatency);
  }
}
