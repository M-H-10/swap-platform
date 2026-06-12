import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { RadarChart, Radar, PolarGrid, PolarAngleAxis, ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip } from 'recharts';
import { getProject, calculateSwap, createSwap, searchUsers } from '../api';
import { useAuth } from '../context/AuthContext';
import toast from 'react-hot-toast';
import { Search, User } from 'lucide-react';

const SwapCalculator = () => {
  const { projectId } = useParams();
  const { user } = useAuth();
  const [project, setProject] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [selectedUser, setSelectedUser] = useState(null);
  const [searching, setSearching] = useState(false);
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    getProject(projectId).then(({ data }) => setProject(data));
  }, [projectId]);

  // البحث عن مستخدم بالاسم
  const handleSearch = async (q) => {
    setSearchQuery(q);
    setSelectedUser(null);
    setResult(null);
    if (q.length < 2) { setSearchResults([]); return; }
    setSearching(true);
    try {
      const { data } = await searchUsers(q);
      setSearchResults(data);
    } catch {
      setSearchResults([]);
    }
    setSearching(false);
  };

  const selectUser = (u) => {
    setSelectedUser(u);
    setSearchQuery(u.username);
    setSearchResults([]);
  };

  const handleCalculate = async () => {
    if (!selectedUser) { toast.error('Please select Party B first'); return; }
    setLoading(true);
    try {
      const { data } = await calculateSwap({ projectId, partyBId: selectedUser._id });
      setResult(data);
      toast.success('Swap calculated!');
    } catch (err) {
      toast.error(err.response?.data?.message || 'Calculation failed');
    }
    setLoading(false);
  };

  const handleCreateSwap = async () => {
    if (!selectedUser || !result) return;
    setCreating(true);
    try {
      await createSwap({
        projectId,
        partyBId: selectedUser._id,
        partyARole: user.role || 'developer',
        partyBRole: selectedUser.role || 'developer'
      });
      toast.success('Swap contract created! Party B will see it in their Dashboard.');
    } catch (err) {
      toast.error(err.response?.data?.message || 'Failed to create swap');
    }
    setCreating(false);
  };

  const riskData = result ? [
    { subject: 'Expected Value', A: Math.max(result.partyA?.riskProfile?.expectedValue || 0, 0) / 1000, B: Math.max(result.partyB?.riskProfile?.expectedValue || 0, 0) / 1000 },
    { subject: 'Max Gain', A: (result.partyA?.riskProfile?.maxGain || 0) / 1000, B: (result.partyB?.riskProfile?.maxGain || 0) / 1000 },
    { subject: 'Max Loss', A: (result.partyA?.riskProfile?.maxLoss || 0) / 1000, B: (result.partyB?.riskProfile?.maxLoss || 0) / 1000 },
  ] : [];

  const scenarioData = result && project ? [
    { name: 'Failure', A: -(result.partyA?.fixedPaymentOnFailure || 0), B: -(result.partyB?.fixedPaymentOnFailure || 0) },
    { name: '25%', A: Math.round(project.budget * 0.25 * result.swapRatio), B: Math.round(project.budget * 0.25 * (1 - result.swapRatio)) },
    { name: '50%', A: Math.round(project.budget * 0.5 * result.swapRatio), B: Math.round(project.budget * 0.5 * (1 - result.swapRatio)) },
    { name: '75%', A: Math.round(project.budget * 0.75 * result.swapRatio), B: Math.round(project.budget * 0.75 * (1 - result.swapRatio)) },
    { name: 'Full', A: Math.round(project.budget * result.swapRatio), B: Math.round(project.budget * (1 - result.swapRatio)) },
  ] : [];

  const getProbColor = (p) => p >= 70 ? '#00d4aa' : p >= 40 ? '#f59e0b' : '#e94560';

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
            <span>Budget: <strong>₽{project.budget?.toLocaleString()}</strong></span>
            <span>Duration: <strong>{project.duration} days</strong></span>
            <span>Success Probability: <strong style={{ color: getProbColor(project.successProbability) }}>{project.successProbability}%</strong></span>
          </div>
        </motion.div>
      )}

      {/* بحث بالاسم */}
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
              {/* نتائج البحث */}
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
                        key={u._id}
                        onClick={() => selectUser(u)}
                        style={{
                          padding: '10px 16px', cursor: 'pointer', display: 'flex',
                          alignItems: 'center', gap: '10px', borderBottom: '1px solid rgba(255,255,255,0.05)'
                        }}
                        onMouseEnter={e => e.currentTarget.style.background = 'rgba(108,99,255,0.15)'}
                        onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
                      >
                        <User size={14} color="#6c63ff" />
                        <span style={{ fontWeight: 600 }}>{u.username}</span>
                        <span style={{ color: '#888', fontSize: '0.8rem' }}>{u.role}</span>
                        <span style={{ marginLeft: 'auto', color: '#f59e0b', fontSize: '0.8rem' }}>⭐ {u.reputation}</span>
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
            <span style={{ color: '#f59e0b' }}>⭐ {selectedUser.reputation}</span>
            <span style={{ color: '#6c63ff' }}>✓ {selectedUser.successRate}% success</span>
          </motion.div>
        )}
      </motion.div>

      {result && (
        <motion.div className="results-section"
          initial={{ opacity: 0, y: 40 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3 }}>

          <div className="success-probability">
            <div className="prob-gauge">
              <svg viewBox="0 0 120 60" className="gauge-svg">
                <path d="M10 60 A50 50 0 0 1 110 60" fill="none" stroke="#1a1a2e" strokeWidth="12" />
                <path d="M10 60 A50 50 0 0 1 110 60" fill="none"
                  stroke={getProbColor(result.successProbability)}
                  strokeWidth="12"
                  strokeDasharray={`${result.successProbability * 1.57} 157`} />
              </svg>
              <div className="gauge-value">{result.successProbability}%</div>
              <div className="gauge-label">Success Probability</div>
            </div>
          </div>

          <div className="swap-terms">
            <div className="party-card party-a">
              <h4>🅐 Party A (You)</h4>
              <div className="share-big">{result.partyA?.shareOnSuccess}%</div>
              <div className="share-label">on success</div>
              <div className="payment">Pay ₽{result.partyA?.fixedPaymentOnFailure?.toLocaleString()} on failure</div>
              <div className="risk-score">Risk Score: {result.partyA?.riskProfile?.riskScore}</div>
              <div style={{ color: '#888', fontSize: '0.8rem', marginTop: '4px' }}>
                EV: ₽{result.partyA?.riskProfile?.expectedValue?.toLocaleString()}
              </div>
            </div>
            <div className="swap-icon">⚡</div>
            <div className="party-card party-b">
              <h4>🅑 {selectedUser?.username}</h4>
              <div className="share-big">{result.partyB?.shareOnSuccess}%</div>
              <div className="share-label">on success</div>
              <div className="payment">Pay ₽{result.partyB?.fixedPaymentOnFailure?.toLocaleString()} on failure</div>
              <div className="risk-score">Risk Score: {result.partyB?.riskProfile?.riskScore}</div>
              <div style={{ color: '#888', fontSize: '0.8rem', marginTop: '4px' }}>
                EV: ₽{result.partyB?.riskProfile?.expectedValue?.toLocaleString()}
              </div>
            </div>
          </div>

          <div className="escrow-info">
            <div className="escrow-card">
              🔒 Escrow Account: <strong>₽{result.escrowAmount?.toLocaleString()}</strong>
              <span className="escrow-badge">Locked until completion</span>
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
              <h4>Scenario Analysis (₽)</h4>
              <ResponsiveContainer width="100%" height={250}>
                <AreaChart data={scenarioData}>
                  <XAxis dataKey="name" tick={{ fill: '#888', fontSize: 11 }} />
                  <YAxis tick={{ fill: '#888', fontSize: 11 }} />
                  <Tooltip contentStyle={{ background: '#1a1a2e', border: '1px solid #6c63ff' }}
                    formatter={(v) => `₽${v.toLocaleString()}`} />
                  <Area type="monotone" dataKey="A" stroke="#6c63ff" fill="#6c63ff" fillOpacity={0.3} name="Party A" />
                  <Area type="monotone" dataKey="B" stroke="#e94560" fill="#e94560" fillOpacity={0.3} name="Party B" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <motion.button
            className="btn-create-swap"
            onClick={handleCreateSwap}
            disabled={creating}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}>
            {creating ? <span className="spinner" /> : '🚀 Create Swap Contract'}
          </motion.button>
        </motion.div>
      )}
    </div>
  );
};

export default SwapCalculator;