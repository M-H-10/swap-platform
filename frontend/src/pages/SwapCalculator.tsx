import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
  RadarChart, Radar, PolarGrid, PolarAngleAxis, ResponsiveContainer,
  BarChart, Bar, XAxis, YAxis, Tooltip, Legend,
} from 'recharts';
import { getProject, calculateSwap, createSwap, searchUsers } from '../api';
import { useAuth } from '../context/auth-context';
import { bpToPercent, errorMessage, formatPercent, formatRub, probabilityColor } from '../lib/format';
import type { Project, PublicUser, SwapCalculation } from '../types';
import toast from 'react-hot-toast';
import { Search, User } from 'lucide-react';

const SwapCalculator = () => {
  const { projectId = '' } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [project, setProject] = useState<Project | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<PublicUser[]>([]);
  const [selectedUser, setSelectedUser] = useState<PublicUser | null>(null);
  const [result, setResult] = useState<SwapCalculation | null>(null);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    getProject(projectId)
      .then(({ data }) => setProject(data))
      .catch((err) => toast.error(errorMessage(err, 'Project not found')));
  }, [projectId]);

  // Search for party B by username (the server leaves you out of the results).
  const handleSearch = async (q: string) => {
    setSearchQuery(q);
    setSelectedUser(null);
    setResult(null);
    if (q.trim().length < 2) { setSearchResults([]); return; }
    try {
      const { data } = await searchUsers(q.trim());
      setSearchResults(data.items);
    } catch {
      setSearchResults([]);
    }
  };

  const selectUser = (u: PublicUser) => {
    setSelectedUser(u);
    setSearchQuery(u.username);
    setSearchResults([]);
  };

  const handleCalculate = async () => {
    if (!selectedUser) { toast.error('Please select Party B first'); return; }
    setLoading(true);
    try {
      const { data } = await calculateSwap({ project_id: projectId, partner_id: selectedUser.id });
      setResult(data);
      toast.success('Swap calculated!');
    } catch (err) {
      toast.error(errorMessage(err, 'Calculation failed'));
    }
    setLoading(false);
  };

  const handleCreateSwap = async () => {
    if (!selectedUser || !result) return;
    setCreating(true);
    try {
      await createSwap({ project_id: projectId, partner_id: selectedUser.id });
      toast.success('Swap contract proposed! Party B will see it in their Dashboard.');
      navigate('/dashboard');
    } catch (err) {
      toast.error(errorMessage(err, 'Failed to create swap'));
    }
    setCreating(false);
  };

  const t = result?.terms;
  const thousands = (kopecks: number) => kopecks / 100 / 1000; // kopecks -> thousands of rubles

  const riskData = t ? [
    { subject: 'Expected Value', A: Math.max(thousands(t.risk_a.expected_value), 0), B: Math.max(thousands(t.risk_b.expected_value), 0) },
    { subject: 'Max Gain', A: thousands(t.risk_a.max_gain), B: thousands(t.risk_b.max_gain) },
    { subject: 'Max Loss', A: thousands(t.risk_a.max_loss), B: thousands(t.risk_b.max_loss) },
  ] : [];

  // What each party gets in the two possible endings (rubles).
  const scenarioData = t ? [
    { name: 'Failure', A: -t.risk_a.max_loss / 100, B: -t.risk_b.max_loss / 100 },
    { name: 'Success', A: t.risk_a.max_gain / 100, B: t.risk_b.max_gain / 100 },
  ] : [];

  const probability = t?.success_probability_bp ?? 0;

  return (
    <div className="swap-calculator">
      <motion.h1 initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
        ⚡ Swap Calculator
      </motion.h1>

      {project && (
        <motion.div className="project-info-card"
          initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }}>
          <h3>{project.title}</h3>
          <div className="info-row">
            <span>Budget: <strong>{formatRub(project.budget)}</strong></span>
            <span>Duration: <strong>{project.duration_days} days</strong></span>
            <span>Success Probability: <strong style={{ color: probabilityColor(project.success_probability_bp) }}>{formatPercent(project.success_probability_bp)}</strong></span>
          </div>
          {user && project.creator.id !== user.id && (
            <p style={{ color: '#e94560', marginTop: '8px' }}>Only the creator of this project can propose a swap.</p>
          )}
        </motion.div>
      )}

      <motion.div className="calculator-input"
        initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2 }}>
        <h3>Find Party B by Username</h3>
        <div className="search-wrapper" style={{ position: 'relative' }}>
          <div className="input-row">
            <div style={{ position: 'relative', flex: 1 }}>
              <Search size={16} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: '#888' }} />
              <input
                style={{ paddingLeft: '36px' }}
                placeholder="Search by username..."
                value={searchQuery}
                onChange={(e) => handleSearch(e.target.value)}
              />
              <AnimatePresence>
                {searchResults.length > 0 && (
                  <motion.div
                    initial={{ opacity: 0, y: -10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -10 }}
                    style={{
                      position: 'absolute', top: '100%', left: 0, right: 0,
                      background: '#1a1a2e', border: '1px solid rgba(108,99,255,0.4)',
                      borderRadius: '8px', zIndex: 10, marginTop: '4px'
                    }}>
                    {searchResults.map(u => (
                      <div
                        key={u.id}
                        onClick={() => selectUser(u)}
                        style={{
                          padding: '10px 16px', cursor: 'pointer', display: 'flex',
                          alignItems: 'center', gap: '10px', borderBottom: '1px solid rgba(255,255,255,0.05)'
                        }}
                        onMouseEnter={e => { e.currentTarget.style.background = 'rgba(108,99,255,0.15)'; }}
                        onMouseLeave={e => { e.currentTarget.style.background = 'transparent'; }}
                      >
                        <User size={14} color="#6c63ff" />
                        <span style={{ fontWeight: 600 }}>{u.username}</span>
                        <span style={{ color: '#888', fontSize: '0.8rem' }}>{u.role}</span>
                        <span style={{ marginLeft: 'auto', color: '#f59e0b', fontSize: '0.8rem' }}>⭐ {formatPercent(u.reputation_bp)}</span>
                      </div>
                    ))}
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
            <motion.button
              className="btn-primary"
              onClick={handleCalculate}
              disabled={loading || !selectedUser}
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}>
              {loading ? <span className="spinner" /> : 'Calculate ⚡'}
            </motion.button>
          </div>
        </div>

        {selectedUser && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            style={{
              marginTop: '12px', padding: '10px 16px',
              background: 'rgba(0,212,170,0.1)', border: '1px solid rgba(0,212,170,0.3)',
              borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '12px'
            }}>
            <User size={16} color="#00d4aa" />
            <span>Selected: <strong>{selectedUser.username}</strong></span>
            <span style={{ color: '#888' }}>{selectedUser.role}</span>
            <span style={{ color: '#f59e0b' }}>⭐ reputation {formatPercent(selectedUser.reputation_bp)}</span>
          </motion.div>
        )}
      </motion.div>

      {result && t && (
        <motion.div className="results-section"
          initial={{ opacity: 0, y: 40 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3 }}>

          <div className="success-probability">
            <div className="prob-gauge">
              <svg viewBox="0 0 120 60" className="gauge-svg">
                <path d="M10 60 A50 50 0 0 1 110 60" fill="none" stroke="#1a1a2e" strokeWidth="12" />
                <path d="M10 60 A50 50 0 0 1 110 60" fill="none"
                  stroke={probabilityColor(probability)}
                  strokeWidth="12"
                  strokeDasharray={`${bpToPercent(probability) * 1.57} 157`} />
              </svg>
              <div className="gauge-value">{formatPercent(probability)}</div>
              <div className="gauge-label">Success Probability</div>
            </div>
          </div>

          <div className="swap-terms">
            <div className="party-card party-a">
              <h4>🅐 {result.party_a.username} (You)</h4>
              <div className="share-big">{formatPercent(t.share_a_bp)}</div>
              <div className="share-label">on success</div>
              <div className="payment">Guarantee: {formatRub(t.penalty_a)}, lost on failure</div>
              <div style={{ color: '#888', fontSize: '0.8rem', marginTop: '4px' }}>
                Expected value: {formatRub(t.risk_a.expected_value)}
              </div>
            </div>
            <div className="swap-icon">⚡</div>
            <div className="party-card party-b">
              <h4>🅑 {result.party_b.username}</h4>
              <div className="share-big">{formatPercent(t.share_b_bp)}</div>
              <div className="share-label">on success</div>
              <div className="payment">Guarantee: {formatRub(t.penalty_b)}, lost on failure</div>
              <div style={{ color: '#888', fontSize: '0.8rem', marginTop: '4px' }}>
                Expected value: {formatRub(t.risk_b.expected_value)}
              </div>
            </div>
          </div>

          <div className="escrow-info">
            <div className="escrow-card">
              🔒 Escrow Account: <strong>{formatRub(t.escrow_amount)}</strong>
              <span className="escrow-badge">Locked when Party B accepts</span>
            </div>
          </div>

          <div className="charts-grid">
            <div className="chart-card">
              <h4>Risk Profile Comparison (₽ thousands)</h4>
              <ResponsiveContainer width="100%" height={250}>
                <RadarChart data={riskData}>
                  <PolarGrid stroke="#2a2a4a" />
                  <PolarAngleAxis dataKey="subject" tick={{ fill: '#888', fontSize: 11 }} />
                  <Radar name="Party A" dataKey="A" stroke="#6c63ff" fill="#6c63ff" fillOpacity={0.3} />
                  <Radar name="Party B" dataKey="B" stroke="#e94560" fill="#e94560" fillOpacity={0.3} />
                </RadarChart>
              </ResponsiveContainer>
            </div>
            <div className="chart-card">
              <h4>Result for each party (₽)</h4>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={scenarioData}>
                  <XAxis dataKey="name" tick={{ fill: '#888', fontSize: 11 }} />
                  <YAxis tick={{ fill: '#888', fontSize: 11 }} />
                  <Tooltip contentStyle={{ background: '#1a1a2e', border: '1px solid #6c63ff' }}
                    formatter={(v) => `₽${Number(v).toLocaleString('en-US')}`} />
                  <Legend />
                  <Bar dataKey="A" fill="#6c63ff" name={result.party_a.username} />
                  <Bar dataKey="B" fill="#e94560" name={result.party_b.username} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>

          <motion.button
            className="btn-create-swap"
            onClick={handleCreateSwap}
            disabled={creating}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}>
            {creating ? <span className="spinner" /> : '🚀 Propose Swap Contract'}
          </motion.button>
        </motion.div>
      )}
    </div>
  );
};

export default SwapCalculator;