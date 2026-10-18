// The shapes of the JSON the Go server sends and receives.
// Money is always in kopecks (integer), percentages in basis points
// (10000 = 100%), exactly as in the database. lib/format.ts turns them
// into rubles and % only when they are shown on the screen.

export type Role = 'developer' | 'designer' | 'manager' | 'other';
export type Category = 'web' | 'mobile' | 'design' | 'marketing' | 'other';
export type ProjectStatus = 'open' | 'in_progress' | 'completed' | 'failed' | 'cancelled';
export type SwapStatus = 'proposed' | 'active' | 'completed' | 'failed' | 'cancelled' | 'rejected';
export type Outcome = 'success' | 'failure';

// ----- users and login -----

export interface User {
  id: string;
  username: string;
  email: string;
  role: Role;
  reputation_bp: number;
  created_at: string;
}

/** What anyone may see about another user (no email). */
export interface PublicUser {
  id: string;
  username: string;
  role: Role;
  reputation_bp: number;
  created_at: string;
}

export interface AuthResponse {
  access_token: string;
  token_type: 'Bearer';
  expires_in: number; // seconds
  user: User;
}

export interface RegisterInput {
  username: string;
  email: string;
  password: string;
  role: Role;
}

export interface LoginInput {
  email: string;
  password: string;
}

// ----- projects -----

export interface KPI {
  name: string;
  target: string;
  weight: number; // all weights of a project add up to 100
}

export interface Project {
  id: string;
  creator: { id: string; username: string };
  title: string;
  description: string;
  category: Category;
  budget: number; // kopecks
  duration_days: number;
  complexity: number; // 1..10
  team_size: number;
  kpis: KPI[];
  success_probability_bp: number;
  status: ProjectStatus;
  created_at: string;
  updated_at: string;
}

export interface ProjectInput {
  title: string;
  description: string;
  category: Category;
  budget: number; // kopecks
  duration_days: number;
  complexity: number;
  team_size: number;
  kpis: KPI[];
}

export interface ScoreFactor {
  name: string;
  points: number;
}

export interface Score {
  score_bp: number;
  success_probability_bp: number;
  factors: ScoreFactor[];
}

export interface Page<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

// ----- swaps -----

export interface RiskProfile {
  max_gain: number; // kopecks
  max_loss: number; // kopecks
  expected_value: number; // kopecks
}

export interface SwapTerms {
  share_a_bp: number;
  share_b_bp: number;
  escrow_amount: number; // kopecks
  penalty_a: number; // kopecks
  penalty_b: number; // kopecks
  success_probability_bp: number;
  risk_a: RiskProfile;
  risk_b: RiskProfile;
}

export interface SwapParty {
  id: string;
  username: string;
  role: Role;
}

/** POST /api/swaps/calculate: the terms, nothing saved yet. */
export interface SwapCalculation {
  project: { id: string; title: string };
  party_a: SwapParty;
  party_b: SwapParty;
  terms: SwapTerms;
}

export interface Swap extends SwapCalculation {
  id: string;
  status: SwapStatus;
  outcome_a: Outcome | null;
  outcome_b: Outcome | null;
  cancel_requested_by: string | null;
  accepted_at: string | null;
  deadline_at: string | null;
  closed_at: string | null;
  created_at: string;
}

// ----- reputation -----

export interface Reputation {
  score_bp: number;
  completed_count: number;
  failed_count: number;
  cancelled_count: number;
  success_rate_bp: number;
  avg_rating_x100: number | null;
  avg_budget: number; // kopecks
  total_volume: number; // kopecks
  trust_weight_bp: number;
  reason: string;
  updated_at: string;
}

export interface ReputationProfile {
  user: PublicUser;
  reputation: Reputation;
  history: { score_bp: number; reason: string; created_at: string }[];
  reviews: {
    id: string;
    swap_id: string;
    author: { id: string; username: string };
    rating: number;
    comment: string;
    created_at: string;
  }[];
}

/** Every error from the server has this shape. */
export interface ApiErrorBody {
  error: { code: number; message: string; field?: string };
}