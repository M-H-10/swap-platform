import type { ReactNode } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './context/AuthProvider';
import { useAuth } from './context/auth-context';
import Navbar from './components/Navbar';
import ParticleBackground from './components/ParticleBackground';
import Home from './pages/Home';
import Login from './pages/Login';
import Register from './pages/Register';
import Projects from './pages/Projects';
import CreateProject from './pages/CreateProject';
import SwapCalculator from './pages/SwapCalculator';
import Dashboard from './pages/Dashboard';

const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const { user, ready } = useAuth();
  // Wait for the first "is there a session?" answer, otherwise a logged-in
  // user who reloads the page would be sent to /login for a moment.
  if (!ready) return <div className="loading">Loading...</div>;
  return user ? children : <Navigate to="/login" replace />;
};

const AppContent = () => (
  <>
    <ParticleBackground />
    <Navbar />
    <main className="main-content">
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/projects" element={<Projects />} />
        <Route path="/create-project" element={<ProtectedRoute><CreateProject /></ProtectedRoute>} />
        <Route path="/swap-calculator/:projectId" element={<ProtectedRoute><SwapCalculator /></ProtectedRoute>} />
        <Route path="/dashboard" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
      </Routes>
    </main>
  </>
);

const App = () => (
  <AuthProvider>
    <AppContent />
  </AuthProvider>
);

export default App;