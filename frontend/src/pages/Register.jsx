import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { register as registerAPI } from '../api';
import { useAuth } from '../context/AuthContext';
import toast from 'react-hot-toast';

const Register = () => {
  const [form, setForm] = useState({ username: '', email: '', password: '', role: 'developer' });
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      const { data } = await registerAPI(form);
      login({ username: data.username, userId: data.userId, role: data.role }, data.token);
      toast.success(`Welcome, ${data.username}!`);
      navigate('/dashboard');
    } catch (err) {
      toast.error(err.response?.data?.message || 'Registration failed');
    }
    setLoading(false);
  };

  return (
    <div className="auth-container">
      <motion.div className="auth-box"
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5 }}>
        <div className="auth-header">
          <div className="auth-icon">🚀</div>
          <h2>Join SwapPlatform</h2>
          <p>Create your account and start trading reputation</p>
        </div>
        <form onSubmit={handleSubmit}>
          {['username', 'email', 'password'].map(field => (
            <div key={field} className="form-group">
              <label>{field.charAt(0).toUpperCase() + field.slice(1)}</label>
              <input type={field === 'password' ? 'password' : field === 'email' ? 'email' : 'text'}
                placeholder={field === 'username' ? 'John Doe' : field === 'email' ? 'your@email.com' : '••••••••'}
                value={form[field]}
                onChange={(e) => setForm({...form, [field]: e.target.value})} required />
            </div>
          ))}
          <div className="form-group">
            <label>Your Role</label>
            <select value={form.role} onChange={(e) => setForm({...form, role: e.target.value})}>
              <option value="developer">Developer</option>
              <option value="designer">Designer</option>
              <option value="manager">Manager</option>
              <option value="other">Other</option>
            </select>
          </div>
          <motion.button type="submit" disabled={loading} className="btn-submit"
            whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
            {loading ? <span className="spinner" /> : 'Create Account'}
          </motion.button>
        </form>
        <p className="auth-link">Already have an account? <Link to="/login">Login</Link></p>
      </motion.div>
    </div>
  );
};

export default Register;