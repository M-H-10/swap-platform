import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { getSwaps, getMe, acceptSwap } from '../api';
import { useAuth } from '../context/AuthContext';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import toast from 'react-hot-toast';

const Dashboard = () => {
  const { user } = useAuth();
  const [swaps, setSwaps] = useState([]);
  const [profile, setProfile] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([getSwaps(), getMe()]).then(([s, p]) => {
      setSwaps(s.data);
      setProfile(p.data);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const handleAccept = async (id) => {
    try {
      await acceptSwap(id);
      setSwaps(swaps.map(s => s._id === id ? {...s, status: 'active', escrowStatus: 'locked'} : s));
      toast.success('Swap accepted! Escrow locked.');
    } catch {
      toast.error('Failed to accept');
    }
  };

  const chartData = swaps.slice(0, 6).map(s => ({
    name: s.project?.title?.substring(0, 10) || 'Project',
    escrow: s.escrowAmount,
    status: s.status
  }));

  const statusColors = { proposed: '#f59e0b', active: '#6c63ff', completed: '#00d4aa', cancelled: '#e94560' };

  return (
    <div className="dashboard">
      <motion.div className="dashboard-header"
        initial={{ opacity: 0, y: -20 }} animate={{ opacity: 1, y: 0 }}>
        <h1>Dashboard</h1>
        <div className="user-profile">
          <div className="avatar">{user?.username?.[0]?.toUpperCase()}</div>
          <div>
            <h3>{user?.username}</h3>
            <span className="role-badge">{user?.role}</span>
          </div>
        </div>
      </motion.div>

      {profile && (
        <div className="stats-row">
          {[
            { label: 'Reputation Score', value: profile.reputation, icon: '⭐', color: '#f59e0b' },
            { label: 'Completed Projects', value: profile.completedProjects, icon: '✅', color: '#00d4aa' },
            { label: 'Success Rate', value: `${profile.successRate}%`, icon: '📈', color: '#6c63ff' },
            { label: 'Active Swaps', value: swaps.filter(s => s.status === 'active').length, icon: '⚡', color: '#e94560' },
          ].map((stat, i) => (
            <motion.div key={i} className="stat-card-dash"
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: i * 0.1 }}
              style={{ '--accent': stat.color }}>
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
                  <Cell key={i} fill={statusColors[entry.status] || '#6c63ff'} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </motion.div>
      )}

      <div className="swaps-list">
        <h3>My Swap Contracts</h3>
        {loading ? <div className="loading">Loading...</div> :
         swaps.length === 0 ? <div className="empty">No swap contracts yet. Browse projects to create one!</div> :
         swaps.map((swap, i) => (
          <motion.div key={swap._id} className="swap-item"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: i * 0.08 }}>
            <div className="swap-project">{swap.project?.title || 'Project'}</div>
            <div className="swap-details">
              <span>Party A: <strong>{swap.partyA?.user?.username}</strong> ({swap.partyA?.shareOnSuccess}%)</span>
              <span>Party B: <strong>{swap.partyB?.user?.username}</strong> ({swap.partyB?.shareOnSuccess}%)</span>
              <span>Escrow: <strong>₽{swap.escrowAmount?.toLocaleString()}</strong></span>
            </div>
            <div className="swap-footer">
              <span className={`status-badge status-${swap.status}`}>{swap.status}</span>
              <span className={`escrow-badge escrow-${swap.escrowStatus}`}>🔒 {swap.escrowStatus}</span>
              {swap.status === 'proposed' && swap.partyB?.user?._id === user?.userId && (
                <motion.button className="btn-accept" onClick={() => handleAccept(swap._id)}
                  whileHover={{ scale: 1.05 }}>Accept ✓</motion.button>
              )}
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  );
};

export default Dashboard;