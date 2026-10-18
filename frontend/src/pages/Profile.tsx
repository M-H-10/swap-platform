import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { getReputation } from '../api';
import { cssVar, errorMessage, formatDate, formatPercent, formatRub } from '../lib/format';
import type { ReputationProfile } from '../types';

// Why the score changed (the "reason" saved with every snapshot).
const reasonLabel: Record<string, string> = {
  registration: 'Account created',
  swap_completed: 'Swap completed',
  swap_failed: 'Swap failed',
  swap_cancelled: 'Swap cancelled',
  review_received: 'New review',
};

const Stars = ({ value }: { value: number }) => (
  <span className="stars" aria-label={`${value} of 5 stars`}>
    {'★'.repeat(value)}<span style={{ opacity: 0.25 }}>{'★'.repeat(5 - value)}</span>
  </span>
);

const Profile = () => {
  const { id = '' } = useParams();
  const [profile, setProfile] = useState<ReputationProfile | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    getReputation(id)
      .then(({ data }) => setProfile(data))
      .catch((err) => setError(errorMessage(err, 'User not found')));
  }, [id]);

  if (error) return <div className="empty">{error}</div>;
  if (!profile) return <div className="loading">Loading...</div>;

  const { user, reputation: rep, history, reviews } = profile;
  const chartData = history.map((h) => ({
    date: new Date(h.created_at).toLocaleString('en-GB', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }),
    score: h.score_bp / 100,
    reason: reasonLabel[h.reason] ?? h.reason,
  }));

  const stats = [
    { label: 'Reputation', value: formatPercent(rep.score_bp) },
    { label: 'Success rate', value: formatPercent(rep.success_rate_bp) },
    { label: 'Average rating', value: rep.avg_rating_x100 === null ? '—' : (rep.avg_rating_x100 / 100).toFixed(1) + ' / 5' },
    { label: 'Trust weight', value: formatPercent(rep.trust_weight_bp) },
  ];

  return (
    <div className="page">
      <div className="page-title">
        <div className="avatar">{user.username[0]?.toUpperCase()}</div>
        <div>
          <h1>{user.username}</h1>
          <span className="role-badge">{user.role}</span>
        </div>
        <p className="muted">Member since {formatDate(user.created_at)}</p>
      </div>

      <div className="stats-row">
        {stats.map((s, i) => (
          <motion.div key={s.label} className="stat-card-dash"
            initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.06 }}>
            <div className="stat-val">{s.value}</div>
            <div className="stat-lbl">{s.label}</div>
          </motion.div>
        ))}
      </div>

      <div className="grid-2">
        <div className="panel">
          <h3>Track record</h3>
          <dl className="kv">
            <dt>Completed swaps</dt><dd>{rep.completed_count}</dd>
            <dt>Failed swaps</dt><dd>{rep.failed_count}</dd>
            <dt>Cancelled swaps</dt><dd>{rep.cancelled_count}</dd>
            <dt>Average project budget</dt><dd>{formatRub(rep.avg_budget)}</dd>
            <dt>Total volume</dt><dd>{formatRub(rep.total_volume)}</dd>
          </dl>
          <p className="muted" style={{ marginTop: '1rem' }}>
            Trust weight is how much this user's reviews count for other people. It grows with
            the age of the account (full after 90 days) and with completed swaps (full after 5),
            so fake new accounts cannot push a rating up.
          </p>
        </div>

        <div className="panel">
          <h3>Reputation over time (%)</h3>
          {chartData.length < 2 ? (
            <p className="muted">The chart appears after the first change of the score.</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={chartData}>
                <CartesianGrid stroke={cssVar('--border')} vertical={false} />
                <XAxis dataKey="date" tick={{ fill: cssVar('--text2'), fontSize: 11 }} />
                <YAxis domain={[0, 100]} tick={{ fill: cssVar('--text2'), fontSize: 11 }} width={32} />
                <Tooltip
                  contentStyle={{ background: cssVar('--bg2'), border: `1px solid ${cssVar('--accent')}` }}
                  formatter={(v, _name, item) => [`${v}%`, item.payload.reason]} />
                <Line type="monotone" dataKey="score" stroke={cssVar('--accent')} strokeWidth={2} dot={{ r: 3 }} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      <div className="panel">
        <h3>Reviews ({reviews.length})</h3>
        {reviews.length === 0 ? (
          <p className="muted">No reviews yet. Reviews can be written after a finished swap.</p>
        ) : reviews.map((r) => (
          <div key={r.id} className="review">
            <div className="review-head">
              <span>
                <Link className="link" to={`/users/${r.author.id}`}>{r.author.username}</Link>{' '}
                <Stars value={r.rating} />
              </span>
              <span className="muted">{formatDate(r.created_at)}</span>
            </div>
            {r.comment && <p>{r.comment}</p>}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Profile;