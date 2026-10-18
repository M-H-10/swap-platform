import { useState, useEffect, type CSSProperties } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { getProjects } from '../api';
import { useAuth } from '../context/auth-context';
import { formatPercent, formatRub, probabilityColor } from '../lib/format';
import type { Project } from '../types';
import { TrendingUp, Clock, Users, DollarSign } from 'lucide-react';
import toast from 'react-hot-toast';

const Projects = () => {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const { user } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    getProjects() // open projects, newest first
      .then(({ data }) => setProjects(data.items))
      .catch(() => toast.error('Failed to load'))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="projects-page">
      <motion.div className="page-header"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}>
        <h1>Open projects</h1>
        <p>Find projects to create a reputation swap contract</p>
        {user && (
          <Link to="/create-project" className="btn-primary">New project</Link>
        )}
      </motion.div>

      {loading ? (
        <div className="loading-grid">
          {[1,2,3].map(i => <div key={i} className="skeleton-card" />)}
        </div>
      ) : projects.length === 0 ? (
        <div className="empty">No open projects yet. Create the first one!</div>
      ) : (
        <div className="projects-grid">
          {projects.map((p, i) => (
            <motion.div key={p.id} className="project-card"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: Math.min(i, 8) * 0.04, duration: 0.3 }}>
              <div className="card-header">
                <span className="category-badge">{p.category}</span>
                <div className="prob-circle" style={{ '--color': probabilityColor(p.success_probability_bp) } as CSSProperties}>
                  <span>{formatPercent(p.success_probability_bp)}</span>
                  <small>success</small>
                </div>
              </div>
                            <h3><Link to={`/projects/${p.id}`} className="card-title-link">{p.title}</Link></h3>
              <p>{p.description.length > 100 ? p.description.substring(0, 100) + '...' : p.description}</p>
              <div className="card-stats">
                <span><DollarSign size={14} /> {formatRub(p.budget)}</span>
                <span><Clock size={14} /> {p.duration_days} days</span>
                <span><Users size={14} /> {p.team_size} {p.team_size === 1 ? 'member' : 'members'}</span>
                <span><TrendingUp size={14} /> complexity {p.complexity}/10</span>
              </div>
              <div className="card-footer">
                <span className="author">by {p.creator.username}</span>
                {/* Only the creator offers a swap on a project (party A). */}
                {user && user.id === p.creator.id && (
                  <button
                    className="btn-swap"
                    onClick={() => navigate(`/swap-calculator/${p.id}`)}>
                    Find a partner
                  </button>
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