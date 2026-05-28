import { Link, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { motion } from 'framer-motion';
import toast from 'react-hot-toast';
import { BarChart2, Home, FolderOpen, PlusCircle, LogOut, User } from 'lucide-react';

const Navbar = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = () => {
    logout();
    toast.success('Logged out');
    navigate('/');
  };

  const isActive = (path) => location.pathname === path;

  return (
    <motion.nav
      className="navbar"
      initial={{ y: -100 }}
      animate={{ y: 0 }}
      transition={{ type: 'spring', stiffness: 100 }}
    >
      <Link to="/" className="logo">
        <span className="logo-icon">⚡</span>
        <span>SwapPlatform</span>
      </Link>
      <div className="nav-links">
        <Link to="/" className={isActive('/') ? 'active' : ''}><Home size={16} /> Home</Link>
        <Link to="/projects" className={isActive('/projects') ? 'active' : ''}><FolderOpen size={16} /> Projects</Link>
        {user ? (
          <>
            <Link to="/create-project" className={isActive('/create-project') ? 'active' : ''}><PlusCircle size={16} /> New Project</Link>
            <Link to="/dashboard" className={isActive('/dashboard') ? 'active' : ''}><BarChart2 size={16} /> Dashboard</Link>
            <span className="user-badge"><User size={14} /> {user.username}</span>
            <button onClick={handleLogout} className="logout-btn"><LogOut size={14} /> Logout</button>
          </>
        ) : (
          <>
            <Link to="/login">Login</Link>
            <Link to="/register" className="register-btn">Register</Link>
          </>
        )}
      </div>
    </motion.nav>
  );
};

export default Navbar;