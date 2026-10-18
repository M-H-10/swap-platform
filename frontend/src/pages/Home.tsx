import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import TermSheet from '../components/TermSheet';
import { useAuth } from '../context/auth-context';
import type { SwapTerms } from '../types';

// An example contract for the first screen. These are the real numbers that
// the calculator gives for the seed project "Brand identity for a coffee shop".
const noRisk = { expected_value: 0, max_gain: 0, max_loss: 0 };
const exampleTerms: SwapTerms = {
  share_a_bp: 7090, share_b_bp: 2910,
  escrow_amount: 800_000, penalty_a: 232_800, penalty_b: 567_200,
  success_probability_bp: 4820, risk_a: noRisk, risk_b: noRisk,
};

const steps = [
  {
    title: 'The calculator proposes terms',
    text: 'It reads the project and the reputation of both people, and gives a share of the result and a guarantee for each side.',
  },
  {
    title: 'Guarantees go into escrow',
    text: 'When the partner accepts, both guarantees are locked. Nobody can take the money out alone.',
  },
  {
    title: 'Both report the result',
    text: 'If the answers match, the escrow is released or paid as penalties, and the reputation of both people is updated.',
  },
];

// One quick, quiet entrance: a short rise and a fade, nothing bounces.
const rise = (delay = 0) => ({
  initial: { opacity: 0, y: 10 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.4, delay, ease: [0.23, 1, 0.32, 1] as const },
});

const Home = () => {
  const { user } = useAuth();

  return (
    <div className="home">
      <section className="home-hero">
        <motion.div {...rise()}>
          <p className="eyebrow">Reputation swaps for freelancers and startups</p>
          <h1>Share the result of a project, not only the invoice.</h1>
          <p className="lead">
            Swap Platform writes a fair contract between a project owner and a partner: who gets which
            share if the project succeeds, and how much each side guarantees if it fails.
          </p>
          <div className="hero-buttons">
            <Link to="/projects" className="btn-primary">Browse projects</Link>
            {user
              ? <Link to="/dashboard" className="btn-secondary">My swaps</Link>
              : <Link to="/register" className="btn-secondary">Create an account</Link>}
          </div>
        </motion.div>

        <motion.div className="hero-sheet" {...rise(0.1)}>
          <div className="hero-sheet-head">
            <h3>Brand identity for a coffee shop</h3>
            <span className="tag">Example contract</span>
          </div>
          <TermSheet partyA="boris_design" partyB="carol_pm" terms={exampleTerms} />
          <div className="hero-sheet-foot">
            <span>Success probability <b>48.2%</b></span>
            <span>Deadline <b>30 days</b></span>
          </div>
        </motion.div>
      </section>

      <section className="home-section">
        <h2>How a swap works</h2>
        <p className="lead">Three steps, and the money is never in one person's hands.</p>
        <div className="steps-row">
          {steps.map((s, i) => (
            <motion.div key={s.title} className="how-step"
              {...rise(0.15 + i * 0.05)}>
              <span className="how-num">{i + 1}</span>
              <h3>{s.title}</h3>
              <p>{s.text}</p>
            </motion.div>
          ))}
        </div>
      </section>

      <section className="home-section">
        <h2>Two opinions on every project</h2>
        <p className="lead">
          The chance of success decides the terms. You see how it was calculated, and a second opinion
          from a model trained on finished projects.
        </p>
        <div className="two-sources">
          <div className="source-card">
            <span className="tag tag-muted">Formula</span>
            <h3>Rules you can read</h3>
            <p>Reputation, success rate, experience, complexity, budget and deadline each add or remove
              points. The project page shows every factor.</p>
          </div>
          <div className="source-card">
            <span className="tag">ML model</span>
            <h3>Learned from data</h3>
            <p>A RandomForest model gives its own probability in the background. When the two disagree,
              you know the project needs a closer look.</p>
          </div>
        </div>
      </section>

      {!user && (
        <section className="home-section">
          <div className="home-cta">
            <div>
              <h2>Start with a project</h2>
              <p>Create an account, describe a project, and the calculator proposes the first terms.</p>
            </div>
            <Link to="/register" className="btn-primary">Create an account</Link>
          </div>
        </section>
      )}
    </div>
  );
};

export default Home;