import React, { useEffect, useState } from 'react';
import { MessageSquare, Users, ArrowLeft } from 'lucide-react';
import { apiFetch } from '../api/auth';

interface DialogsTableProps {
  onLogout: () => void;
}

interface DialogInfo {
  id: number;
  title: string;
  type: string;
}

export const DialogsTable: React.FC<DialogsTableProps> = ({ onLogout }) => {
  const [dialogs, setDialogs] = useState<DialogInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchDialogs = async () => {
      try {
        const res = await apiFetch('/dialogs');
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json();
        setDialogs(data);
      } catch (err: any) {
        console.error('Failed to fetch dialogs:', err);
        setError('Failed to fetch dialogs. ' + err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchDialogs();
  }, []);

  return (
    <div className="card glass-card large-card fade-in">
      <div className="flex-between header-row">
        <h2>Your Conversations</h2>
        <button onClick={onLogout} className="text-button flex-center gap-sm text-sm">
          <ArrowLeft size={16} /> Logout
        </button>
      </div>

      {error ? (
        <div className="error-message">{error}</div>
      ) : loading ? (
        <div className="loading-state">
          <div className="spinner"></div>
          <p>Loading dialogs...</p>
        </div>
      ) : (
        <div className="table-container">
          <table className="beautiful-table">
            <thead>
              <tr>
                <th>Type</th>
                <th>ID</th>
                <th>Title</th>
              </tr>
            </thead>
            <tbody>
              {dialogs.map((d) => (
                <tr key={d.id}>
                  <td>
                    <div className="avatar-icon">
                      {d.type === 'group' && <Users size={18} />}
                      {d.type === 'supergroup/channel' && <MessageSquare size={18} />}
                    </div>
                  </td>
                  <td className="text-muted">{d.id}</td>
                  <td className="font-medium text-light">{d.title}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
