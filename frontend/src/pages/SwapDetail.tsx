import { useEffect, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import toast from 'react-hot-toast';
import {
  acceptSwap, addReview, cancelSwap, getPrediction, getReputation, getSwap, rejectSwap, reportOutcome,
} from '../api';
import TermSheet from '../components/TermSheet';
import { useAuth } from '../context/auth-context';
import { bpToPercent, daysUntil, errorMessage, formatDate, formatPercent, formatRub } from '../lib/format';
import type { Outcome, PredictionComparison, Swap, SwapParty, SwapStatus } from '../types';

const outcomeTag = (o: Outcome | null) =>
  o === null ? <span className="tag tag-muted">Not reported yet</span>
    : o === 'success' ? <span className="tag tag-success">Reported success</span>
      : <span className="tag tag-danger">Reported failure</span>;

// What the middle cell of the term sheet says, by swap status.
const escrowLabel: Record<SwapStatus, string> = {
  proposed: 'Locked when accepted', active: 'Held in escrow', completed: 'Released',
  failed: 'Paid as penalties', cancelled: 'Returned', rejected: 'Not used',
};

const PartyLink = ({ party }: { party: SwapParty }) => (
  <Link className="link" to={`/users/${party.id}`}>{party.username}</Link>
);

const SwapDetail = () => {
  const { id = '' } = useParams();
  const { user } = useAuth();
  const [swap, setSwap] = useState<Swap | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [reviewed, setReviewed] = useState(false);
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState('');
  const [prediction, setPrediction] = useState<PredictionComparison | null>(null);

  useEffect(() => {
    getSwap(id)
      .then(({ data }) => {
        setSwap(data);
        // The ML answer for the project, to compare with the formula.
        getPrediction(data.project.id).then((r) => setPrediction(r.data)).catch(() => {});
      })
      .catch((err) => toast.error(errorMessage(err, 'Swap not found')))
      .finally(() => setLoading(false));
  }, [id]);

  const iAmA = !!swap && swap.party_a.id === user?.id;
  const me = swap ? (iAmA ? swap.party_a : swap.party_b) : null;
  const other = swap ? (iAmA ? swap.party_b : swap.party_a) : null;
  const finished = swap?.status === 'completed' || swap?.status === 'failed';

  // Did I already review this swap? My review is in the OTHER party's
  // public profile, so we look for it there.
  useEffect(() => {
    if (!finished || !other || !user) return;
    getReputation(other.id)
      .then(({ data }) => setReviewed(
        data.reviews.some((r) => r.swap_id === id && r.author.id === user.id)))
      .catch(() => {});
  }, [finished, other, user, id]);

  // Runs one action on the swap and shows the updated swap.
  const run = async (action: () => Promise<{ data: Swap }>, done: string) => {
    setBusy(true);
    try {
      const { data } = await action();
      setSwap(data);
      toast.success(done);
    } catch (err) {
      toast.error(errorMessage(err));
    }
    setBusy(false);
  };

  const report = (outcome: Outcome) => {
    const question = outcome === 'success'
      ? 'Report SUCCESS? When both parties report success, each guarantee is returned (minus the 0.5% fee).'
      : 'Report FAILURE? When both parties report failure, each guarantee goes to the OTHER party as a penalty.';
    if (!window.confirm(question)) return;
    run(() => reportOutcome(id, outcome), 'Your answer is saved.');
  };

  const cancel = () => {
    if (!swap) return;
    const question = swap.status === 'active'
      ? 'An active swap is cancelled only when BOTH parties agree. Ask to cancel?'
      : 'Cancel this swap proposal?';
    if (!window.confirm(question)) return;
    run(() => cancelSwap(id), swap.status === 'active' ? 'Cancel request sent.' : 'Swap cancelled.');
  };

  const sendReview = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      await addReview(id, { rating, comment });
      setReviewed(true);
      toast.success('Thank you! Your review is saved.');
    } catch (err) {
      toast.error(errorMessage(err, 'Could not save the review'));
    }
    setBusy(false);
  };

  if (loading) return <div className="loading">Loading...</div>;
  if (!swap || !me || !other) return <div className="empty">Swap not found.</div>;

  const t = swap.terms;
  const myOutcome = iAmA ? swap.outcome_a : swap.outcome_b;
  const disagree = swap.outcome_a !== null && swap.outcome_b !== null && swap.outcome_a !== swap.outcome_b;
  const iAskedCancel = swap.cancel_requested_by === user?.id;
  const otherAskedCancel = swap.cancel_requested_by !== null && !iAskedCancel;
  const deadlinePassed = swap.deadline_at !== null && new Date(swap.deadline_at) < new Date();

  const daysLeft = daysUntil(swap.deadline_at);
  const mlBP = prediction?.ml?.success_probability_bp ?? null;

  return (
    <div className="page">
      <Link className="back-link" to="/dashboard">← My swaps</Link>
      <div className="page-title">
        <h1>{swap.project.title}</h1>
        <span className={`status-badge status-${swap.status}`}>{swap.status}</span>
        {swap.status === 'active' && daysLeft !== null && daysLeft >= 0 && (
          <span className="muted">{daysLeft} {daysLeft === 1 ? 'day' : 'days'} to the deadline</span>
        )}
        <p className="muted">
          Swap between <PartyLink party={swap.party_a} /> and <PartyLink party={swap.party_b} /> ·{' '}
          <Link className="link" to={`/projects/${swap.project.id}`}>open project</Link>
        </p>
      </div>

      <div className="swap-layout">
        <div className="swap-main">
          <motion.section className="panel" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, ease: [0.23, 1, 0.32, 1] }}>
            <h3>Terms of the swap</h3>
            <TermSheet partyA={swap.party_a.username} partyB={swap.party_b.username}
              terms={t} escrowLabel={escrowLabel[swap.status]} />

            {swap.status === 'proposed' && (
              <div className="panel-section">
                <h3>Proposal</h3>
                {iAmA ? (
                  <p className="muted">Waiting for {other.username} to accept or reject.</p>
                ) : (
                  <p className="muted">
                    Accepting locks your guarantee of {formatRub(t.penalty_b)} until the swap ends.
                  </p>
                )}
                <div className="actions">
                  {!iAmA && (
                    <>
                      <button className="btn-small primary" disabled={busy}
                        onClick={() => run(() => acceptSwap(id), 'Swap accepted! Escrow locked.')}>Accept terms</button>
                      <button className="btn-small danger" disabled={busy}
                        onClick={() => run(() => rejectSwap(id), 'Swap rejected.')}>Reject</button>
                    </>
                  )}
                  {iAmA && <button className="btn-small danger" disabled={busy} onClick={cancel}>Cancel proposal</button>}
                </div>
              </div>
            )}

            {swap.status === 'active' && (
              <div className="panel-section">
                <h3>Result</h3>
                <p className="muted">
                  Each party reports the result. The money moves only when both answers are the same.
                  You can change your answer until then.
                </p>
                <div className="mt">
                  <div className="outcome-row">
                    <span>{swap.party_a.username}</span>{outcomeTag(swap.outcome_a)}
                  </div>
                  <div className="outcome-row">
                    <span>{swap.party_b.username}</span>{outcomeTag(swap.outcome_b)}
                  </div>
                </div>
                {disagree && (
                  <p className="note note-warning">
                    The answers are different, so nothing happens yet. Talk to the other party and change an answer.
                  </p>
                )}
                <div className="actions">
                  <button className="btn-small primary" disabled={busy || myOutcome === 'success'}
                    onClick={() => report('success')}>It succeeded</button>
                  <button className="btn-small danger" disabled={busy || myOutcome === 'failure'}
                    onClick={() => report('failure')}>It failed</button>
                  {!iAskedCancel && (
                    <button className="btn-small" disabled={busy} onClick={cancel}>
                      {otherAskedCancel ? 'Agree to cancel' : 'Ask to cancel'}
                    </button>
                  )}
                </div>
                {iAskedCancel && <p className="note">You asked to cancel. Waiting for {other.username}.</p>}
                {otherAskedCancel && <p className="note">{other.username} asked to cancel this swap.</p>}
              </div>
            )}

            {(swap.status === 'cancelled' || swap.status === 'rejected') && (
              <p className="note">
                This swap was {swap.status}. {swap.status === 'cancelled' && 'All guarantees were returned in full.'}
              </p>
            )}
          </motion.section>

          {finished && (
            <section className="panel">
              <h3>Review {other.username}</h3>
              {swap.status === 'completed'
                ? <p className="note note-success">Both parties reported success. The guarantees were returned.</p>
                : <p className="note note-warning">Both parties reported failure. The guarantees were paid as penalties.</p>}
              {reviewed ? (
                <p className="muted mt">
                  You already reviewed this swap. See it on <Link className="link" to={`/users/${other.id}`}>
                  {other.username}'s profile</Link>.
                </p>
              ) : (
                <form onSubmit={sendReview} className="mt">
                  <div className="form-group">
                    <label>Rating</label>
                    <div className="star-picker">
                      {[1, 2, 3, 4, 5].map((n) => (
                        <button key={n} type="button" className={n <= rating ? 'on' : ''}
                          aria-label={`${n} stars`} onClick={() => setRating(n)}>★</button>
                      ))}
                    </div>
                  </div>
                  <div className="form-group">
                    <label>Comment (optional, up to 1000 characters)</label>
                    <textarea maxLength={1000} value={comment} onChange={(e) => setComment(e.target.value)}
                      placeholder={`How was it to work with ${other.username}?`} />
                  </div>
                  <button type="submit" className="btn-primary" disabled={busy}>Send review</button>
                </form>
              )}
            </section>
          )}
        </div>

        <div className="swap-side">
          <motion.section className="panel" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.05, ease: [0.23, 1, 0.32, 1] }}>
            <h3>Success probability</h3>
            <div className="prob-head">
              <div>
                <div className="big-number">{formatPercent(t.success_probability_bp)}</div>
                <div className="big-label">used for the terms</div>
              </div>
            </div>
            <div className="compare-row">
              <div className="compare-label"><span>Formula</span><strong>{formatPercent(t.success_probability_bp)}</strong></div>
              <div className="bar-track">
                <div className="bar-fill" style={{ width: `${bpToPercent(t.success_probability_bp)}%`, background: 'var(--text)' }} />
              </div>
            </div>
            {mlBP !== null ? (
              <div className="compare-row">
                <div className="compare-label"><span>ML model</span><strong>{formatPercent(mlBP)}</strong></div>
                <div className="bar-track">
                  <div className="bar-fill" style={{ width: `${bpToPercent(mlBP)}%`, background: 'var(--accent)' }} />
                </div>
              </div>
            ) : (
              <p className="muted">The ML model has no answer for this project yet.</p>
            )}
          </motion.section>

          <motion.section className="panel" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.1, ease: [0.23, 1, 0.32, 1] }}>
            <h3>Timeline</h3>
            <dl className="kv">
              <dt>Proposed</dt><dd>{formatDate(swap.created_at)}</dd>
              <dt>Accepted</dt><dd>{formatDate(swap.accepted_at)}</dd>
              <dt>Deadline</dt><dd>{formatDate(swap.deadline_at)}</dd>
              <dt>Closed</dt><dd>{formatDate(swap.closed_at)}</dd>
            </dl>
            {swap.status === 'active' && deadlinePassed && (
              <p className="note note-warning">The deadline has passed. Please report the result.</p>
            )}
          </motion.section>
        </div>
      </div>
    </div>
  );
};

export default SwapDetail;