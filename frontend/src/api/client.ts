import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { AuthResponse } from '../types';

// How the login works with the Go server (stage 3):
//   * the ACCESS token (15 minutes) comes in the JSON body. We keep it only
//     in memory (this variable), never in localStorage, so a malicious
//     script cannot steal it from the disk;
//   * the REFRESH token (7 days) is an HttpOnly cookie. JavaScript cannot
//     read it at all; the browser sends it by itself to /api/auth/*.
// When the access token expires the server answers 401. Then we ask
// /api/auth/refresh for a new one and repeat the request, so the user
// never notices.

let accessToken: string | null = null;

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

// AuthContext registers a function here, so the client can tell the app
// "the session is over" (refresh failed) without importing React.
let onSessionExpired: () => void = () => {};

export function setSessionExpiredHandler(fn: () => void): void {
  onSessionExpired = fn;
}

export const api = axios.create({
  // "/api" goes to the Vite proxy, which forwards it to the Go server
  // (see vite.config.ts). VITE_API_URL can override it later.
  baseURL: import.meta.env.VITE_API_URL ?? '/api',
  withCredentials: true, // send the refresh cookie
});

api.interceptors.request.use((config) => {
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

// One refresh at a time. Every refresh token works only ONCE (stage 3:
// the server rotates it, and treats a second use as theft). So:
//   * inside one tab: if 5 requests fail together, they all wait for the
//     same refresh instead of starting 5 of them;
//   * between tabs: navigator.locks makes a second tab wait until the
//     first tab's refresh is finished, so it sends the NEW cookie.
let refreshing: Promise<AuthResponse> | null = null;

const doRefresh = () =>
  api.post<AuthResponse>('/auth/refresh').then(({ data }) => {
    setAccessToken(data.access_token);
    return data;
  });

export function refreshSession(): Promise<AuthResponse> {
  if (!refreshing) {
    const run = navigator.locks
      ? navigator.locks.request('swap-platform-refresh', doRefresh)
      : doRefresh();
    refreshing = run.finally(() => {
      refreshing = null;
    });
  }
  return refreshing;
}

type RetryConfig = InternalAxiosRequestConfig & { _retried?: boolean };

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryConfig | undefined;
    const isAuthCall = config?.url?.startsWith('/auth/');

    // Only a 401 on a normal request is worth a refresh, and only once.
    if (error.response?.status !== 401 || !config || isAuthCall || config._retried) {
      return Promise.reject(error);
    }
    config._retried = true;

    try {
      await refreshSession();
    } catch {
      setAccessToken(null);
      onSessionExpired();
      return Promise.reject(error);
    }
    return api(config); // the same request again, now with the new token
  },
);