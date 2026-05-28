import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { createProject } from '../api';
import toast from 'react-hot-toast';
import { Plus, Trash2 } from 'lucide-react';

const CreateProject = () => {
  const [step, setStep] = useState(1);
  const [form, setForm] = useState({
    title: '', description: '', category: 'web',
    budget: '', duration: '', complexity: 5, teamSize: 2,
    kpis: [{ name: '', target: '', weight: 100 }]
  });
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const addKPI = () => setForm({...form, kpis: [...form.kpis, { name: '', target: '', weight: 0 }]});
  const removeKPI = (i) => setForm({...form, kpis: form.kpis.filter((_, idx) => idx !== i)});
  const updateKPI = (i, field, val) => {
    const kpis = [...form.kpis];
    kpis[i][field] = val;
    setForm({...form, kpis});
  };

  const handleSubmit = async () => {
    setLoading(true);
    try {
      await createProject({ ...form, budget: Number(form.budget), duration: Number(form.duration) });
      toast.success('Project created!');
      navigate('/projects');
    } catch (err) {
      toast.error(err.response?.data?.message || 'Failed');
    }
    setLoading(false);
  };

  return (
    <div className="create-project">
      <div className="steps-indicator">
        {[1, 2, 3].map(s => (
          <div key={s} className={`step ${step >= s ? 'active' : ''} ${step > s ? 'done' : ''}`}>
            <span>{step > s ? '✓' : s}</span>
            <small>{s === 1 ? 'Basics' : s === 2 ? 'Details' : 'KPIs'}</small>
          </div>
        ))}
      </div>

      <AnimatePresence mode="wait">
        {step === 1 && (
          <motion.div key="step1" className="form-step"
            initial={{ opacity: 0, x: 50 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -50 }}>
            <h2>Project Basics</h2>
            <div className="form-group">
              <label>Project Title</label>
              <input placeholder="My Awesome Project" value={form.title}
                onChange={e => setForm({...form, title: e.target.value})} />
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea placeholder="Describe your project..." value={form.description}
                onChange={e => setForm({...form, description: e.target.value})} />
            </div>
            <div className="form-group">
              <label>Category</label>
              <select value={form.category} onChange={e => setForm({...form, category: e.target.value})}>
                {['web','mobile','design','marketing','other'].map(c =>
                  <option key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</option>)}
              </select>
            </div>
            <button className="btn-primary" onClick={() => setStep(2)}
              disabled={!form.title || !form.description}>Next →</button>
          </motion.div>
        )}

        {step === 2 && (
          <motion.div key="step2" className="form-step"
            initial={{ opacity: 0, x: 50 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -50 }}>
            <h2>Financial Details</h2>
            <div className="form-row">
              <div className="form-group">
                <label>Budget (₽)</label>
                <input type="number" placeholder="1000000" value={form.budget}
                  onChange={e => setForm({...form, budget: e.target.value})} />
              </div>
              <div className="form-group">
                <label>Duration (days)</label>
                <input type="number" placeholder="90" value={form.duration}
                  onChange={e => setForm({...form, duration: e.target.value})} />
              </div>
            </div>
            <div className="form-group">
              <label>Complexity: {form.complexity}/10</label>
              <input type="range" min="1" max="10" value={form.complexity}
                onChange={e => setForm({...form, complexity: Number(e.target.value)})} />
              <div className="range-labels"><span>Simple</span><span>Complex</span></div>
            </div>
            <div className="form-group">
              <label>Team Size: {form.teamSize}</label>
              <input type="range" min="2" max="20" value={form.teamSize}
                onChange={e => setForm({...form, teamSize: Number(e.target.value)})} />
            </div>
            <div className="btn-row">
              <button className="btn-secondary" onClick={() => setStep(1)}>← Back</button>
              <button className="btn-primary" onClick={() => setStep(3)}
                disabled={!form.budget || !form.duration}>Next →</button>
            </div>
          </motion.div>
        )}

        {step === 3 && (
          <motion.div key="step3" className="form-step"
            initial={{ opacity: 0, x: 50 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -50 }}>
            <h2>KPI Definition</h2>
            <p className="step-desc">Define success metrics for your swap contract</p>
            {form.kpis.map((kpi, i) => (
              <div key={i} className="kpi-row">
                <input placeholder="KPI Name (e.g. Revenue)" value={kpi.name}
                  onChange={e => updateKPI(i, 'name', e.target.value)} />
                <input placeholder="Target (e.g. ₽1,000,000)" value={kpi.target}
                  onChange={e => updateKPI(i, 'target', e.target.value)} />
                <input type="number" placeholder="Weight %" value={kpi.weight}
                  onChange={e => updateKPI(i, 'weight', Number(e.target.value))} />
                {form.kpis.length > 1 &&
                  <button className="btn-remove" onClick={() => removeKPI(i)}><Trash2 size={16} /></button>}
              </div>
            ))}
            <button className="btn-add-kpi" onClick={addKPI}><Plus size={16} /> Add KPI</button>
            <div className="btn-row">
              <button className="btn-secondary" onClick={() => setStep(2)}>← Back</button>
              <motion.button className="btn-primary" onClick={handleSubmit}
                disabled={loading} whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                {loading ? <span className="spinner" /> : '🚀 Create Project'}
              </motion.button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default CreateProject;