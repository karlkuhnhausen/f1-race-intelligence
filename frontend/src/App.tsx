import { NavLink, Routes, Route, Navigate } from 'react-router-dom';
import CalendarPage from './features/calendar/CalendarPage';
import StandingsPage from './features/standings/StandingsPage';
import RoundDetailPage from './features/rounds/RoundDetailPage';

export default function App() {
  return (
    <main className="min-h-screen bg-background text-foreground font-body">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto max-w-6xl px-6 py-4">
          <h1 className="font-display text-2xl font-bold tracking-tight">
            <span className="text-accent-red">F1</span> Race Intelligence
          </h1>
          <nav className="mt-3 flex gap-6 text-sm uppercase tracking-wider">
            <NavLink
              to="/calendar"
              className={({ isActive }) =>
                `font-display transition-colors ${
                  isActive
                    ? 'text-accent-cyan'
                    : 'text-muted-foreground hover:text-foreground'
                }`
              }
            >
              Calendar
            </NavLink>
            <NavLink
              to="/standings"
              className={({ isActive }) =>
                `font-display transition-colors ${
                  isActive
                    ? 'text-accent-cyan'
                    : 'text-muted-foreground hover:text-foreground'
                }`
              }
            >
              Standings
            </NavLink>
          </nav>
        </div>
      </header>
      <div className="mx-auto max-w-6xl px-6 py-6">
        <Routes>
          <Route path="/calendar" element={<CalendarPage />} />
          <Route path="/standings" element={<StandingsPage />} />
          <Route path="/rounds/:round" element={<RoundDetailPage />} />
          <Route path="*" element={<Navigate to="/calendar" replace />} />
        </Routes>
      </div>
    </main>
  );
}
