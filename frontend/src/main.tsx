import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import '@fontsource/ibm-plex-sans/400.css'
import '@fontsource/ibm-plex-sans/500.css'
import '@fontsource/ibm-plex-sans/600.css'
import App from './App'
import './index.css'
import './pages.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
      {/* The colours come from CSS variables, so the toasts follow the theme. */}
      <Toaster position="top-right" toastOptions={{
        style: {
          background: 'var(--surface)', color: 'var(--text)',
          border: '1px solid var(--border)', boxShadow: 'var(--shadow-pop)',
          fontSize: '0.9rem',
        },
      }} />
    </BrowserRouter>
  </StrictMode>
)