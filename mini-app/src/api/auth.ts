const BASE_URL = '/api/api';

const getSessionID = () => {
  let id = localStorage.getItem('tg_session_id');
  if (!id) {
    id = Array.from(crypto.getRandomValues(new Uint8Array(16)))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
    localStorage.setItem('tg_session_id', id);
  }
  return id;
};

export const apiFetch = async (path: string, options: RequestInit = {}) => {
  const token = localStorage.getItem('tg_jwt_token');
  const sessionID = getSessionID();

  const headers = {
    'Content-Type': 'application/json',
    'X-Session-ID': sessionID,
    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
    ...(options.headers || {}),
  };

  const response = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    localStorage.removeItem('tg_jwt_token');
  }

  return response;
};

export const authApi = {
  sendCode: async (phone: string) => {
    const res = await apiFetch('/auth/send-code', {
      method: 'POST',
      body: JSON.stringify({ phone }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json(); // { phone_code_hash }
  },

  verifyCode: async (phone: string, code: string, phoneCodeHash: string) => {
    const res = await apiFetch('/auth/verify-code', {
      method: 'POST',
      body: JSON.stringify({ phone, code, phone_code_hash: phoneCodeHash }),
    });

    if (res.status === 403) {
      const data = await res.json();
      if (data.error === '2fa_required') {
        return { requires_2fa: true };
      }
    }

    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    localStorage.setItem('tg_jwt_token', data.token);
    return data;
  },

  verifyPassword: async (password: string) => {
    const res = await apiFetch('/auth/verify-password', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    localStorage.setItem('tg_jwt_token', data.token);
    return data;
  },

  getMe: async () => {
    const res = await apiFetch('/auth/me');
    if (!res.ok) throw new Error('Not authenticated');
    return res.json();
  },

  logout: async () => {
    await apiFetch('/auth/logout', { method: 'POST' });
    localStorage.removeItem('tg_jwt_token');
  }
};
