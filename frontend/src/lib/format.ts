import { isAxiosError } from 'axios';
import type { ApiErrorBody } from '../types';

// The server keeps money in kopecks and percentages in basis points, so
// there is never a rounding error in the database. These helpers convert
// only at the edge: when a value is shown, or when the user types one.

/** 1250000 kopecks -> "₽12,500" (kopecks are shown only when there are any). */
export function formatRub(kopecks: number): string {
  return '₽' + (kopecks / 100).toLocaleString('en-US', { maximumFractionDigits: 2 });
}

/** What the user typed in rubles ("12500.5") -> kopecks (1250050). */
export function rublesToKopecks(rubles: string | number): number {
  return Math.round(Number(rubles) * 100);
}

/** Basis points -> percent number: 3740 -> 37.4 */
export function bpToPercent(bp: number): number {
  return bp / 100;
}

/** Basis points -> text: 3740 -> "37.4%" */
export function formatPercent(bp: number): string {
  return bpToPercent(bp).toLocaleString('en-US', { maximumFractionDigits: 1 }) + '%';
}

/** Colour for a success probability (basis points). */
export function probabilityColor(bp: number): string {
  if (bp >= 7000) return '#00d4aa';
  if (bp >= 4000) return '#f59e0b';
  return '#e94560';
}

/** The message to show for any failed request. */
export function errorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (isAxiosError<ApiErrorBody>(err)) {
    const body = err.response?.data;
    if (body?.error?.message) return body.error.message;
    if (!err.response) return 'The server is not reachable. Is it running?';
  }
  return fallback;
}