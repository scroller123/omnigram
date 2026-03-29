import React, { useState } from 'react';

interface CredentialsFormProps {
  onSaved: (apiId: number, apiHash: string) => void;
}

export const CredentialsForm: React.FC<CredentialsFormProps> = ({ onSaved }) => {
  const [apiId, setApiId] = useState('');
  const [apiHash, setApiHash] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!apiId || !apiHash) {
      setError('Both API ID and API Hash are required');
      return;
    }
    const idAsNumber = parseInt(apiId, 10);
    if (isNaN(idAsNumber)) {
      setError('API ID must be a number');
      return;
    }

    localStorage.setItem('tg_api_id', apiId);
    localStorage.setItem('tg_api_hash', apiHash);
    onSaved(idAsNumber, apiHash);
  };

  return (
    <div className="card glass-card fade-in">
      <h2>Telegram App Credentials</h2>
      <p className="subtitle">Enter your API ID and Hash from my.telegram.org</p>
      
      {error && <div className="error-message">{error}</div>}
      
      <form onSubmit={handleSubmit} className="auth-form">
        <div className="form-group">
          <label htmlFor="apiId">API ID</label>
          <input
            id="apiId"
            type="text"
            value={apiId}
            onChange={(e) => setApiId(e.target.value)}
            placeholder="e.g. 1234567"
            className="input-field"
          />
        </div>
        
        <div className="form-group">
          <label htmlFor="apiHash">API Hash</label>
          <input
            id="apiHash"
            type="text"
            value={apiHash}
            onChange={(e) => setApiHash(e.target.value)}
            placeholder="e.g. 0123456789abcdef0123456789abcdef"
            className="input-field"
          />
        </div>
        
        <button type="submit" className="primary-button">
          Save Credentials
        </button>
      </form>
    </div>
  );
};
