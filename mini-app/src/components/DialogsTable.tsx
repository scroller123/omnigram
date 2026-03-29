import React, { useEffect, useState } from 'react';
import { TelegramClient } from 'telegram';
import { MessageSquare, Users, User, ArrowLeft } from 'lucide-react';

interface DialogsTableProps {
  client: TelegramClient;
  onLogout: () => void;
}

interface DialogInfo {
  id: string;
  name: string;
  lastMessage: string;
  date: number;
  unreadCount: number;
  type: 'user' | 'chat' | 'channel';
}

export const DialogsTable: React.FC<DialogsTableProps> = ({ client, onLogout }) => {
  const [dialogs, setDialogs] = useState<DialogInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchDialogs = async () => {
      try {
        const dialogsData = await client.getDialogs({ limit: 50 });
        const parsedDialogs = dialogsData.map((d: any) => ({
          id: d.id?.toString() || Math.random().toString(),
          name: d.title || d.name || 'Unknown',
          lastMessage: d.message?.message || d.message?.text || 'Media message',
          date: d.date || 0,
          unreadCount: d.unreadCount || 0,
          type: (d.isChannel ? 'channel' : d.isGroup ? 'chat' : 'user') as 'user' | 'chat' | 'channel',
        }));
        setDialogs(parsedDialogs);
      } catch (err: any) {
        console.error('Failed to fetch dialogs:', err);
        setError('Failed to fetch dialogs. ' + err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchDialogs();
  }, [client]);

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
                <th>Name</th>
                <th>Last Message</th>
                <th>Unread</th>
              </tr>
            </thead>
            <tbody>
              {dialogs.map((d) => (
                <tr key={d.id}>
                  <td>
                    <div className="avatar-icon">
                      {d.type === 'user' && <User size={18} />}
                      {d.type === 'chat' && <Users size={18} />}
                      {d.type === 'channel' && <MessageSquare size={18} />}
                    </div>
                  </td>
                  <td className="font-medium text-light">{d.name}</td>
                  <td className="text-muted truncate-cell">{d.lastMessage}</td>
                  <td>
                    {d.unreadCount > 0 ? (
                      <span className="badge">{d.unreadCount}</span>
                    ) : (
                      <span className="text-muted">-</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
