import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { TrendingUp, Shield, Zap, Users } from 'lucide-react';

const features = [
  { icon: <TrendingUp size={32} />, title: 'Smart Swap Calculation', desc: 'AI-powered algorithm calculates fair swap ratios based on reputation and KPIs' },
  { icon: <Shield size={32} />, title: 'Escrow Protection', desc: 'Automatic escrow account secures both parties during the project lifecycle' },
  { icon: <Zap size={32} />, title: 'Risk Visualization', desc: 'Real-time risk profile charts for transparent decision making' },
  { icon: <Users size={32} />, title: 'Reputation System', desc: 'Historical performance data drives fair contract terms automatically' },
];

const Home = () => (
  <div className="home">
    <div className="hero">
      <motion.div
        className="hero-content"
        initial={{ opacity: 0, y: 60 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.8, ease: 'easeOut' }}
      >
        <motion.div
          className="hero-badge"
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ delay: 0.3, type: 'spring' }}
        >
          🚀 New Financial Instrument
        </motion.div>
        <h1>Reputation <span className="gradient-text">Swap</span> Platform</h1>
        <p>The world's first automated clearing center for reputation-based derivative contracts between freelancers and startups</p>
        <div className="hero-buttons">
          <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
            <Link to="/projects" className="btn-primary">Explore Projects</Link>
          </motion.div>
          <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
            <Link to="/register" className="btn-secondary">Get Started Free</Link>
          </motion.div>
        </div>
      </motion.div>

      <motion.div
        className="hero-stats"
        initial={{ opacity: 0, x: 60 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ duration: 0.8, delay: 0.3 }}
      >
        {[
          { label: 'Active Swaps', value: '127' },
          { label: 'Total Volume', value: '₽2.4M' },
          { label: 'Success Rate', value: '78%' },
        ].map((stat, i) => (
          <motion.div key={i} className="stat-card"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5 + i * 0.1 }}
          >
            <span className="stat-value">{stat.value}</span>
            <span className="stat-label">{stat.label}</span>
          </motion.div>
        ))}
      </motion.div>
    </div>

    <div className="features">
      <motion.h2
        initial={{ opacity: 0 }}
        whileInView={{ opacity: 1 }}
        viewport={{ once: true }}
      >
        How It Works
      </motion.h2>
      <div className="features-grid">
        {features.map((f, i) => (
          <motion.div
            key={i}
            className="feature-card"
            initial={{ opacity: 0, y: 40 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ delay: i * 0.15 }}
            whileHover={{ y: -8, boxShadow: '0 20px 40px rgba(108,99,255,0.3)' }}
          >
            <div className="feature-icon">{f.icon}</div>
            <h3>{f.title}</h3>
            <p>{f.desc}</p>
          </motion.div>
        ))}
      </div>
    </div>
  </div>
);

export default Home;