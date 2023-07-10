import autoplay from "./autoplay";

// test for iOS, which only supports RTC and native HLS
export function nativeRequired() {
  return (
    /iPad|iPhone|iPod/.test(navigator.userAgent) && !("MSStream" in window)
  );
}

export default class NativePlayer {
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
