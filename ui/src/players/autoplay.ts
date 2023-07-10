// play video when ready and restore and save volume
export default function autoplay(video: HTMLVideoElement) {
  video.onvolumechange = null;
  video.muted = true;
  let forcedMute = false;
  video.onplay = function () {
    // update saved volume once playing, after we're done testing whether we can unmute
    video.onvolumechange = () => {
      localStorage.setItem("unmute", String(!video.muted));
      localStorage.setItem("volume", String(Math.round(video.volume * 100)));
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
    video.oncanplay = null;
    // restore saved volume
    const vol = Number(localStorage.getItem("volume"));
    if (vol) {
      video.volume = vol / 100;
    }
    if (localStorage.getItem("unmute") !== "false") {
      video.muted = false;
    }
    video.play().catch(() => {
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
