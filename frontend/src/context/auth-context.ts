import { createContext, useContext } from 'react';
import type { LoginInput, RegisterInput, User } from '../types';

export interface AuthContextValue {
  /** The logged-in user, or null. */
  user: User | null;
  /** false while we are still asking the server "is there a session?" */
  ready: boolean;
  login: (data: LoginInput) => Promise<User>;
  register: (data: RegisterInput) => Promise<User>;
  logout: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);

/** Use the login state in any component inside <AuthProvider>. */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used inside <AuthProvider>');
  }
  return ctx;
}