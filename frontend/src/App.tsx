import { NavLink, Routes, Route, Navigate } from 'react-router-dom';
import CalendarPage from './features/calendar/CalendarPage';
import StandingsPage from './features/standings/StandingsPage';
import RoundDetailPage from './features/rounds/RoundDetailPage';

export default function App() {
  return (
    <main>
      <h1>F1 Race Intelligence Dashboard</h1>
      <nav style={{ marginBottom: '1rem' }}>
        <NavLink
          to="/calendar"
          style={({ isActive }) => ({ fontWeight: isActive ? 'bold' : 'normal', marginRight: '0.5rem' })}
        >
          Calendar
        </NavLink>
        <NavLink
          to="/standings"
          style={({ isActive }) => ({ fontWeight: isActive ? 'bold' : 'normal' })}
        >
          Standings
        </NavLink>
      </nav>
      <Routes>
        <Route path="/calendar" element={<CalendarPage />} />
        <Route path="/standings" element={<StandingsPage />} />
        <Route path="/rounds/:round" element={<RoundDetailPage />} />
        <Route path="*" element={<Navigate to="/calendar" replace />} />
      </Routes>
    </main>
  );
}
