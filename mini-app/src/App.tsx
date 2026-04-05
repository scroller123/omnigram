import { useState, useEffect } from 'react';
import { authApi } from './api/auth';
import { LoginForm } from './components/LoginForm';
import { DialogsTable } from './components/DialogsTable';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [checkingAuth, setCheckingAuth] = useState(true);
  const [user, setUser] = useState<any>(null);

  // Load user on mount
  useEffect(() => {
    const token = localStorage.getItem('tg_jwt_token');
    if (token) {
      authApi.getMe()
        .then((userData) => {
          setUser(userData);
          setIsAuthenticated(true);
          setCheckingAuth(false);
        })
        .catch(() => {
          setIsAuthenticated(false);
          setCheckingAuth(false);
        });
    } else {
      setCheckingAuth(false);
    }
  }, []);

  const handleLoginSuccess = () => {
    authApi.getMe().then((userData) => {
      setUser(userData);
      setIsAuthenticated(true);
    });
  };

  const handleLogout = async () => {
    await authApi.logout();
    setIsAuthenticated(false);
    setUser(null);
  };

  return (
    <div className="app-container">
      <div className="background-shapes">
        <div className="shape shape-1"></div>
        <div className="shape shape-2"></div>
        <div className="shape shape-3"></div>
      </div>
      
      <main className="content">
        <header className="app-header">
          <h1>Telegram MTProto Client</h1>
          {user ? (
            <p className="subtitle">Logged in as {user.first_name} {user.last_name} (@{user.username})</p>
          ) : (
            <p className="subtitle">Securely connect and view your dialogs using Backend-API MTProto.</p>
          )}
        </header>

        {checkingAuth ? (
          <div className="flex-center m-lg">
            <div className="spinner"></div>
            <p>Checking session...</p>
          </div>
        ) : !isAuthenticated ? (
          <div className="flex-center-y">
            <LoginForm 
              onLoginSuccess={handleLoginSuccess} 
            />
          </div>
        ) : (
          <DialogsTable onLogout={handleLogout} />
        )}
      </main>
    </div>
  );
}

export default App;
