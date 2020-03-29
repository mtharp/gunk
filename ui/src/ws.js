export default class WSSession {
    constructor(loc) {
        this.onCandidate = null;
        this.pendOffer = null;
        this.pendAnswer = null;
        let wsURL = (loc.protocol == "https:" ? "wss:" : "ws:") + "//" + loc.hostname;
        if (loc.port != "") {
            wsURL += ":" + loc.port;
        }
        this.wsURL = wsURL + "/ws";
        this.ws = new WebSocket(this.wsURL);
        this.ws.addEventListener("message", (ev) => this.recvMsg(ev.data));
    }

    sendMsg(obj) {
        this.ws.send(JSON.stringify(obj));
        console.log("sent", obj);
    }

    recvMsg(data) {
        let msg = JSON.parse(data);
        console.log("received", msg);
        switch (msg.type) {
            case "offer":
                if (this.pendOffer) {
                    this.pendOffer(msg.sdp);
                }
                this.pendOffer = null;
                break;
            case "answer":
                if (this.pendAnswer) {
                    this.pendAnswer(msg.sdp);
                }
                this.pendAnswer = null;
                break;
            case "candidate":
                if (this.onCandidate) {
                    this.onCandidate(msg.candidate);
                }
                break;
        }
    }

    play(channel) {
        let p = new Promise((resolve) => { this.pendOffer = resolve });
        this.sendMsg({ type: "play", name: channel });
        return p;
    }

    offer(channel, offer) {
        let p = new Promise((resolve) => { this.pendAnswer = resolve });
        this.sendMsg({
            type: "offer",
            name: channel,
            sdp: offer,
        });
        return p;
    }

    answer(answer) {
        this.sendMsg({ type: "answer", sdp: answer });
    }

    candidate(candidate) {
        if (!candidate || !candidate.candidate) {
            return;
        }
        this.sendMsg({ type: "candidate", candidate: candidate });
    }

    stop() {
        this.pendAnser = null;
        this.onCandidate = null;
        this.sendMsg({ type: "stop" });
    }
}
