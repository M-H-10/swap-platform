import { useEffect, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { isAxiosError } from 'axios';
import { motion } from 'framer-motion';
import toast from 'react-hot-toast';
import { deposit, getTransactions, getWallet } from '../api';
import { errorMessage, formatDate, formatRub, rublesToKopecks } from '../lib/format';
import type { Transaction, TransactionType, Wallet as WalletData } from '../types';

const PAGE_SIZE = 20;

const loadFirstPage = () => Promise.all([getWallet(), getTransactions(PAGE_SIZE, 0)]);

// How each kind of ledger line changes the AVAILABLE balance:
// +1 adds money, -1 takes it away, 0 only moves money inside the
// frozen part (it is already counted in "frozen").
const typeInfo: Record<TransactionType, { label: string; sign: 1 | -1 | 0 }> = {
  deposit: { label: 'Deposit', sign: 1 },
  escrow_lock: { label: 'Guarantee locked', sign: -1 },
  escrow_release: { label: 'Guarantee returned', sign: 1 },
  penalty_paid: { label: 'Penalty paid', sign: 0 },
  penalty_received: { label: 'Penalty received', sign: 1 },
  fee: { label: 'Platform fee', sign: 0 },
};

const amountClass = { 1: 'amount-plus', [-1]: 'amount-minus', 0: 'amount-neutral' } as const;
const amountSign = { 1: '+', [-1]: '−', 0: '' } as const;

const Wallet = () => {
  const [wallet, setWallet] = useState<WalletData | null>(null);
  const [items, setItems] = useState<Transaction[]>([]);
  const [total, setTotal] = useState(0);
  const [amount, setAmount] = useState('100000');
  const [loading, setLoading] = useState(true);
  const [depositing, setDepositing] = useState(false);
  const [depositAllowed, setDepositAllowed] = useState(true);

  // Shows the wallet and the first page of the history.
  const show = (w: WalletData, page: { items: Transaction[]; total: number }) => {
    setWallet(w);
    setItems(page.items);
    setTotal(page.total);
  };

  useEffect(() => {
    loadFirstPage()
      .then(([w, t]) => show(w.data, t.data))
      .catch((err) => toast.error(errorMessage(err, 'Failed to load the wallet')))
      .finally(() => setLoading(false));
  }, []);

  const loadMore = async () => {
    try {
      const { data } = await getTransactions(PAGE_SIZE, items.length);
      setItems((list) => [...list, ...data.items]);
      setTotal(data.total);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const handleDeposit = async (e: FormEvent) => {
    e.preventDefault();
    const kopecks = rublesToKopecks(amount);
    if (!(kopecks > 0)) { toast.error('Enter an amount in rubles'); return; }
    setDepositing(true);
    try {
      await deposit(kopecks);
      toast.success(`${formatRub(kopecks)} added`);
      const [w, t] = await loadFirstPage(); // the new line appears at the top
      show(w.data, t.data);
    } catch (err) {
      // 404 = the route does not exist: the server runs in production mode,
      // where free test money is switched off.
      if (isAxiosError(err) && err.response?.status === 404) {
        setDepositAllowed(false);
      } else {
        toast.error(errorMessage(err, 'Deposit failed'));
      }
    }
    setDepositing(false);
  };

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div className="page">
      <div className="page-title">
        <h1>Wallet</h1>
      </div>

      <div className="grid-2">
        <motion.div className="panel" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }}>
          <div className="big-number">{formatRub(wallet?.balance ?? 0)}</div>
          <div className="big-label">Available: can be used as a guarantee</div>
          <div className="big-number" style={{ marginTop: '1.2rem', color: 'var(--warning)' }}>
            {formatRub(wallet?.frozen_balance ?? 0)}
          </div>
          <div className="big-label">Frozen: guarantees of your active swaps</div>
        </motion.div>

        <motion.div className="panel" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}>
          <h3>Add test money</h3>
          {depositAllowed ? (
            <form className="inline-form" onSubmit={handleDeposit}>
              <div className="form-group">
                <label>Amount (₽, up to 1,000,000)</label>
                <input type="number" min="1" max="1000000" step="0.01" value={amount}
                  onChange={(e) => setAmount(e.target.value)} />
              </div>
              <button type="submit" className="btn-primary" disabled={depositing}>
                {depositing ? <span className="spinner" /> : 'Deposit'}
              </button>
            </form>
          ) : (
            <p className="note note-warning">Test deposits are switched off on this server.</p>
          )}
          <p className="muted" style={{ marginTop: '1rem' }}>
            This is demo money for testing swaps. There are no real payments.
          </p>
        </motion.div>
      </div>

      <div className="panel">
        <h3>History ({total})</h3>
        {items.length === 0 ? (
          <div className="empty">No transactions yet. Add test money above.</div>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Type</th>
                  <th>Description</th>
                  <th className="num">Amount</th>
                  <th className="num">Available after</th>
                  <th className="num">Frozen after</th>
                </tr>
              </thead>
              <tbody>
                {items.map((t) => {
                  const info = typeInfo[t.type];
                  return (
                    <tr key={t.id}>
                      <td className="muted">{formatDate(t.created_at)}</td>
                      <td>{info.label}</td>
                      <td>
                        {t.swap_id
                          ? <Link className="link" to={`/swaps/${t.swap_id}`}>{t.description}</Link>
                          : t.description}
                      </td>
                      <td className={`num ${amountClass[info.sign]}`}>
                        {amountSign[info.sign]}{formatRub(t.amount)}
                      </td>
                      <td className="num">{formatRub(t.balance_after)}</td>
                      <td className="num muted">{formatRub(t.frozen_after)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        {items.length < total && (
          <div className="actions">
            <button className="btn-secondary" onClick={loadMore}>Load more</button>
          </div>
        )}
      </div>
    </div>
  );
};

export default Wallet;