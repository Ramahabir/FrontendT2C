import { useState, useEffect } from 'react';
import './App.css';
import Login from './components/Login';
import Register from './components/Register';
import Dashboard from './components/Dashboard';
import { GetCurrentUser } from "../wailsjs/go/main/App";

function App() {
    const [currentView, setCurrentView] = useState('login');
    const [user, setUser] = useState(null);

    useEffect(() => {
        // Check if user is already logged in
        GetCurrentUser().then((response) => {
            if (response.success && response.data) {
                setUser(response.data);
                setCurrentView('dashboard');
            }
        }).catch(err => {
            console.log('No user logged in');
        });
    }, []);

    const handleLogin = (userData) => {
        setUser(userData);
        setCurrentView('dashboard');
    };

    const handleLogout = () => {
        setUser(null);
        setCurrentView('login');
    };

    return (
        <div id="App">
            {currentView === 'login' && (
                <Login 
                    onLogin={handleLogin} 
                    onSwitchToRegister={() => setCurrentView('register')} 
                />
            )}
            {currentView === 'register' && (
                <Register 
                    onRegisterSuccess={() => setCurrentView('login')} 
                    onSwitchToLogin={() => setCurrentView('login')} 
                />
            )}
            {currentView === 'dashboard' && user && (
                <Dashboard user={user} onLogout={handleLogout} onUpdateUser={setUser} />
            )}
        </div>
    );
}

export default App;
