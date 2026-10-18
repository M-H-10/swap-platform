# Swap Platform: frontend

React 19 + TypeScript + Vite. It talks to the Go API through the `/api`
proxy in `vite.config.ts`.

```bash
npm install
npm run dev        # http://localhost:5173 (the Go server must run on :5002)
npm run typecheck  # TypeScript
npm run lint       # ESLint
npm test           # Vitest
npm run build      # production build in dist/
```

See the main [README](../README.md) for the whole project.