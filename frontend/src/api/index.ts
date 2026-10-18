import { api } from './client';
import type {
  AuthResponse, LoginInput, Page, PredictionComparison, Project, ProjectInput, PublicUser,
  RegisterInput, ReputationProfile, Review, ReviewInput, Score, Swap, SwapCalculation,
  Outcome, Transaction, User, Wallet,
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
/** The formula's score with all its factors, nothing saved. */
export const estimateProject = (data: ProjectInput) => api.post<Score>('/projects/estimate', data);

// ----- ML prediction (stage 7) -----
export const getPrediction = (projectId: string) =>
  api.get<PredictionComparison>(`/projects/${projectId}/prediction`);
export const requestPrediction = (projectId: string) =>
  api.post<{ status: 'pending' }>(`/projects/${projectId}/prediction`);

// ----- wallet -----
export const getWallet = () => api.get<Wallet>('/wallet');
export const getTransactions = (limit: number, offset: number) =>
  api.get<{ items: Transaction[]; total: number }>('/wallet/transactions', { params: { limit, offset } });
/** Free test money. The server allows it only in development. */
export const deposit = (amount: number) => api.post<Wallet>('/wallet/deposit', { amount });

// ----- swaps -----
export interface SwapProposal {
  project_id: string;
  partner_id: string;
}
export const calculateSwap = (data: SwapProposal) => api.post<SwapCalculation>('/swaps/calculate', data);
export const createSwap = (data: SwapProposal) => api.post<Swap>('/swaps', data);
export const getSwaps = () => api.get<{ items: Swap[] }>('/swaps');
export const getSwap = (id: string) => api.get<Swap>(`/swaps/${id}`);
export const acceptSwap = (id: string) => api.post<Swap>(`/swaps/${id}/accept`);
export const rejectSwap = (id: string) => api.post<Swap>(`/swaps/${id}/reject`);
export const cancelSwap = (id: string) => api.post<Swap>(`/swaps/${id}/cancel`);
export const reportOutcome = (id: string, outcome: Outcome) =>
  api.post<Swap>(`/swaps/${id}/outcome`, { outcome });

// ----- reviews (stage 6) -----
export const addReview = (swapId: string, data: ReviewInput) =>
  api.post<Review>(`/swaps/${swapId}/review`, data);