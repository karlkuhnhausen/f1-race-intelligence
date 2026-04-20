import { useState } from 'react';
import CalendarPage from './features/calendar/CalendarPage';
import StandingsPage from './features/standings/StandingsPage';

type Page = 'calendar' | 'standings';

export default function App() {
  const [page, setPage] = useState<Page>('calendar');

  return (
    <main>
      <h1>F1 Race Intelligence Dashboard</h1>
      <nav style={{ marginBottom: '1rem' }}>
        <button
          onClick={() => setPage('calendar')}
          style={{ fontWeight: page === 'calendar' ? 'bold' : 'normal', marginRight: '0.5rem' }}
        >
          Calendar
        </button>
        <button
          onClick={() => setPage('standings')}
          style={{ fontWeight: page === 'standings' ? 'bold' : 'normal' }}
        >
          Standings
        </button>
      </nav>
      {page === 'calendar' ? <CalendarPage /> : <StandingsPage />}
    </main>
  );
}
