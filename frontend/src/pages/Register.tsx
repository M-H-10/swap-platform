import { useState, type FormEvent } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useAuth } from '../context/auth-context';
import { errorMessage } from '../lib/format';
import type { RegisterInput, Role } from '../types';
import toast from 'react-hot-toast';

const textFields = [
  { name: 'username', type: 'text', label: 'Username', placeholder: 'john_doe' },
  { name: 'email', type: 'email', label: 'Email', placeholder: 'your@email.com' },
  { name: 'password', type: 'password', label: 'Password (8+ characters)', placeholder: '••••••••' },
] as const;

const Register = () => {
  const [form, setForm] = useState<RegisterInput>({ username: '', email: '', password: '', role: 'developer' });
  const [loading, setLoading] = useState(false);
  const { register } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const user = await register(form);
      toast.success(`Welcome, ${user.username}!`);
      navigate('/dashboard');
    } catch (err) {
      toast.error(errorMessage(err, 'Registration failed'));
    }
    setLoading(false);
  };

  return (
    <div className="auth-container">
      <motion.div className="auth-box"
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.23, 1, 0.32, 1] }}>
        <div className="auth-header">
          <h2>Create an account</h2>
          <p>Create your account and start trading reputation</p>
        </div>
        <form onSubmit={handleSubmit}>
          {textFields.map(f => (
            <div key={f.name} className="form-group">
              <label>{f.label}</label>
              <input type={f.type} placeholder={f.placeholder}
                value={form[f.name]}
                onChange={(e) => setForm({...form, [f.name]: e.target.value})} required />
            </div>
          ))}
          <div className="form-group">
            <label>Your Role</label>
            <select value={form.role} onChange={(e) => setForm({...form, role: e.target.value as Role})}>
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