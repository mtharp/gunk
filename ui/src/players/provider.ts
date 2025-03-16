import RTCPlayer from "./rtcplayer";
// import NativePlayer, { nativeRequired } from "./nativeplayer";
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
  // TODO: restore native player

  if (props.rtcActive) {
    const player = new RTCPlayer(video, props.ch.whep_url);
    await player.start();
    return player;
    // } else if (nativeRequired()) {
    //   return new NativePlayer(video, props.ch.native_url);
  } else {
    const { HLSPlayer } = await import("./hlsplayer");
    return new HLSPlayer(video, props.ch.hls_url, props.lowLatency);
  }
}
