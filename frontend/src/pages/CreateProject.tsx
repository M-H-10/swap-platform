import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { createProject } from '../api';
import { errorMessage, formatPercent, rublesToKopecks } from '../lib/format';
import type { Category, KPI } from '../types';
import toast from 'react-hot-toast';
import { Plus, Trash2 } from 'lucide-react';

// The form keeps what the user typed (text). It is turned into numbers,
// and rubles into kopecks, only when the project is sent.
interface FormState {
  title: string;
  description: string;
  category: Category;
  budget: string; // rubles, as typed
  duration: string; // days, as typed
  complexity: number;
  teamSize: number;
  kpis: KPI[];
}

const categories: Category[] = ['web', 'mobile', 'design', 'marketing', 'other'];

const CreateProject = () => {
  const [step, setStep] = useState(1);
  const [form, setForm] = useState<FormState>({
    title: '', description: '', category: 'web',
    budget: '', duration: '', complexity: 5, teamSize: 2,
    kpis: [{ name: '', target: '', weight: 100 }]
  });
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const addKPI = () => setForm({...form, kpis: [...form.kpis, { name: '', target: '', weight: 0 }]});
  const removeKPI = (i: number) => setForm({...form, kpis: form.kpis.filter((_, idx) => idx !== i)});
  const updateKPI = <K extends keyof KPI>(i: number, field: K, val: KPI[K]) => {
    const kpis = form.kpis.map((k, idx) => idx === i ? { ...k, [field]: val } : k);
    setForm({...form, kpis});
  };

  // Rows the user left empty are not sent. The server checks that the
  // weights of the remaining KPIs add up to 100.
  const filledKPIs = form.kpis.filter(k => k.name.trim() || k.target.trim());
  const weightSum = filledKPIs.reduce((sum, k) => sum + k.weight, 0);

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const { data } = await createProject({
        title: form.title,
        description: form.description,
        category: form.category,
        budget: rublesToKopecks(form.budget),
        duration_days: Number(form.duration),
        complexity: form.complexity,
        team_size: form.teamSize,
        kpis: filledKPIs,
      });
      toast.success(`Project created! Success probability: ${formatPercent(data.score.success_probability_bp)}`);
      navigate('/projects');
    } catch (err) {
      toast.error(errorMessage(err, 'Failed'));
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
              <label>Description (10+ characters)</label>
              <textarea placeholder="Describe your project..." value={form.description}
                onChange={e => setForm({...form, description: e.target.value})} />
            </div>
            <div className="form-group">
              <label>Category</label>
              <select value={form.category} onChange={e => setForm({...form, category: e.target.value as Category})}>
                {categories.map(c =>
                  <option key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</option>)}
              </select>
            </div>
            <button className="btn-primary" onClick={() => setStep(2)}
              disabled={form.title.trim().length < 3 || form.description.trim().length < 10}>Next →</button>
          </motion.div>
        )}

        {step === 2 && (
          <motion.div key="step2" className="form-step"
            initial={{ opacity: 0, x: 50 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -50 }}>
            <h2>Financial Details</h2>
            <div className="form-row">
              <div className="form-group">
                <label>Budget (₽)</label>
                <input type="number" min="100" placeholder="1000000" value={form.budget}
                  onChange={e => setForm({...form, budget: e.target.value})} />
              </div>
              <div className="form-group">
                <label>Duration (days)</label>
                <input type="number" min="1" placeholder="90" value={form.duration}
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
              <input type="range" min="1" max="20" value={form.teamSize}
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
            <p className="step-desc">Define success metrics for your swap contract (optional; the weights must add up to 100)</p>
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
            {filledKPIs.length > 0 && weightSum !== 100 && (
              <p className="step-desc form-error">Weights add up to {weightSum}, they must add up to 100.</p>
            )}
            <button className="btn-add-kpi" onClick={addKPI}><Plus size={16} /> Add KPI</button>
            <div className="btn-row">
              <button className="btn-secondary" onClick={() => setStep(2)}>← Back</button>
              <motion.button className="btn-primary" onClick={handleSubmit}
                disabled={loading || (filledKPIs.length > 0 && weightSum !== 100)}
                whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                {loading ? <span className="spinner" /> : 'Create project'}
              </motion.button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default CreateProject;