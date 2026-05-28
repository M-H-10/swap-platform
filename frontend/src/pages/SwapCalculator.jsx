import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { RadarChart, Radar, PolarGrid, PolarAngleAxis, ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip } from 'recharts';
import { getProject, calculateSwap, createSwap } from '../api';
import { useAuth } from '../context/AuthContext';
import toast from 'react-hot-toast';

const SwapCalculator = () => {
  const { projectId } = useParams();
  const { user } = useAuth();
  const [project, setProject] = useState(null);
  const [partyBId, setPartyBId] = useState('');
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    getProject(projectId).then(({ data }) => setProject(data));
  }, [projectId]);

  const handleCalculate = async () => {
    if (!partyBId) { toast.error('Enter Party B user ID'); return; }
    setLoading(true);
    try {
      const { data } = await calculateSwap({ projectId, partyBId });
      setResult(data);
      toast.success('Swap calculated!');
    } catch (err) {
      toast.error(err.response?.data?.message || 'Calculation failed');
    }
    setLoading(false);
  };

  const handleCreateSwap = async () => {
    setCreating(true);
    try {
      await createSwap({ projectId, partyBId, partyARole: user.role, partyBRole: 'developer' });
      toast.success('Swap contract created!');
    } catch (err) {
      toast.error('Failed to create swap');
    }
    setCreating(false);
  };

  const riskData = result ? [
    { subject: 'Expected Value', A: result.partyA.riskProfile.expectedValue / 1000, B: result.partyB.riskProfile.expectedValue / 1000 },
    { subject: 'Max Gain', A: result.partyA.riskProfile.maxGain / 1000, B: result.partyB.riskProfile.maxGain / 1000 },
    { subject: 'Max Loss', A: -result.partyA.riskProfile.maxLoss / 1000, B: -result.partyB.riskProfile.maxLoss / 1000 },
  ] : [];

  const scenarioData = result && project ? [
    { name: 'Total Failure', A: -result.partyA.fixedPaymentOnFailure, B: -result.partyB.fixedPaymentOnFailure },
    { name: 'Partial (25%)', A: project.budget * 0.25 * result.swapRatio, B: project.budget * 0.25 * (1 - result.swapRatio) },
    { name: 'Break Even', A: project.budget * 0.5 * result.swapRatio, B: project.budget * 0.5 * (1 - result.swapRatio) },
    { name: 'Success (75%)', A: project.budget * 0.75 * result.swapRatio, B: project.budget * 0.75 * (1 - result.swapRatio) },
    { name: 'Full Success', A: project.budget * result.swapRatio, B: project.budget * (1 - result.swapRatio) },
  ] : [];

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
            <span>Success Probability: <strong className="prob">{project.successProbability}%</strong></span>
          </div>
        </motion.div>
      )}

      <motion.div className="calculator-input"
        initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2 }}>
        <h3>Party B Information</h3>
        <div className="input-row">
          <input placeholder="Enter Party B User ID" value={partyBId}
            onChange={e => setPartyBId(e.target.value)} />
          <motion.button className="btn-primary" onClick={handleCalculate} disabled={loading}
            whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
            {loading ? <span className="spinner" /> : 'Calculate Swap ⚡'}
          </motion.button>
        </div>
      </motion.div>

      {result && (
        <motion.div className="results-section"
          initial={{ opacity: 0, y: 40 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3 }}>

          <div className="success-probability">
            <div className="prob-gauge">
              <svg viewBox="0 0 120 60" className="gauge-svg">
                <path d="M10 60 A50 50 0 0 1 110 60" fill="none" stroke="#1a1a2e" strokeWidth="12" />
                <path d="M10 60 A50 50 0 0 1 110 60" fill="none"
                  stroke={result.successProbability > 60 ? '#00d4aa' : result.successProbability > 35 ? '#f59e0b' : '#e94560'}
                  strokeWidth="12" strokeDasharray={`${result.successProbability * 1.57} 157`} />
              </svg>
              <div className="gauge-value">{result.successProbability}%</div>
              <div className="gauge-label">Success Probability</div>
            </div>
          </div>

          <div className="swap-terms">
            <div className="party-card party-a">
              <h4>🅐 Party A (You)</h4>
              <div className="share-big">{result.partyA.shareOnSuccess}%</div>
              <div className="share-label">on success</div>
              <div className="payment">Pay ₽{result.partyA.fixedPaymentOnFailure?.toLocaleString()} on failure</div>
              <div className="risk-score">Risk Score: {result.partyA.riskProfile.riskScore}</div>
            </div>
            <div className="swap-icon">⚡</div>
            <div className="party-card party-b">
              <h4>🅑 Party B</h4>
              <div className="share-big">{result.partyB.shareOnSuccess}%</div>
              <div className="share-label">on success</div>
              <div className="payment">Pay ₽{result.partyB.fixedPaymentOnFailure?.toLocaleString()} on failure</div>
              <div className="risk-score">Risk Score: {result.partyB.riskProfile.riskScore}</div>
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
              <h4>Risk Profile Comparison</h4>
              <ResponsiveContainer width="100%" height={250}>
                <RadarChart data={riskData}>
                  <PolarGrid stroke="#2a2a4a" />
                  <PolarAngleAxis dataKey="subject" tick={{ fill: '#888', fontSize: 12 }} />
                  <Radar name="Party A" dataKey="A" stroke="#6c63ff" fill="#6c63ff" fillOpacity={0.3} />
                  <Radar name="Party B" dataKey="B" stroke="#e94560" fill="#e94560" fillOpacity={0.3} />
                </RadarChart>
              </ResponsiveContainer>
            </div>

            <div className="chart-card">
              <h4>Scenario Analysis (₽ thousands)</h4>
              <ResponsiveContainer width="100%" height={250}>
                <AreaChart data={scenarioData}>
                  <XAxis dataKey="name" tick={{ fill: '#888', fontSize: 11 }} />
                  <YAxis tick={{ fill: '#888', fontSize: 11 }} />
                  <Tooltip contentStyle={{ background: '#1a1a2e', border: '1px solid #6c63ff' }} />
                  <Area type="monotone" dataKey="A" stroke="#6c63ff" fill="#6c63ff" fillOpacity={0.3} name="Party A" />
                  <Area type="monotone" dataKey="B" stroke="#e94560" fill="#e94560" fillOpacity={0.3} name="Party B" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <motion.button className="btn-create-swap" onClick={handleCreateSwap}
            disabled={creating} whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
            {creating ? <span className="spinner" /> : '🚀 Create Swap Contract'}
          </motion.button>
        </motion.div>
      )}
    </div>
  );
};

export default SwapCalculator;