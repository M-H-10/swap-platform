import type { CSSProperties } from 'react';
import { useTheme } from '../context/theme-context';
import { cssVar } from './format';

/** Colours for the recharts charts, read from the CSS variables.
 *  Recharts needs real colour values, not var(--...), so we read them here.
 *  Because this hook uses useTheme(), every chart draws again with the new
 *  colours right after the light / dark switch. */
export function useChartTheme() {
  const { theme } = useTheme();
  const text2 = cssVar('--text2');

  const tooltip: CSSProperties = {
    background: cssVar('--surface'),
    border: `1px solid ${cssVar('--border')}`,
    borderRadius: 8,
    color: cssVar('--text'),
    boxShadow: cssVar('--shadow-pop'),
    fontSize: 13,
  };

  return {
    theme,
    grid: cssVar('--border'),
    tick: { fill: text2, fontSize: 12 },
    tooltip,
    tooltipCursor: { fill: cssVar('--surface-2') },
    partyA: cssVar('--party-a'),
    partyB: cssVar('--party-b'),
    accent: cssVar('--accent'),
    escrow: cssVar('--escrow'),
    success: cssVar('--success'),
    danger: cssVar('--danger'),
    warning: cssVar('--warning'),
    muted: cssVar('--text3'),
  };
}