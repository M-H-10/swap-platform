import { useEffect, useState, type ReactNode } from 'react';
import * as authApi from '../api';
import { refreshSession, setAccessToken, setSessionExpiredHandler } from '../api/client';
import type { AuthResponse, LoginInput, RegisterInput, User } from '../types';
import { AuthContext } from './auth-context';

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    // If the refresh fails later (cookie expired, user logged out in
    // another tab), the client calls this and the user is logged out here.
    setSessionExpiredHandler(() => setUser(null));

    // On every page load: the access token was only in memory, so it is
    // gone. If the browser still has the refresh cookie, this gives us a
    // new access token and the user, and the user stays logged in.
    refreshSession()
      .then((data) => setUser(data.user))
      .catch(() => setUser(null))
      .finally(() => setReady(true));
  }, []);

  const startSession = (data: AuthResponse): User => {
    setAccessToken(data.access_token);
    setUser(data.user);
    return data.user;
  };

  const login = async (input: LoginInput) => startSession((await authApi.login(input)).data);

  const register = async (input: RegisterInput) => startSession((await authApi.register(input)).data);

  const logout = async () => {
    try {
      await authApi.logout(); // deletes the refresh token on the server and the cookie
    } finally {
      setAccessToken(null);
      setUser(null);
    }
  };

  return (
    <AuthContext.Provider value={{ user, ready, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}