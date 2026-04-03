import React, { useEffect, useState } from 'react';
import { MessageSquare, Users, ArrowLeft, Zap, CheckCircle, Clock, AlertCircle } from 'lucide-react';
import { apiFetch } from '../api/auth';

interface DialogsTableProps {
  onLogout: () => void;
}

interface DialogInfo {
  id: number;
  title: string;
  type: string;
}

interface EnrichmentTask {
  dialog_id: number;
  status: string;
  total_messages: number;
  processed_messages: number;
  error_message?: string;
}

export const DialogsTable: React.FC<DialogsTableProps> = ({ onLogout }) => {
  const [dialogs, setDialogs] = useState<DialogInfo[]>([]);
  const [tasks, setTasks] = useState<Record<number, EnrichmentTask>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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

  const fetchTasks = async () => {
    try {
      const res = await apiFetch('/enrich/tasks');
      if (!res.ok) return;
      const data: EnrichmentTask[] = await res.json();
      const taskMap: Record<number, EnrichmentTask> = {};
      data.forEach(t => taskMap[t.dialog_id] = t);
      setTasks(taskMap);
    } catch (err) {
      console.error('Failed to fetch tasks:', err);
    }
  };

  useEffect(() => {
    fetchDialogs();
    fetchTasks();
    const interval = setInterval(fetchTasks, 3000); // Poll every 3 seconds
    return () => clearInterval(interval);
  }, []);

  const handleEnrich = async (dialogId: number) => {
    try {
      const res = await apiFetch('/enrich', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dialog_id: dialogId })
      });
      if (!res.ok) throw new Error(await res.text());
      fetchTasks(); // Immediate refresh
    } catch (err: any) {
      alert('Failed to start enrichment: ' + err.message);
    }
  };

  const renderStatus = (dialogId: number) => {
    const task = tasks[dialogId];
    if (!task) return null;

    const progress = task.total_messages > 0 
      ? Math.round((task.processed_messages / task.total_messages) * 100) 
      : 0;

    return (
      <div className="flex-center-y" style={{ alignItems: 'flex-start', minWidth: '150px' }}>
        <div className={`status-badge status-${task.status.toLowerCase()}`}>
          {task.status}
        </div>
        {task.status === 'processing' && (
          <>
            <div className="progress-container">
              <div className="progress-bar" style={{ width: `${progress}%` }}></div>
            </div>
            <div className="progress-stats">
              <span className="stats-numbers">{task.processed_messages.toLocaleString()} / {task.total_messages.toLocaleString()}</span>
              <span className="stats-percent">{progress}%</span>
            </div>
          </>
        )}
        {task.status === 'completed' && (
          <div style={{ fontSize: '0.65rem', color: 'var(--text-muted)', marginTop: '2px' }}>
            {task.processed_messages} messages encrypted
          </div>
        )}
        {task.status === 'failed' && task.error_message && (
          <div style={{ fontSize: '0.65rem', color: 'var(--error)', marginTop: '2px' }}>
            {task.error_message}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="card glass-card large-card fade-in">
      <div className="flex-between header-row" style={{ padding: '0 1rem', marginBottom: '1rem' }}>
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
                <th>Title</th>
                <th>Status</th>
                <th>Action</th>
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
                  <td className="font-medium text-light">{d.title}</td>
                  <td>{renderStatus(d.id)}</td>
                  <td>
                    <button 
                      onClick={() => handleEnrich(d.id)}
                      disabled={tasks[d.id]?.status === 'processing'}
                      className="action-button"
                    >
                      <Zap size={14} /> Enrich
                    </button>
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
