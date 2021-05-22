const initialDelay = 100;
const maxDelay = 10000;

export default class WSSession {
  constructor (loc) {
    this.onCandidate = null;
    this.pendOffer = null;
    let wsURL = (loc.protocol === 'https:' ? 'wss:' : 'ws:') + '//' + loc.hostname;
    if (loc.port) {
      wsURL += ':' + loc.port;
    }
    this.wsURL = wsURL + '/ws';
    this.session = '';
    this.delay = initialDelay;
    this.timer = null;
    this.last = performance.now();
    this.ping = window.setInterval(() => this.doPing(), 10000);
    this.onChannel = null;
    this.connect();
  }

  connect () {
    this.timer = null;
    let url = this.wsURL;
    if (this.session) {
      url += '?session=' + this.session;
    }
    console.log('websocket connecting to', url);
    this.ws = new WebSocket(url);
    this.ws.addEventListener('message', (ev) => this.recvMsg(ev.data));
    this.ws.addEventListener('error', (ev) => this.onError(ev));
  }

  close () {
    this.ws.close();
    this.ws = null;
    if (this.timer !== null) {
      window.clearTimeout(this.timer);
      this.timer = null;
    }
    if (this.ping !== null) {
      window.clearInterval(this.ping);
      this.ping = null;
    }
  }

  onError (ev) {
    this.ws.close();
    this.ws = null;
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

  sendMsg (obj) {
    if (this.ws === null) {
      return;
    }
    const lastReceived = (performance.now() - this.last) / 1000;
    if (lastReceived > 12) {
      this.onError('no message received in ' + lastReceived + ' seconds');
      return;
    }
    this.ws.send(JSON.stringify(obj));
  }

  recvMsg (data) {
    const msg = JSON.parse(data);
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
        this.pendOffer = null;
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

  play (channel) {
    const p = new Promise((resolve) => { this.pendOffer = resolve; });
    this.sendMsg({ type: 'play', name: channel });
    return p;
  }

  answer (answer) {
    this.sendMsg({ type: 'answer', sdp: answer });
  }

  candidate (candidate) {
    if (!candidate || !candidate.candidate) {
      return;
    }
    this.sendMsg({ type: 'candidate', candidate: candidate });
  }

  stop () {
    this.onCandidate = null;
    this.sendMsg({ type: 'stop' });
  }
}
