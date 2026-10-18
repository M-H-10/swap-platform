import { api } from './client';
import type {
  AuthResponse, LoginInput, Page, Project, ProjectInput, PublicUser, RegisterInput,
  ReputationProfile, Score, Swap, SwapCalculation, User,
} from '../types';

// One small typed function per server route. The pages call these and
// never write URLs themselves.

// ----- auth -----
export const register = (data: RegisterInput) => api.post<AuthResponse>('/auth/register', data);
export const login = (data: LoginInput) => api.post<AuthResponse>('/auth/login', data);
export const logout = () => api.post<void>('/auth/logout');
export const getMe = () => api.get<User>('/auth/me');

// ----- users -----
export const searchUsers = (q: string) =>
  api.get<{ items: PublicUser[] }>('/users', { params: { q, limit: 8 } });
export const getReputation = (userId: string) =>
  api.get<ReputationProfile>(`/users/${userId}/reputation`);

// ----- projects -----
export const getProjects = (params?: { status?: string; category?: string; q?: string }) =>
  api.get<Page<Project>>('/projects', { params });
export const getProject = (id: string) => api.get<Project>(`/projects/${id}`);
export const createProject = (data: ProjectInput) =>
  api.post<{ project: Project; score: Score }>('/projects', data);

// ----- swaps -----
export interface SwapProposal {
  project_id: string;
  partner_id: string;
}
export const calculateSwap = (data: SwapProposal) => api.post<SwapCalculation>('/swaps/calculate', data);
export const createSwap = (data: SwapProposal) => api.post<Swap>('/swaps', data);
export const getSwaps = () => api.get<{ items: Swap[] }>('/swaps');
export const acceptSwap = (id: string) => api.post<Swap>(`/swaps/${id}/accept`);
export const rejectSwap = (id: string) => api.post<Swap>(`/swaps/${id}/reject`);
export const cancelSwap = (id: string) => api.post<Swap>(`/swaps/${id}/cancel`);