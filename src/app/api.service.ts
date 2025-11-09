import { Injectable } from '@angular/core';
import { CoreAuthService, CoreAuthSession, EnsureAuthOptions } from '@berjis/angular-auth';

@Injectable({ providedIn: 'root' })
export class ApiService {
  constructor(private auth: CoreAuthService) {}

  ensureAuth(options: EnsureAuthOptions = {}): Promise<CoreAuthSession> {
    return this.auth.ensureAuth({ maxAgeMs: 1500, ...options });
  }

  onSessionChange(handler: (session: CoreAuthSession) => void): () => void {
    return this.auth.onSessionChange(handler);
  }

  currentSession(): CoreAuthSession {
    return this.auth.getSession();
  }
}

