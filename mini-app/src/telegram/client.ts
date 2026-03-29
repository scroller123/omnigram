import { TelegramClient } from 'telegram';
import { StringSession } from 'telegram/sessions';

// Using a custom LocalStorage session to persist the GramJS session string
export class StorageSession extends StringSession {
  constructor() {
    // try to load existing string from localStorage
    const saved = localStorage.getItem('tg_session_string') || '';
    super(saved);
  }

  override save(): string {
    const sessionString = super.save();
    localStorage.setItem('tg_session_string', sessionString);
    return sessionString;
  }

  clear() {
    localStorage.removeItem('tg_session_string');
    super.setDC(0, '', 0);
    super.setAuthKey(undefined);
  }
}

let activeClient: TelegramClient | null = null;

export function getClient(apiId: number, apiHash: string): TelegramClient {
  if (!activeClient) {
    const session = new StorageSession();
    activeClient = new TelegramClient(session, apiId, apiHash, {
      connectionRetries: 5,
    });
  }
  return activeClient;
}

export function resetClient() {
  activeClient = null;
}
