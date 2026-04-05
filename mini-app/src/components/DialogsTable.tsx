import React, { useEffect, useState } from 'react';
import { MessageSquare, Users, User, ArrowLeft, Zap } from 'lucide-react';
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

  const getTypeLabel = (type: string) => {
    switch (type) {
      case 'user': return '1-1 Chat';
      case 'bot': return 'Bot Chat';
      case 'group': return 'Group';
      case 'supergroup/channel': return 'Channel';
      default: return type;
    }
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
        <>
          <div className="table-container">
            <table className="beautiful-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Title</th>
                  <th style={{ width: '120px' }}>Category</th>
                  <th>Status</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {dialogs.map((d) => (
                  <tr key={d.id}>
                    <td>
                      <div className="avatar-icon">
                        {(d.type === 'user' || d.type === 'bot') && <User size={18} />}
                        {d.type === 'group' && <Users size={18} />}
                        {d.type === 'supergroup/channel' && <MessageSquare size={18} />}
                      </div>
                    </td>
                    <td className="font-medium text-light">{d.title}</td>
                    <td className="text-muted text-sm">{getTypeLabel(d.type)}</td>
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

          <div className="search-container">
            <h3>AI Search Across Enriched Messages</h3>
            <SearchSection />
          </div>
        </>
      )}
    </div>
  );
};

const SearchSection: React.FC = () => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<any[]>([]);
  const [analysis, setAnalysis] = useState('');
  const [searching, setSearching] = useState(false);

  const handleSearch = async () => {
    if (!query.trim()) return;
    setSearching(true);
    setAnalysis('');
    try {
      const res = await apiFetch('/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setResults(data.results || []);
      setAnalysis(data.analysis || '');
    } catch (err: any) {
      alert('Search failed: ' + err.message);
    } finally {
      setSearching(false);
    }
  };

  const getTgLink = (m: any) => {
    // If we have a username, use it. Otherwise, use channel_id format for private chats
    if (m.channel_username) {
      return `https://t.me/${m.channel_username}/${m.id}`;
    }
    // Private chats use 'c/' prefix and stripped ID (removes -100 prefix)
    const strippedId = String(m.channel_id).replace(/^-100/, '');
    return `https://t.me/c/${strippedId}/${m.id}`;
  };

  return (
    <div>
      <div className="search-input-group">
        <input 
          type="text" 
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Ask anything about your messages..."
          className="search-input"
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
        />
        <button 
          onClick={handleSearch} 
          disabled={searching}
          className="search-button"
        >
          {searching ? <div className="spinner-sm"></div> : <Zap size={16} />} Search
        </button>
      </div>

      {analysis && (
        <div className="analysis-card">
          <div className="analysis-header">
            <Zap size={18} /> AI Analysis
          </div>
          <div className="analysis-text">{analysis}</div>
        </div>
      )}

      <div className="search-results">
        {results.map((r, idx) => (
          <div key={idx} className="result-card">
            <div className="result-header">
              <div className="result-meta">
                <span>{new Date(r.date).toLocaleString()}</span>
                <span>ID: {r.id}</span>
              </div>
              <a 
                href={getTgLink(r)} 
                target="_blank" 
                rel="noopener noreferrer" 
                className="tg-link"
              >
                View in Telegram
              </a>
            </div>
            <div className="result-content">{r.context_text}</div>
          </div>
        ))}
      </div>
    </div>
  );
};
