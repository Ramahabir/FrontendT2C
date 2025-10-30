import { useState, useEffect } from 'react';
import { GetSubmissions, EndSession, GetCurrentUser, GetSensorReading, ConfirmSensorSubmission } from "../../wailsjs/go/main/App";
import './Dashboard.css';

function Dashboard({ user, onLogout, onUpdateUser }) {
    const [submissions, setSubmissions] = useState([]);
    const [message, setMessage] = useState('');
    const [loading, setLoading] = useState(false);
    const [currentBalance, setCurrentBalance] = useState(user.balance);
    const [scanning, setScanning] = useState(false);
    const [sensorData, setSensorData] = useState(null);

    useEffect(() => {
        loadSubmissions();
    }, []);

    const loadSubmissions = async () => {
        try {
            const response = await GetSubmissions();
            if (response.success) {
                setSubmissions(response.data);
            }
        } catch (err) {
            console.error('Failed to load submissions:', err);
        }
    };

    const handleStartScan = async () => {
        setMessage('');
        setScanning(true);
        setSensorData(null);

        // Simulate sensor scanning delay
        setTimeout(async () => {
            try {
                const response = await GetSensorReading();
                if (response.success) {
                    setSensorData(response.data);
                    setMessage('Item detected! Review and confirm to submit.');
                } else {
                    setMessage(response.message || 'Sensor detection failed. Please try again.');
                }
            } catch (err) {
                setMessage('Sensor error. Please try again.');
            } finally {
                setScanning(false);
            }
        }, 2000); // 2 second scanning simulation
    };

    const handleConfirmSubmission = async () => {
        if (!sensorData) return;

        setLoading(true);
        setMessage('');

        try {
            const response = await ConfirmSensorSubmission(
                sensorData.material,
                sensorData.weight
            );
            
            if (response.success) {
                setMessage(response.message);
                setCurrentBalance(response.data.newBalance);
                
                // Update user object
                const updatedUser = await GetCurrentUser();
                if (updatedUser.success) {
                    onUpdateUser(updatedUser.data);
                }
                
                // Reload submissions
                loadSubmissions();
                
                // Reset sensor data
                setSensorData(null);
                
                // Clear message after 3 seconds
                setTimeout(() => setMessage(''), 3000);
            } else {
                setMessage(response.message);
            }
        } catch (err) {
            setMessage('Submission failed. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    const handleCancelScan = () => {
        setSensorData(null);
        setMessage('');
    };

    const handleEndSession = async () => {
        if (window.confirm('Are you sure you want to end this recycling session?')) {
            setLoading(true);
            try {
                const response = await EndSession();
                if (response.success) {
                    setMessage('Session ended successfully. Returning to QR screen...');
                    setTimeout(() => {
                        onLogout();
                    }, 1500);
                } else {
                    setMessage(response.message || 'Failed to end session');
                }
            } catch (err) {
                setMessage('Error ending session');
            } finally {
                setLoading(false);
            }
        }
    };

    const formatCurrency = (amount) => {
        return new Intl.NumberFormat('id-ID', {
            style: 'currency',
            currency: 'IDR',
            minimumFractionDigits: 0,
        }).format(amount);
    };

    const getMaterialIcon = (material) => {
        switch(material) {
            case 'plastic': return '♻️';
            case 'metal': return '⚙️';
            case 'paper': return '📄';
            case 'glass': return '🔷';
            default: return '🗑️';
        }
    };

    return (
        <div className="dashboard-container">
            <header className="dashboard-header">
                <div>
                    <h1>🗑️ Trash 2 Cash</h1>
                    <p>Welcome, {user.name}!</p>
                    <span className="session-badge">🟢 Active Session</span>
                </div>
                <div className="header-actions">
                    <button onClick={handleEndSession} className="btn-end-session" disabled={loading}>
                        End Session
                    </button>
                </div>
            </header>

            <div className="balance-card">
                <h2>Your Balance</h2>
                <div className="balance-amount">{formatCurrency(currentBalance)}</div>
            </div>

            <div className="submit-section">
                <h2>🔍 Sensor-Based Submission</h2>
                
                {message && (
                    <div className={`message ${message.includes('successful') || message.includes('detected') ? 'success' : 'error'}`}>
                        {message}
                    </div>
                )}

                {!scanning && !sensorData && (
                    <div className="scan-prompt">
                        <p>Place your recyclable item in the sensor area and click scan</p>
                        <button 
                            onClick={handleStartScan} 
                            className="btn-scan"
                            disabled={loading}
                        >
                            🔍 Start Sensor Scan
                        </button>
                    </div>
                )}

                {scanning && (
                    <div className="scanning-animation">
                        <div className="spinner"></div>
                        <p>Scanning item...</p>
                        <p className="scanning-text">Detecting material and weight...</p>
                    </div>
                )}

                {sensorData && !scanning && (
                    <div className="sensor-result">
                        <h3>✅ Item Detected</h3>
                        <div className="sensor-details">
                            <div className="sensor-item">
                                <span className="sensor-label">Material:</span>
                                <span className="sensor-value">
                                    {getMaterialIcon(sensorData.material)} {sensorData.material.toUpperCase()}
                                </span>
                            </div>
                            <div className="sensor-item">
                                <span className="sensor-label">Weight:</span>
                                <span className="sensor-value">{sensorData.weight.toFixed(2)} kg</span>
                            </div>
                            <div className="sensor-item reward-item">
                                <span className="sensor-label">Reward:</span>
                                <span className="sensor-value reward-value">
                                    {formatCurrency(sensorData.reward)}
                                </span>
                            </div>
                        </div>
                        <div className="action-buttons">
                            <button 
                                onClick={handleConfirmSubmission} 
                                className="btn-confirm"
                                disabled={loading}
                            >
                                {loading ? 'Processing...' : '✓ Confirm & Submit'}
                            </button>
                            <button 
                                onClick={handleCancelScan} 
                                className="btn-cancel"
                                disabled={loading}
                            >
                                ✗ Cancel
                            </button>
                        </div>
                    </div>
                )}
            </div>

            <div className="history-section">
                <h2>Submission History</h2>
                {submissions.length === 0 ? (
                    <p className="no-data">No submissions yet. Start recycling!</p>
                ) : (
                    <div className="history-table">
                        <table>
                            <thead>
                                <tr>
                                    <th>Date</th>
                                    <th>Material</th>
                                    <th>Weight (kg)</th>
                                    <th>Reward</th>
                                </tr>
                            </thead>
                            <tbody>
                                {submissions.map((sub) => (
                                    <tr key={sub.id}>
                                        <td>{sub.createdAt}</td>
                                        <td className="capitalize">
                                            {getMaterialIcon(sub.material)} {sub.material}
                                        </td>
                                        <td>{sub.weight.toFixed(2)}</td>
                                        <td className="reward">{formatCurrency(sub.reward)}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    );
}

export default Dashboard;
