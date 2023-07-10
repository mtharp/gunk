import { defineComponent } from "vue";
import { h } from "vue";

const MINUTE = 60;
const HOUR = MINUTE * 60;
const DAY = HOUR * 24;
const WEEK = DAY * 7;
const MONTH = DAY * 30;
const YEAR = DAY * 365;

const currentLocale = [
  ["%s second ago", "%s seconds ago"],
  ["%s minute ago", "%s minutes ago"],
  ["%s hour ago", "%s hours ago"],
  ["%s day ago", "%s days ago"],
  ["%s week ago", "%s weeks ago"],
  ["%s month ago", "%s months ago"],
  ["%s year ago", "%s years ago"],
];

function pluralOrSingular(data: number, locale: string[]) {
  const count = Math.round(data);
  return count > 1
    ? locale[1].replace(/%s/, count.toString())
    : locale[0].replace(/%s/, count.toString());
}

export default defineComponent({
  props: {
    since: {
      type: Number,
      required: true,
    },
    autoUpdate: Number,
  },
  data() {
    return {
      now: new Date().getTime(),
      interval: null as number | null,
    };
  },
  computed: {
    sinceTime() {
      return new Date(this.since).getTime();
    },
    timeForTitle() {
      return new Date(this.sinceTime).toLocaleString();
    },
    timeago() {
      const seconds = this.now / 1000 - this.sinceTime / 1000;

      const ret =
        seconds <= 5
          ? "just now"
          : seconds < MINUTE
          ? pluralOrSingular(seconds, currentLocale[0])
          : seconds < HOUR
          ? pluralOrSingular(seconds / MINUTE, currentLocale[1])
          : seconds < DAY
          ? pluralOrSingular(seconds / HOUR, currentLocale[2])
          : seconds < WEEK
          ? pluralOrSingular(seconds / DAY, currentLocale[3])
          : seconds < MONTH
          ? pluralOrSingular(seconds / WEEK, currentLocale[4])
          : seconds < YEAR
          ? pluralOrSingular(seconds / MONTH, currentLocale[5])
          : pluralOrSingular(seconds / YEAR, currentLocale[6]);

      return ret;
    },
  },
  mounted() {
    if (this.autoUpdate) {
      this.update();
    }
  },
  render() {
    return h(
      "time",
      {
        attrs: {
          datetime: new Date(this.since),
          title: this.timeForTitle,
        },
      },
      this.timeago
    );
  },
  methods: {
    update() {
      if (!this.autoUpdate) {
        return;
      }
      const period = this.autoUpdate * 1000;
      this.interval = setInterval(() => {
        this.now = new Date().getTime();
      }, period);
    },
    stopUpdate() {
      if (this.interval) {
        clearInterval(this.interval);
        this.interval = null;
      }
    },
  },
  beforeUnmount() {
    this.stopUpdate();
  },
});
