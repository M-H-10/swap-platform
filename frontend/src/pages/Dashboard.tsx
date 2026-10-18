import { useState, useEffect, type CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { getSwaps, getReputation, acceptSwap, rejectSwap, cancelSwap } from '../api';
import { useAuth } from '../context/auth-context';
import { errorMessage, formatPercent, formatRub } from '../lib/format';
import type { Reputation, Swap, SwapStatus } from '../types';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import toast from 'react-hot-toast';

const statusColors: Record<SwapStatus, string> = {
  proposed: '#f59e0b', active: '#6c63ff', completed: '#00d4aa',
  failed: '#e94560', cancelled: '#e94560', rejected: '#888888',
};

// Where the guarantee money is, derived from the swap status.
const escrowLabel: Record<SwapStatus, string> = {
  proposed: 'not locked yet', active: 'locked', completed: 'released',
  failed: 'paid as penalty', cancelled: 'returned', rejected: 'none',
};

const Dashboard = () => {
  const { user } = useAuth();
  const [swaps, setSwaps] = useState<Swap[]>([]);
  const [reputation, setReputation] = useState<Reputation | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) return;
    Promise.all([getSwaps(), getReputation(user.id)])
      .then(([s, r]) => {
        setSwaps(s.data.items);
        setReputation(r.data.reputation);
      })
      .catch((err) => toast.error(errorMessage(err, 'Failed to load')))
      .finally(() => setLoading(false));
  }, [user]);

  // Every action returns the updated swap, so we just replace it in the list.
  const runAction = async (action: (id: string) => Promise<{ data: Swap }>, id: string, done: string) => {
    try {
      const { data } = await action(id);
      setSwaps(list => list.map(s => s.id === id ? data : s));
      toast.success(done);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const handleCancel = (swap: Swap) => {
    const question = swap.status === 'active'
      ? 'An active swap is cancelled only when BOTH parties agree. Ask to cancel?'
      : 'Cancel this swap proposal?';
    if (!window.confirm(question)) return;
    runAction(cancelSwap, swap.id, swap.status === 'active' ? 'Cancel request sent.' : 'Swap cancelled.');
  };

  const chartData = swaps.slice(0, 6).map(s => ({
    name: s.project.title.substring(0, 10),
    escrow: s.terms.escrow_amount / 100, // rubles
    status: s.status,
  }));

  return (
    <div className="dashboard">
      <motion.div className="dashboard-header"
        initial={{ opacity: 0, y: -20 }} animate={{ opacity: 1, y: 0 }}>
        <h1>Dashboard</h1>
        <div className="user-profile">
          <div className="avatar">{user?.username?.[0]?.toUpperCase()}</div>
          <div>
            <h3>{user?.username}</h3>
            <span className="role-badge">{user?.role}</span>{' '}
            <Link className="link" to={`/users/${user?.id}`}>public profile →</Link>
          </div>
        </div>
      </motion.div>

      {reputation && (
        <div className="stats-row">
          {[
            { label: 'Reputation Score', value: formatPercent(reputation.score_bp), icon: '⭐', color: '#f59e0b' },
            { label: 'Completed Projects', value: reputation.completed_count, icon: '✅', color: '#00d4aa' },
            { label: 'Success Rate', value: formatPercent(reputation.success_rate_bp), icon: '📈', color: '#6c63ff' },
            { label: 'Active Swaps', value: swaps.filter(s => s.status === 'active').length, icon: '⚡', color: '#e94560' },
          ].map((stat, i) => (
            <motion.div key={stat.label} className="stat-card-dash"
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: i * 0.1 }}
              style={{ '--accent': stat.color } as CSSProperties}>
              <div className="stat-icon">{stat.icon}</div>
              <div className="stat-val">{stat.value}</div>
              <div className="stat-lbl">{stat.label}</div>
            </motion.div>
          ))}
        </div>
      )}

      {chartData.length > 0 && (
        <motion.div className="chart-section"
          initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.4 }}>
          <h3>Escrow Overview (₽)</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={chartData}>
              <XAxis dataKey="name" tick={{ fill: '#888', fontSize: 12 }} />
              <YAxis tick={{ fill: '#888', fontSize: 12 }} />
              <Tooltip contentStyle={{ background: '#1a1a2e', border: '1px solid #6c63ff' }} />
              <Bar dataKey="escrow" radius={[4, 4, 0, 0]}>
                {chartData.map((entry, i) => (
                  <Cell key={i} fill={statusColors[entry.status]} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </motion.div>
      )}

      <div className="swaps-list">
        <h3>My Swap Contracts</h3>
        {loading ? <div className="loading">Loading...</div> :
         swaps.length === 0 ? <div className="empty">No swap contracts yet. Create a project and find a partner!</div> :
         swaps.map((swap, i) => {
          const iAmA = swap.party_a.id === user?.id;
          const iAmB = swap.party_b.id === user?.id;
          const iAskedCancel = swap.cancel_requested_by === user?.id;
          const otherAskedCancel = swap.cancel_requested_by !== null && !iAskedCancel;
          return (
            <motion.div key={swap.id} className="swap-item"
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: i * 0.08 }}>
              <div className="swap-project">
                <Link className="card-title-link" to={`/swaps/${swap.id}`}>{swap.project.title}</Link>
              </div>
              <div className="swap-details">
                <span>Party A: <Link className="link" to={`/users/${swap.party_a.id}`}>{swap.party_a.username}</Link> ({formatPercent(swap.terms.share_a_bp)})</span>
                <span>Party B: <Link className="link" to={`/users/${swap.party_b.id}`}>{swap.party_b.username}</Link> ({formatPercent(swap.terms.share_b_bp)})</span>
                <span>Escrow: <strong>{formatRub(swap.terms.escrow_amount)}</strong></span>
              </div>
              <div className="swap-footer">
                <span className={`status-badge status-${swap.status}`}>{swap.status}</span>
                <span className="escrow-badge">🔒 {escrowLabel[swap.status]}</span>
                {otherAskedCancel && swap.status === 'active' && (
                  <span className="escrow-badge">the other party asked to cancel</span>
                )}

                {swap.status === 'proposed' && iAmB && (
                  <>
                    <motion.button className="btn-accept" whileHover={{ scale: 1.05 }}
                      onClick={() => runAction(acceptSwap, swap.id, 'Swap accepted! Escrow locked.')}>Accept ✓</motion.button>
                    <motion.button className="btn-cancel" whileHover={{ scale: 1.05 }}
                      onClick={() => runAction(rejectSwap, swap.id, 'Swap rejected.')}>Reject ✕</motion.button>
                  </>
                )}
                {swap.status === 'proposed' && iAmA && (
                  <motion.button className="btn-cancel" whileHover={{ scale: 1.05 }}
                    onClick={() => handleCancel(swap)}>Cancel ✕</motion.button>
                )}
                {(swap.status === 'active' || swap.status === 'completed' || swap.status === 'failed') && (
                  <Link className="btn-small" to={`/swaps/${swap.id}`}>
                    {swap.status === 'active' ? 'Report result →' : 'Details & review →'}
                  </Link>
                )}
                {swap.status === 'active' && !iAskedCancel && (
                  <motion.button className="btn-cancel" whileHover={{ scale: 1.05 }}
                    onClick={() => handleCancel(swap)}>
                    {otherAskedCancel ? 'Agree to cancel ✕' : 'Ask to cancel ✕'}
                  </motion.button>
                )}
              </div>
            </motion.div>
          );
        })}
      </div>
    </div>
  );
};

export default Dashboard;