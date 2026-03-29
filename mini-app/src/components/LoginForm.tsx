import React, { useState } from 'react';
import { TelegramClient } from 'telegram';
import { Api } from 'telegram';

interface LoginFormProps {
  client: TelegramClient;
  onLoginSuccess: () => void;
  onLogout: () => void;
}

export const LoginForm: React.FC<LoginFormProps> = ({ client, onLoginSuccess, onLogout }) => {
  const [step, setStep] = useState<'phone' | 'code' | 'password'>('phone');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [phoneCodeHash, setPhoneCodeHash] = useState('');
  const [phoneCode, setPhoneCode] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const requestPhoneCode = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!phoneNumber) return;
    
    setError('');
    setLoading(true);
    try {
      await client.connect();
      const { phoneCodeHash } = await client.sendCode(
        {
          apiId: client.apiId,
          apiHash: client.apiHash,
        },
        phoneNumber
      );
      setPhoneCodeHash(phoneCodeHash);
      setStep('code');
    } catch (err: any) {
      console.error(err);
      setError(err.message || 'Failed to request code');
    } finally {
      setLoading(false);
    }
  };

  const submitCode = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!phoneCode) return;

    setError('');
    setLoading(true);
    try {
      await client.invoke(
        new Api.auth.SignIn({
          phoneNumber,
          phoneCodeHash,
          phoneCode,
        })
      );
      // save session
      (client.session as any).save();
      onLoginSuccess();
    } catch (err: any) {
      if (err.message.includes('SESSION_PASSWORD_NEEDED')) {
        setStep('password');
      } else {
        console.error(err);
        setError(err.message || 'Invalid code');
      }
    } finally {
      setLoading(false);
    }
  };

  const submitPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password) return;

    setError('');
    setLoading(true);
    try {
      await client.signInWithPassword(
        {
          apiId: client.apiId,
          apiHash: client.apiHash,
        },
        {
          password: async () => password,
          onError: (e) => {
            setError(e.message);
            setLoading(false);
          }
        }
      );
      (client.session as any).save();
      onLoginSuccess();
    } catch (err: any) {
      console.error(err);
      setError(err.message || 'Invalid password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="card glass-card fade-in">
      <div className="flex-between">
        <h2>Login to Telegram</h2>
        <button type="button" onClick={onLogout} className="text-button text-sm">Clear Credentials</button>
      </div>
      
      {error && <div className="error-message">{error}</div>}
      
      {step === 'phone' && (
        <form onSubmit={requestPhoneCode} className="auth-form slide-in">
          <p className="subtitle">Enter your phone number</p>
          <div className="form-group">
            <input
              type="tel"
              value={phoneNumber}
              onChange={(e) => setPhoneNumber(e.target.value)}
              placeholder="+1234567890"
              className="input-field"
              disabled={loading}
            />
          </div>
          <button type="submit" className="primary-button" disabled={loading || !phoneNumber}>
            {loading ? 'Sending...' : 'Send Code'}
          </button>
        </form>
      )}

      {step === 'code' && (
        <form onSubmit={submitCode} className="auth-form slide-in">
          <p className="subtitle">Enter the code sent to your Telegram app</p>
          <div className="form-group">
            <input
              type="text"
              value={phoneCode}
              onChange={(e) => setPhoneCode(e.target.value)}
              placeholder="12345"
              className="input-field code-input"
              disabled={loading}
            />
          </div>
          <button type="submit" className="primary-button" disabled={loading || !phoneCode}>
            {loading ? 'Verifying...' : 'Login'}
          </button>
        </form>
      )}

      {step === 'password' && (
        <form onSubmit={submitPassword} className="auth-form slide-in">
          <p className="subtitle">Enter your Two-Step Verification Password</p>
          <div className="form-group">
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="input-field"
              disabled={loading}
            />
          </div>
          <button type="submit" className="primary-button" disabled={loading || !password}>
            {loading ? 'Verifying...' : 'Submit'}
          </button>
        </form>
      )}
    </div>
  );
};
