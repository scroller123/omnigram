import { useState, useEffect } from 'react';
import { TelegramClient } from 'telegram';
import { StorageSession, getClient, resetClient } from './telegram/client';
import { CredentialsForm } from './components/CredentialsForm';
import { LoginForm } from './components/LoginForm';
import { DialogsTable } from './components/DialogsTable';

function App() {
  const [apiId, setApiId] = useState<number | null>(null);
  const [apiHash, setApiHash] = useState<string | null>(null);
  const [client, setClient] = useState<TelegramClient | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [checkingAuth, setCheckingAuth] = useState(true);

  // Load credentials on mount
  useEffect(() => {
    const savedId = localStorage.getItem('tg_api_id');
    const savedHash = localStorage.getItem('tg_api_hash');
    const sessionStr = localStorage.getItem('tg_session_string');

    if (savedId && savedHash) {
      setApiId(parseInt(savedId, 10));
      setApiHash(savedHash);
      
      const tgClient = getClient(parseInt(savedId, 10), savedHash);
      setClient(tgClient);
      
      // If we have a session string, let's verify if we are really logged in
      if (sessionStr) {
        tgClient.connect().then(async () => {
          const isAuth = await tgClient.checkAuthorization();
          setIsAuthenticated(isAuth);
          setCheckingAuth(false);
        }).catch((e) => {
          console.error("Auth check failed:", e);
          setCheckingAuth(false);
        });
      } else {
        setCheckingAuth(false);
      }
    } else {
      setCheckingAuth(false);
    }
  }, []);

  const handleCredentialsSaved = (id: number, hash: string) => {
    setApiId(id);
    setApiHash(hash);
    const tgClient = getClient(id, hash);
    setClient(tgClient);
  };

  const handleClearCredentials = () => {
    localStorage.removeItem('tg_api_id');
    localStorage.removeItem('tg_api_hash');
    if (client) {
      (client.session as StorageSession).clear();
      resetClient();
    }
    setApiId(null);
    setApiHash(null);
    setClient(null);
    setIsAuthenticated(false);
  };

  const handleLogout = async () => {
    if (client) {
      await client.invoke(new (client as any).api.auth.LogOut());
      (client.session as StorageSession).clear();
    }
    setIsAuthenticated(false);
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
          <p className="subtitle">Securely connect and view your dialogs using pure JS MTProto.</p>
        </header>

        {checkingAuth ? (
          <div className="flex-center m-lg">
            <div className="spinner"></div>
            <p>Checking session...</p>
          </div>
        ) : !apiId || !apiHash ? (
          <div className="flex-center-y">
            <CredentialsForm onSaved={handleCredentialsSaved} />
          </div>
        ) : !isAuthenticated && client ? (
          <div className="flex-center-y">
            <LoginForm 
              client={client} 
              onLoginSuccess={() => setIsAuthenticated(true)} 
              onLogout={handleClearCredentials} 
            />
          </div>
        ) : isAuthenticated && client ? (
          <DialogsTable client={client} onLogout={handleLogout} />
        ) : null}
      </main>
    </div>
  );
}

export default App;
