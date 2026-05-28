import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { getProjects } from '../api';
import { useAuth } from '../context/AuthContext';
import { TrendingUp, Clock, Users, DollarSign } from 'lucide-react';
import toast from 'react-hot-toast';

const Projects = () => {
  const [projects, setProjects] = useState([]);
  const [loading, setLoading] = useState(true);
  const { user } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    getProjects().then(({ data }) => { setProjects(data); setLoading(false); })
      .catch(() => { toast.error('Failed to load'); setLoading(false); });
  }, []);

  const getProbColor = (prob) => {
    if (prob >= 70) return '#00d4aa';
    if (prob >= 40) return '#f59e0b';
    return '#e94560';
  };

  return (
    <div className="projects-page">
      <motion.div className="page-header"
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}>
        <h1>Active Projects</h1>
        <p>Find projects to create a reputation swap contract</p>
        {user && (
          <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
            <Link to="/create-project" className="btn-primary">+ New Project</Link>
          </motion.div>
        )}
      </motion.div>

      {loading ? (
        <div className="loading-grid">
          {[1,2,3].map(i => <div key={i} className="skeleton-card" />)}
        </div>
      ) : (
        <div className="projects-grid">
          {projects.map((p, i) => (
            <motion.div key={p._id} className="project-card"
              initial={{ opacity: 0, y: 40 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.1 }}
              whileHover={{ y: -6 }}>
              <div className="card-header">
                <span className="category-badge">{p.category}</span>
                <div className="prob-circle" style={{ '--color': getProbColor(p.successProbability) }}>
                  <span>{p.successProbability}%</span>
                  <small>success</small>
                </div>
              </div>
              <h3>{p.title}</h3>
              <p>{p.description.substring(0, 100)}...</p>
              <div className="card-stats">
                <span><DollarSign size={14} /> ₽{p.budget.toLocaleString()}</span>
                <span><Clock size={14} /> {p.duration} days</span>
                <span><Users size={14} /> {p.teamSize} members</span>
                <span><TrendingUp size={14} /> complexity {p.complexity}/10</span>
              </div>
              <div className="card-footer">
                <span className="author">by {p.creator?.username}</span>
                {user && user.userId !== p.creator?._id && (
                  <motion.button
                    className="btn-swap"
                    whileHover={{ scale: 1.05 }}
                    onClick={() => navigate(`/swap-calculator/${p._id}`)}>
                    Calculate Swap ⚡
                  </motion.button>
                )}
              </div>
            </motion.div>
          ))}
        </div>
      )}
    </div>
  );
};

export default Projects;