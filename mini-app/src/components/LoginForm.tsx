import React, { useState } from 'react';
import { authApi } from '../api/auth';

interface LoginFormProps {
  onLoginSuccess: (token: string) => void;
}

export const LoginForm: React.FC<LoginFormProps> = ({ onLoginSuccess }) => {
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
      const data = await authApi.sendCode(phoneNumber);
      setPhoneCodeHash(data.phone_code_hash);
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
      const data = await authApi.verifyCode(phoneNumber, phoneCode, phoneCodeHash);
      if (data.requires_2fa) {
        setStep('password');
      } else {
        onLoginSuccess(data.token);
      }
    } catch (err: any) {
      console.error(err);
      setError(err.message || 'Invalid code');
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
      const data = await authApi.verifyPassword(password);
      onLoginSuccess(data.token);
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
        <h2>Login with Telegram</h2>
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
