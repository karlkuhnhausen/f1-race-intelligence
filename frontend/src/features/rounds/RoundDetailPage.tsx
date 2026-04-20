import { useEffect, useState } from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { fetchRoundDetail, type RoundDetailResponse, type SessionDetail } from './roundApi';
import RaceResults from './RaceResults';
import SessionResultsTable from './SessionResultsTable';

export default function RoundDetailPage() {
  const { round } = useParams<{ round: string }>();
  const [searchParams] = useSearchParams();
  const year = Number(searchParams.get('year') ?? 2026);
  const roundNum = Number(round);

  const [data, setData] = useState<RoundDetailResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!roundNum || roundNum < 1) return;
    setLoading(true);
    fetchRoundDetail(roundNum, year)
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [roundNum, year]);

  if (loading) return <div className="loading">Loading round details…</div>;
  if (error) return <div className="error">Error: {error}</div>;
  if (!data) return null;

  // Order sessions: practice → qualifying → race
  const sessionOrder: Record<string, number> = {
    practice1: 1, practice2: 2, practice3: 3,
    sprint_qualifying: 4, sprint: 5,
    qualifying: 6, race: 7,
  };

  const sortedSessions = [...data.sessions].sort(
    (a, b) => (sessionOrder[a.session_type] ?? 99) - (sessionOrder[b.session_type] ?? 99)
  );

  return (
    <section className="round-detail-page">
      <Link to="/calendar" style={{ marginBottom: '1rem', display: 'inline-block' }}>
        ← Back to Calendar
      </Link>
      <h2>
        Round {data.round}: {data.race_name}
      </h2>
      <p>
        {data.circuit_name} — {data.country_name}
      </p>
      {data.data_as_of_utc && (
        <p className="data-freshness">
          Data as of: {new Date(data.data_as_of_utc).toLocaleString()}
        </p>
      )}

      {sortedSessions.length === 0 ? (
        <p>No session data available yet for this round.</p>
      ) : (
        sortedSessions.map((session) => (
          <SessionCard key={session.session_type} session={session} />
        ))
      )}
    </section>
  );
}

const RACE_TYPES = new Set(['race', 'sprint']);

function SessionCard({ session }: { session: SessionDetail }) {
  const isRace = RACE_TYPES.has(session.session_type);

  return (
    <div className="session-card" style={{ marginBottom: '1.5rem' }}>
      <h3>{session.session_name}</h3>
      <p>
        Status: <span className={`badge ${session.status}`}>{session.status}</span>
        {' | '}
        {new Date(session.date_start_utc).toLocaleString()}
      </p>
      {session.results.length > 0 ? (
        isRace ? (
          <RaceResults results={session.results} />
        ) : (
          <SessionResultsTable results={session.results} sessionType={session.session_type} />
        )
      ) : (
        <p>No results available.</p>
      )}
    </div>
  );
}
