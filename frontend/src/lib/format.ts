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

/** Colour for a success probability (basis points), as a CSS variable,
 *  so it follows the light / dark theme. */
export function probabilityColor(bp: number): string {
  if (bp >= 7000) return 'var(--success)';
  if (bp >= 4000) return 'var(--warning)';
  return 'var(--danger)';
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


/** ISO date from the server -> "18 Oct 2026, 20:41" in the user's time zone. */
export function formatDate(iso: string | null | undefined): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString('en-GB', {
    day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
  });
}

/** Whole days from now until an ISO date (rounded up), or null without a date. */
export function daysUntil(iso: string | null | undefined): number | null {
  if (!iso) return null;
  return Math.ceil((new Date(iso).getTime() - Date.now()) / 86_400_000);
}

/** The value of a CSS variable from :root, e.g. cssVar('--accent') -> "#6c63ff".
 *  Charts need real colour values, so they read them from the same
 *  variables as the CSS. Change a colour once in CSS and the charts follow. */
export function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}