import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
  RadarChart, Radar, PolarGrid, PolarAngleAxis, ResponsiveContainer,
  BarChart, Bar, XAxis, YAxis, Tooltip, Legend,
} from 'recharts';
import { getProject, calculateSwap, createSwap, searchUsers } from '../api';
import { useAuth } from '../context/auth-context';
import { errorMessage, formatPercent, formatRub, probabilityColor } from '../lib/format';
import { useChartTheme } from '../lib/chart-theme';
import TermSheet from '../components/TermSheet';
import type { Project, PublicUser, SwapCalculation } from '../types';
import toast from 'react-hot-toast';
import { Search, User } from 'lucide-react';

const SwapCalculator = () => {
  const { projectId = '' } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const chart = useChartTheme();
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
        Swap calculator
      </motion.h1>

      {project && (
        <motion.div className="project-info-card"
          initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
          <h3>{project.title}</h3>
          <div className="info-row">
            <span>Budget <strong>{formatRub(project.budget)}</strong></span>
            <span>Duration <strong>{project.duration_days} days</strong></span>
            <span>Success probability <strong style={{ color: probabilityColor(project.success_probability_bp) }}>{formatPercent(project.success_probability_bp)}</strong></span>
          </div>
          {user && project.creator.id !== user.id && (
            <p className="form-error">Only the creator of this project can propose a swap.</p>
          )}
        </motion.div>
      )}

      <motion.div className="calculator-input"
        initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.05, duration: 0.3 }}>
        <h3>Find party B by username</h3>
        <div className="input-row">
          <div className="search-box">
            <Search size={16} />
            <input
              placeholder="Search by username..."
              aria-label="Search by username"
              value={searchQuery}
              onChange={(e) => handleSearch(e.target.value)}
            />
            <AnimatePresence>
              {searchResults.length > 0 && (
                <motion.div className="search-results"
                  initial={{ opacity: 0, y: -4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: 0.15 }}>
                  {searchResults.map(u => (
                    <button type="button" key={u.id} className="search-result" onClick={() => selectUser(u)}>
                      <User size={14} />
                      <strong>{u.username}</strong>
                      <span className="role">{u.role}</span>
                      <span className="rep">reputation {formatPercent(u.reputation_bp)}</span>
                    </button>
                  ))}
                </motion.div>
              )}
            </AnimatePresence>
          </div>
          <button className="btn-primary" onClick={handleCalculate} disabled={loading || !selectedUser}>
            {loading ? <span className="spinner" /> : 'Calculate terms'}
          </button>
        </div>

        {selectedUser && (
          <motion.div className="selected-user" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
            <User size={16} />
            <span>Selected <strong>{selectedUser.username}</strong></span>
            <span className="muted-inline">{selectedUser.role}</span>
            <span className="muted-inline">reputation {formatPercent(selectedUser.reputation_bp)}</span>
          </motion.div>
        )}
      </motion.div>

      {result && t && (
        <motion.div className="results-section"
          initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.35, ease: [0.23, 1, 0.32, 1] }}>

          <section className="panel term-sheet-card">
            <h3>Proposed terms · success probability {formatPercent(probability)}</h3>
            <TermSheet partyA={result.party_a.username} partyB={result.party_b.username}
              terms={t} escrowLabel="Locked when B accepts" />
          </section>

          <div className="risk-row">
            {[
              { name: result.party_a.username, r: t.risk_a },
              { name: result.party_b.username, r: t.risk_b },
            ].map(({ name, r }) => (
              <div className="risk-card" key={name}>
                <h4>{name}</h4>
                <dl className="kv">
                  <dt>Expected value</dt><dd>{formatRub(r.expected_value)}</dd>
                  <dt>On success</dt><dd className="amount-plus">+{formatRub(r.max_gain)}</dd>
                  <dt>On failure</dt><dd className="amount-minus">−{formatRub(r.max_loss)}</dd>
                </dl>
              </div>
            ))}
          </div>

          <div className="charts-grid">
            <div className="chart-card">
              <h4>Risk profile (₽ thousands)</h4>
              <ResponsiveContainer width="100%" height={250}>
                <RadarChart data={riskData}>
                  <PolarGrid stroke={chart.grid} />
                  <PolarAngleAxis dataKey="subject" tick={chart.tick} />
                  <Radar name={result.party_a.username} dataKey="A" stroke={chart.partyA} fill={chart.partyA} fillOpacity={0.25} />
                  <Radar name={result.party_b.username} dataKey="B" stroke={chart.partyB} fill={chart.partyB} fillOpacity={0.35} />
                  <Legend wrapperStyle={{ fontSize: 13 }} />
                </RadarChart>
              </ResponsiveContainer>
            </div>
            <div className="chart-card">
              <h4>Result for each party (₽)</h4>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={scenarioData}>
                  <XAxis dataKey="name" tick={chart.tick} stroke={chart.grid} tickLine={false} />
                  <YAxis tick={chart.tick} stroke={chart.grid} tickLine={false} axisLine={false} width={64} />
                  <Tooltip contentStyle={chart.tooltip} cursor={chart.tooltipCursor}
                    formatter={(v) => `₽${Number(v).toLocaleString('en-US')}`} />
                  <Legend wrapperStyle={{ fontSize: 13 }} />
                  <Bar dataKey="A" fill={chart.partyA} name={result.party_a.username} radius={[3, 3, 0, 0]} />
                  <Bar dataKey="B" fill={chart.partyB} name={result.party_b.username} radius={[3, 3, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>

          <button className="btn-create-swap" onClick={handleCreateSwap} disabled={creating}>
            {creating ? <span className="spinner" /> : `Propose this swap to ${result.party_b.username}`}
          </button>
        </motion.div>
      )}
    </div>
  );
};

export default SwapCalculator;