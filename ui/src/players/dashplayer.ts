import dashjs, { MediaPlayer, MediaPlayerSettingClass } from "dashjs";
import autoplay from "./autoplay";

export default class DASHPlayer {
  video: HTMLVideoElement;
  stream: dashjs.MediaPlayerClass;

  constructor(
    video: HTMLVideoElement,
    webURL: string,
    lowLatencyMode: boolean
  ) {
    autoplay(video);
    this.video = video;
    this.stream = MediaPlayer().create();
    this.stream.initialize();
    const settings: MediaPlayerSettingClass = {
      streaming: {
        liveCatchup: {
          playbackRate: {
            max: 0.1,
          },
        },
        utcSynchronization: {
          timeBetweenSyncAttempts: 30,
        },
      },
    };
    if (lowLatencyMode) {
      settings.streaming!.delay = { liveDelay: 2 };
    } else {
      settings.streaming!.delay = { liveDelayFragmentCount: 2 };
    }
    this.stream.updateSettings(settings);
    this.stream.setAutoPlay(true);
    this.stream.attachSource(webURL);
    this.stream.attachView(video);
  }

  destroy() {
    this.stream.reset();
  }

  seekLive() {
    this.stream.seek(this.stream.duration());
    this.stream.play();
  }

  latencyTo() {
    return this.stream.getCurrentLiveLatency();
  }
}
