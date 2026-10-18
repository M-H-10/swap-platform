import { useEffect, useState } from 'react';
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast';
import { LogOut, Moon, Sun } from 'lucide-react';
import { getWallet } from '../api';
import { useAuth } from '../context/auth-context';
import { useTheme } from '../context/theme-context';
import { formatRub } from '../lib/format';

/** The light / dark switch. */
export const ThemeToggle = () => {
  const { theme, toggleTheme } = useTheme();
  const next = theme === 'dark' ? 'light' : 'dark';
  return (
    <button type="button" className="icon-btn" onClick={toggleTheme}
      aria-label={`Switch to ${next} mode`} title={`Switch to ${next} mode`}>
      {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
    </button>
  );
};

const Navbar = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const [available, setAvailable] = useState<number | null>(null);

  // The balance is shown next to the name, like in a bank app. It is loaded
  // again after login and on every page change, so it is never old for long
  // (a deposit or an accepted swap changes it).
  useEffect(() => {
    if (!user) return;
    getWallet().then(({ data }) => setAvailable(data.balance)).catch(() => setAvailable(null));
  }, [user, pathname]);

  const handleLogout = async () => {
    await logout();
    toast.success('Logged out');
    navigate('/');
  };

  const navClass = ({ isActive }: { isActive: boolean }) => (isActive ? 'active' : '');

  return (
    <header className="navbar">
      <Link to="/" className="logo">
        <span className="logo-mark" aria-hidden="true" />
        <span>Swap Platform</span>
      </Link>
      <nav className="nav-links">
        <NavLink to="/projects" className={navClass} end>Projects</NavLink>
        {user && (
          <>
            <NavLink to="/dashboard" className={navClass}>My swaps</NavLink>
            <NavLink to="/wallet" className={navClass}>Wallet</NavLink>
            <NavLink to="/create-project" className={navClass}>New project</NavLink>
          </>
        )}
      </nav>
      <div className="nav-right">
        {user ? (
          <>
            {available !== null && (
              <span className="nav-balance">Available <b>{formatRub(available)}</b></span>
            )}
            <Link to={`/users/${user.id}`} className="user-badge">{user.username}</Link>
            <ThemeToggle />
            <button type="button" onClick={handleLogout} className="icon-btn" aria-label="Log out" title="Log out">
              <LogOut size={16} />
            </button>
          </>
        ) : (
          <>
            <ThemeToggle />
            <Link to="/login" className="btn-ghost">Log in</Link>
            <Link to="/register" className="btn-primary btn-sm">Create account</Link>
          </>
        )}
      </div>
    </header>
  );
};

export default Navbar;