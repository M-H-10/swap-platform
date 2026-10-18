import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import toast from 'react-hot-toast';
import { estimateProject, getPrediction, getProject, requestPrediction } from '../api';
import { useAuth } from '../context/auth-context';
import { bpToPercent, errorMessage, formatDate, formatPercent, formatRub } from '../lib/format';
import type { PredictionComparison, Project, ScoreFactor } from '../types';

// The ML answer comes later through RabbitMQ (stage 7), so while it is
// "pending" we ask again every 3 seconds, at most 10 times.
const POLL_MS = 3000;
const MAX_POLLS = 10;

const factorLabel: Record<string, string> = {
  reputation: 'Your reputation',
  success_rate: 'Your success rate',
  experience: 'Completed projects',
  complexity: 'Complexity',
  account_age: 'Account age',
  team_size: 'Team size',
  budget: 'Budget per person and day',
  deadline: 'Time for this complexity',
};

const Bar = ({ label, bp, color }: { label: string; bp: number; color: string }) => (
  <div className="compare-row">
    <div className="compare-label"><span>{label}</span><strong>{formatPercent(bp)}</strong></div>
    <div className="bar-track">
      <div className="bar-fill" style={{ width: `${bpToPercent(bp)}%`, background: color }} />
    </div>
  </div>
);

const ProjectDetail = () => {
  const { id = '' } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [project, setProject] = useState<Project | null>(null);
  const [prediction, setPrediction] = useState<PredictionComparison | null>(null);
  const [factors, setFactors] = useState<ScoreFactor[]>([]);
  const [polls, setPolls] = useState(0);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([getProject(id), getPrediction(id)])
      .then(([p, pr]) => {
        setProject(p.data);
        setPrediction(pr.data);
      })
      .catch((err) => setError(errorMessage(err, 'Project not found')));
  }, [id]);

  // While the model has not answered, check again a few times.
  useEffect(() => {
    if (prediction?.status !== 'pending' || polls >= MAX_POLLS) return;
    const timer = setTimeout(() => {
      getPrediction(id)
        .then(({ data }) => setPrediction(data))
        .catch(() => {})
        .finally(() => setPolls((n) => n + 1));
    }, POLL_MS);
    return () => clearTimeout(timer);
  }, [prediction, polls, id]);

  // The formula's explanation (which factor gave how many points). The
  // server computes it with the creator's CURRENT statistics, so only the
  // creator can ask for it.
  const isCreator = !!user && !!project && project.creator.id === user.id;
  useEffect(() => {
    if (!isCreator || !project) return;
    estimateProject({
      title: project.title, description: project.description, category: project.category,
      budget: project.budget, duration_days: project.duration_days, complexity: project.complexity,
      team_size: project.team_size, kpis: project.kpis,
    })
      .then(({ data }) => setFactors(data.factors))
      .catch(() => {});
  }, [isCreator, project]);

  const askAgain = async () => {
    try {
      await requestPrediction(id);
      toast.success('Sent to the model. The answer comes in a few seconds.');
      setPolls(0);
      setPrediction((p) => (p ? { ...p, status: 'pending' } : p));
    } catch (err) {
      toast.error(errorMessage(err, 'The ML service is not available'));
    }
  };

  if (error) return <div className="empty">{error}</div>;
  if (!project || !prediction) return <div className="loading">Loading...</div>;

  const formulaBP = prediction.formula.success_probability_bp;
  const ml = prediction.ml;
  const diff = prediction.difference_bp;
  const waiting = prediction.status === 'pending' && polls < MAX_POLLS;

  return (
    <div className="page">
      <Link className="back-link" to="/projects">← Projects</Link>
      <div className="page-title">
        <h1>{project.title}</h1>
        <span className="category-badge">{project.category}</span>
        <span className="tag">{project.status.replace('_', ' ')}</span>
        <p className="muted">
          by <Link className="link" to={`/users/${project.creator.id}`}>{project.creator.username}</Link>
          {' '}· created {formatDate(project.created_at)}
        </p>
      </div>

      <div className="panel">
        <p style={{ whiteSpace: 'pre-line' }}>{project.description}</p>
        {isCreator && project.status === 'open' && (
          <div className="actions">
            <button className="btn-primary" onClick={() => navigate(`/swap-calculator/${project.id}`)}>
              Find a partner
            </button>
          </div>
        )}
      </div>

      <div className="grid-2">
        <div className="panel">
          <h3>Details</h3>
          <dl className="kv">
            <dt>Budget</dt><dd>{formatRub(project.budget)}</dd>
            <dt>Duration</dt><dd>{project.duration_days} days</dd>
            <dt>Complexity</dt><dd>{project.complexity} / 10</dd>
            <dt>Team size</dt><dd>{project.team_size}</dd>
          </dl>
          {project.kpis.length > 0 && (
            <>
              <h3 style={{ marginTop: '1.5rem' }}>KPIs</h3>
              <dl className="kv">
                {project.kpis.map((k, i) => (
                  <div key={i} style={{ display: 'contents' }}>
                    <dt>{k.name}: {k.target}</dt><dd>{k.weight}%</dd>
                  </div>
                ))}
              </dl>
            </>
          )}
        </div>

        <motion.div className="panel" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }}>
          <h3>Success probability: formula vs ML model</h3>
          <Bar label="Formula" bp={formulaBP} color="var(--text)" />
          {ml ? (
            <>
              <Bar label={`ML model (${ml.model_version})`} bp={ml.success_probability_bp}
                color="var(--accent)" />
              <p className="muted">
                Difference: <strong>{diff !== null && diff > 0 ? '+' : ''}{formatPercent(diff ?? 0)}</strong>
                {' '}({diff === 0 ? 'they agree' : diff !== null && diff > 0 ? 'the model is more optimistic' : 'the model is more careful'})
              </p>
            </>
          ) : waiting ? (
            <p className="note"><span className="spinner" /> The model is calculating...</p>
          ) : (
            <>
              <p className="note note-warning">
                No answer from the ML model yet. Is the Python worker running?
              </p>
              {user && (
                <div className="actions">
                  <button className="btn-small" onClick={askAgain}>Ask the model again</button>
                </div>
              )}
            </>
          )}
          <p className="muted" style={{ marginTop: '1rem' }}>
            The formula uses fixed rules written by hand. The model learned from data (RandomForest).
            Showing both lets you compare the two approaches on the same project.
          </p>
        </motion.div>
      </div>

      {factors.length > 0 && (
        <div className="panel">
          <h3>Why the formula gives this number</h3>
          {factors.map((f) => (
            <div key={f.name} className="factor-row">
              <span>{factorLabel[f.name] ?? f.name}</span>
              <span className={f.points > 0 ? 'amount-plus' : f.points < 0 ? 'amount-minus' : 'amount-neutral'}>
                {f.points > 0 ? '+' : ''}{(f.points / 100).toFixed(1)} points
              </span>
            </div>
          ))}
          <p className="muted" style={{ marginTop: '1rem' }}>
            Calculated with your current statistics, so it can differ a little from the saved number
            if your reputation changed after the project was created.
          </p>
        </div>
      )}
    </div>
  );
};

export default ProjectDetail;