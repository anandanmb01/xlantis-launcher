export namespace main {
	
	export class WireGuardConfig {
	    privateIp: string;
	    listenPort: number;
	    host: string;
	    endpoint: string;
	    peerPubKey: string;
	    privateKey: string;
	
	    static createFrom(source: any = {}) {
	        return new WireGuardConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.privateIp = source["privateIp"];
	        this.listenPort = source["listenPort"];
	        this.host = source["host"];
	        this.endpoint = source["endpoint"];
	        this.peerPubKey = source["peerPubKey"];
	        this.privateKey = source["privateKey"];
	    }
	}

}

