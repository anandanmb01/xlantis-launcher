import { useState } from 'react';
import './App.css';
import { WireGuardConnect, WireGuardDisconnect } from "../wailsjs/go/main/App";
import { main } from "../wailsjs/go/models";

function App() {
    const [config, setConfig] = useState({
        privateIp: '',
        listenPort: '',
        host: '',
        endpoint: '',
        peerPubKey: '',
        privateKey: ''
    });
    const [status, setStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'disconnecting'>('disconnected');
    const [error, setError] = useState<string>('');

    const updateField = (field: string) => (e: React.ChangeEvent<HTMLInputElement>) => {
        setConfig(prev => ({ ...prev, [field]: e.target.value }));
        setError('');
    };

    const handleConnect = async () => {
        // Validate all fields
        if (!config.privateIp || !config.listenPort || !config.host || 
            !config.endpoint || !config.peerPubKey || !config.privateKey) {
            setError('All fields are required');
            return;
        }

        setStatus('connecting');
        setError('');

        try {
            const wireguardConfig = new main.WireGuardConfig({
                privateIp: config.privateIp,
                listenPort: parseInt(config.listenPort),
                host: config.host,
                endpoint: config.endpoint,
                peerPubKey: config.peerPubKey,
                privateKey: config.privateKey
            });

            const result = await WireGuardConnect(wireguardConfig);
            if (result) {
                setStatus('connected');
            } else {
                setStatus('disconnected');
                setError('Failed to connect');
            }
        } catch (err: any) {
            setStatus('disconnected');
            setError(err.message || 'Connection failed');
        }
    };

    const handleDisconnect = async () => {
        setStatus('disconnecting');
        setError('');

        try {
            const result = await WireGuardDisconnect();
            if (result) {
                setStatus('disconnected');
            } else {
                setError('Failed to disconnect');
            }
        } catch (err: any) {
            setError(err.message || 'Disconnection failed');
        } finally {
            setStatus('disconnected');
        }
    };

    return (
        <div id="App">
            <div className="container">
                <h1>WireGuard VPN</h1>
                
                <div className="status-bar">
                    <div className={`status-indicator ${status}`}>
                        <span className="status-dot"></span>
                        <span className="status-text">
                            {status === 'connected' ? 'Connected' : 
                             status === 'connecting' ? 'Connecting...' :
                             status === 'disconnecting' ? 'Disconnecting...' : 
                             'Disconnected'}
                        </span>
                    </div>
                </div>

                {error && (
                    <div className="error-message">{error}</div>
                )}

                <div className="form-container">
                    <div className="form-group">
                        <label htmlFor="privateIp">Private IP</label>
                        <input
                            id="privateIp"
                            type="text"
                            className="input"
                            placeholder="10.0.0.2/24"
                            value={config.privateIp}
                            onChange={updateField('privateIp')}
                            disabled={status === 'connected' || status === 'connecting'}
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="listenPort">Listen Port</label>
                        <input
                            id="listenPort"
                            type="number"
                            className="input"
                            placeholder="51820"
                            value={config.listenPort}
                            onChange={updateField('listenPort')}
                            disabled={status === 'connected' || status === 'connecting'}
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="host">Host</label>
                        <input
                            id="host"
                            type="text"
                            className="input"
                            placeholder="vpn.example.com"
                            value={config.host}
                            onChange={updateField('host')}
                            disabled={status === 'connected' || status === 'connecting'}
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="endpoint">Endpoint</label>
                        <input
                            id="endpoint"
                            type="text"
                            className="input"
                            placeholder="vpn.example.com:51820"
                            value={config.endpoint}
                            onChange={updateField('endpoint')}
                            disabled={status === 'connected' || status === 'connecting'}
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="peerPubKey">Peer Public Key</label>
                        <input
                            id="peerPubKey"
                            type="text"
                            className="input"
                            placeholder="Enter peer public key"
                            value={config.peerPubKey}
                            onChange={updateField('peerPubKey')}
                            disabled={status === 'connected' || status === 'connecting'}
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="privateKey">Private Key</label>
                        <input
                            id="privateKey"
                            type="password"
                            className="input"
                            placeholder="Enter your private key"
                            value={config.privateKey}
                            onChange={updateField('privateKey')}
                            disabled={status === 'connected' || status === 'connecting'}
                        />
                    </div>

                    <div className="button-group">
                        <button
                            className="btn btn-connect"
                            onClick={handleConnect}
                            disabled={status === 'connected' || status === 'connecting' || status === 'disconnecting'}
                        >
                            Connect
                        </button>
                        <button
                            className="btn btn-disconnect"
                            onClick={handleDisconnect}
                            disabled={status === 'disconnected' || status === 'connecting' || status === 'disconnecting'}
                        >
                            Disconnect
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}

export default App;
