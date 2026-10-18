import { motion } from 'framer-motion';
import { bpToPercent, formatPercent, formatRub } from '../lib/format';
import type { SwapTerms } from '../types';

interface Props {
  partyA: string;
  partyB: string;
  terms: SwapTerms;
  /** Where the money is now, e.g. "Held in escrow" or "Returned". */
  escrowLabel?: string;
}

// The signature of the design: one bar that shows how the result is split,
// and a "vault" with the two guarantees and the escrow between them.
// The same picture is used in the calculator, the swap page and on Home,
// so a user learns to read it once.
const TermSheet = ({ partyA, partyB, terms, escrowLabel = 'Held in escrow' }: Props) => {
  const a = bpToPercent(terms.share_a_bp);

  return (
    <div className="term-sheet">
      <div className="ts-parties">
        <div>
          <div className="ts-name">{partyA}</div>
          <div className="ts-role">Party A, project owner</div>
        </div>
        <div className="ts-right">
          <div className="ts-name">{partyB}</div>
          <div className="ts-role">Party B, partner</div>
        </div>
      </div>

      {/* The bar opens from left to right once, when the page appears. */}
      <motion.div
        className="ts-split"
        role="img"
        aria-label={`${partyA} gets ${formatPercent(terms.share_a_bp)}, ${partyB} gets ${formatPercent(terms.share_b_bp)}`}
        initial={{ clipPath: 'inset(0 100% 0 0)' }}
        animate={{ clipPath: 'inset(0 0% 0 0)' }}
        transition={{ duration: 0.6, ease: [0.23, 1, 0.32, 1] }}
      >
        <div className="ts-a" style={{ width: `${a}%` }}>{a >= 12 && formatPercent(terms.share_a_bp)}</div>
        <div className="ts-b" style={{ width: `${100 - a}%` }}>{a <= 88 && formatPercent(terms.share_b_bp)}</div>
      </motion.div>
      <div className="ts-caption">Share of the result if the project succeeds</div>

      <div className="ts-vault">
        <div>
          <div className="ts-k">Guarantee A</div>
          <div className="ts-v">{formatRub(terms.penalty_a)}</div>
        </div>
        <div className="ts-escrow">
          <div className="ts-k">{escrowLabel}</div>
          <div className="ts-v">{formatRub(terms.escrow_amount)}</div>
        </div>
        <div className="ts-right">
          <div className="ts-k">Guarantee B</div>
          <div className="ts-v">{formatRub(terms.penalty_b)}</div>
        </div>
      </div>
    </div>
  );
};

export default TermSheet;