import { ChannelInfo } from "@/store/channels";

const initialDelay = 100;
const maxDelay = 10000;

export default class WSSession {
  wsURL: string;
  last: number;
  session = "";
  delay = initialDelay;

  ping?: number;
  timer?: number;
  ws?: WebSocket;

  onChannel?: (ch: ChannelInfo) => void;
  onCandidate?: (cand: RTCIceCandidateInit) => void;
  pendOffer?: (offer: RTCSessionDescriptionInit) => void;

  constructor (loc: Location) {
    let wsURL = (loc.protocol === 'https:' ? 'wss:' : 'ws:') + '//' + loc.hostname;
    if (loc.port) {
      wsURL += ':' + loc.port;
    }
    this.wsURL = wsURL + '/ws';
    this.delay = initialDelay;
    this.last = performance.now();
    this.ping = window.setInterval(() => this.doPing(), 10000);
    this.connect();
  }

  connect () {
    this.timer = undefined;
    let url = this.wsURL;
    if (this.session) {
      url += '?session=' + this.session;
    }
    console.log('websocket connecting to', url);
    this.ws = new WebSocket(url);
    this.ws.addEventListener('message', (ev) => this.recvMsg(ev));
    this.ws.addEventListener('error', (ev) => this.onError(ev));
  }

  close () {
    if (this.ws) {
      this.ws.close();
      this.ws = undefined;
    }
    if (this.timer) {
      window.clearTimeout(this.timer);
      this.timer = undefined;
    }
    if (this.ping) {
      window.clearInterval(this.ping);
      this.ping = undefined;
    }
  }

  onError (ev: Event | string) {
    if (this.ws) {
      this.ws.close();
      this.ws = undefined;
    }
    console.log('websocket disconnected:', ev);
    const delay = this.delay;
    this.delay *= 1.618;
    if (this.delay > maxDelay) {
      this.delay = maxDelay;
    }
    if (this.timer === null) {
      console.log('trying again in', delay / 1000, 'seconds');
      this.timer = window.setTimeout(() => this.connect(), delay);
    }
  }

  sendMsg (obj: any) {
    if (!this.ws) {
      return;
    }
    const lastReceived = (performance.now() - this.last) / 1000;
    if (lastReceived > 12) {
      this.onError('no message received in ' + lastReceived + ' seconds');
      return;
    }
    this.ws.send(JSON.stringify(obj));
  }

  recvMsg (ev: MessageEvent) {
    const msg = JSON.parse(ev.data);
    this.last = performance.now();
    switch (msg.type) {
      case 'connected':
        this.session = msg.id;
        this.delay = initialDelay;
        break;
      case 'offer':
        if (this.pendOffer) {
          this.pendOffer(msg.sdp);
        }
        this.pendOffer = undefined;
        break;
      case 'candidate':
        if (this.onCandidate) {
          this.onCandidate(msg.candidate);
        }
        break;
      case 'channel':
        if (this.onChannel) {
          this.onChannel(msg.channel);
        }
        break;
    }
  }

  doPing () {
    this.sendMsg({ type: 'ping' });
  }

  play (channel: string): Promise<RTCSessionDescriptionInit> {
    const p = new Promise<RTCSessionDescriptionInit>((resolve) => { this.pendOffer = resolve; });
    this.sendMsg({ type: 'play', name: channel });
    return p;
  }

  answer (answer: RTCSessionDescriptionInit) {
    this.sendMsg({ type: 'answer', sdp: answer });
  }

  candidate (candidate: RTCIceCandidateInit) {
    if (!candidate || !candidate.candidate) {
      return;
    }
    this.sendMsg({ type: 'candidate', candidate: candidate });
  }

  stop () {
    this.onCandidate = undefined;
    this.sendMsg({ type: 'stop' });
  }
}
