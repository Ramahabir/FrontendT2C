import { useState, useEffect } from 'react';
import { GenerateQRLoginCode, CheckQRLoginStatus } from "../../wailsjs/go/main/App";
import './Auth.css';

function Login({ onLogin, onSwitchToRegister }) {
    const [qrCode, setQrCode] = useState('');
    const [token, setToken] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        generateNewQR();
    }, []);

    const generateNewQR = async () => {
        setError('');
        setLoading(true);
        try {
            const response = await GenerateQRLoginCode();
            if (response.success) {
                setQrCode(response.data.qrCode);
                setToken(response.data.token);
                startPolling(response.data.token);
            } else {
                setError(response.message);
            }
        } catch (err) {
            setError('Failed to generate QR code. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    const startPolling = (sessionToken) => {
        const pollInterval = setInterval(async () => {
            try {
                const response = await CheckQRLoginStatus(sessionToken);
                if (response.success && response.data.status === 'authenticated') {
                    clearInterval(pollInterval);
                    onLogin(response.data);
                } else if (!response.success && response.data?.status === 'expired') {
                    clearInterval(pollInterval);
                    setError('QR code expired. Generating new code...');
                    setTimeout(generateNewQR, 2000);
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
                <h2>QR Code Login</h2>
                
                {error && <div className="error-message">{error}</div>}
                
                <div className="qr-section">
                    {loading ? (
                        <div className="qr-loading">
                            <div className="spinner-small"></div>
                            <p>Generating QR Code...</p>
                        </div>
                    ) : (
                        <>
                            <div className="qr-code-container">
                                {qrCode && <img src={qrCode} alt="QR Code" className="qr-code-image" />}
                            </div>
                            <p className="qr-instructions">
                                Scan this QR code with your mobile device to login
                            </p>
                            <p className="qr-sub-instructions">
                                The QR code will expire in 5 minutes
                            </p>
                        </>
                    )}
                </div>

                <button 
                    onClick={generateNewQR} 
                    className="btn-secondary"
                    disabled={loading}
                >
                    Generate New QR Code
                </button>

                <p className="switch-auth">
                    Don't have an account?{' '}
                    <span onClick={onSwitchToRegister} className="link">Register here</span>
                </p>
            </div>
        </div>
    );
}

export default Login;
