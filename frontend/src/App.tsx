import type { ReactNode } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { MotionConfig } from 'framer-motion';
import { AuthProvider } from './context/AuthProvider';
import { useAuth } from './context/auth-context';
import { ThemeProvider } from './context/ThemeProvider';
import Navbar from './components/Navbar';
import Home from './pages/Home';
import Login from './pages/Login';
import Register from './pages/Register';
import Projects from './pages/Projects';
import CreateProject from './pages/CreateProject';
import SwapCalculator from './pages/SwapCalculator';
import Dashboard from './pages/Dashboard';
import Wallet from './pages/Wallet';
import SwapDetail from './pages/SwapDetail';
import Profile from './pages/Profile';
import ProjectDetail from './pages/ProjectDetail';

const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const { user, ready } = useAuth();
  // Wait for the first "is there a session?" answer, otherwise a logged-in
  // user who reloads the page would be sent to /login for a moment.
  if (!ready) return <div className="loading">Loading...</div>;
  return user ? children : <Navigate to="/login" replace />;
};

const AppContent = () => (
  <>
    <Navbar />
    <main className="main-content">
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/projects" element={<Projects />} />
        <Route path="/projects/:id" element={<ProjectDetail />} />
        <Route path="/users/:id" element={<Profile />} />
        <Route path="/create-project" element={<ProtectedRoute><CreateProject /></ProtectedRoute>} />
        <Route path="/swap-calculator/:projectId" element={<ProtectedRoute><SwapCalculator /></ProtectedRoute>} />
        <Route path="/dashboard" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
        <Route path="/wallet" element={<ProtectedRoute><Wallet /></ProtectedRoute>} />
        <Route path="/swaps/:id" element={<ProtectedRoute><SwapDetail /></ProtectedRoute>} />
      </Routes>
    </main>
  </>
);

// reducedMotion="user": if the system asks for less motion, framer-motion
// skips the movement and keeps only the fade.
const App = () => (
  <ThemeProvider>
    <MotionConfig reducedMotion="user">
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </MotionConfig>
  </ThemeProvider>
);

export default App;