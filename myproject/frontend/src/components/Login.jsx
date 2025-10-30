import { useState, useEffect } from 'react';
import { RequestSessionToken, CheckSessionStatus } from "../../wailsjs/go/main/App";
import './Auth.css';

function Login({ onLogin, onSwitchToRegister }) {
    const [qrCode, setQrCode] = useState('');
    const [sessionToken, setSessionToken] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [sessionStatus, setSessionStatus] = useState('pending');

    useEffect(() => {
        requestNewSession();
    }, []);

    const requestNewSession = async () => {
        setError('');
        setLoading(true);
        setSessionStatus('pending');
        try {
            const response = await RequestSessionToken();
            if (response.success) {
                setQrCode(response.data.qrCode);
                setSessionToken(response.data.sessionToken);
                setSessionStatus(response.data.status || 'pending');
                startPolling(response.data.sessionToken);
            } else {
                setError(response.message);
            }
        } catch (err) {
            setError('Failed to request session token. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    const startPolling = (token) => {
        const pollInterval = setInterval(async () => {
            try {
                const response = await CheckSessionStatus();
                if (response.success && response.data.status === 'connected') {
                    clearInterval(pollInterval);
                    setSessionStatus('connected');
                    onLogin(response.data);
                } else if (response.success && response.data.status === 'active') {
                    clearInterval(pollInterval);
                    setSessionStatus('active');
                    onLogin(response.data);
                } else if (!response.success || response.data?.status === 'expired') {
                    clearInterval(pollInterval);
                    setSessionStatus('expired');
                    setError('Session expired. Generating new session...');
                    setTimeout(requestNewSession, 2000);
                }
            } catch (err) {
                console.error('Polling error:', err);
            }
        }, 2000); // Poll every 2 seconds

        // Clean up on unmount
        return () => clearInterval(pollInterval);
    };

    return (
        <div className="auth-container">
            <div className="auth-card qr-login-card">
                <h1>🗑️ Trash 2 Cash</h1>
                <h2>Station QR Code</h2>
                
                {error && <div className="error-message">{error}</div>}
                
                <div className="qr-section">
                    {loading ? (
                        <div className="qr-loading">
                            <div className="spinner-small"></div>
                            <p>Requesting session...</p>
                        </div>
                    ) : (
                        <>
                            <div className="qr-code-container">
                                {qrCode && <img src={qrCode} alt="Session QR Code" className="qr-code-image" />}
                            </div>
                            <div className="session-status">
                                {sessionStatus === 'pending' && (
                                    <>
                                        <p className="qr-instructions">
                                            📱 Scan this QR code with the Trash2Cash mobile app
                                        </p>
                                        <p className="qr-sub-instructions">
                                            The session code will expire in 5 minutes
                                        </p>
                                        <div className="status-indicator pending">
                                            <span className="status-dot"></span>
                                            Waiting for user to scan...
                                        </div>
                                    </>
                                )}
                                {sessionStatus === 'connected' && (
                                    <div className="status-indicator connected">
                                        <span className="status-dot"></span>
                                        ✓ User connected! Loading dashboard...
                                    </div>
                                )}
                                {sessionStatus === 'expired' && (
                                    <div className="status-indicator expired">
                                        <span className="status-dot"></span>
                                        Session expired
                                    </div>
                                )}
                            </div>
                        </>
                    )}
                </div>

                <button 
                    onClick={requestNewSession} 
                    className="btn-secondary"
                    disabled={loading}
                >
                    Generate New Session
                </button>

                <p className="switch-auth">
                    Need to set up the system?{' '}
                    <span onClick={onSwitchToRegister} className="link">Configure here</span>
                </p>
            </div>
        </div>
    );
}

export default Login;
