import { useEffect, useState } from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { fetchRoundDetail, type RoundDetailResponse, type SessionDetail } from './roundApi';
import RaceResults from './RaceResults';
import QualifyingResults from './QualifyingResults';
import PracticeResults from './PracticeResults';

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

  if (loading) return <div className="text-muted-foreground">Loading round details…</div>;
  if (error) return <div className="text-negative">Error: {error}</div>;
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
    <section className="space-y-6">
      <Link
        to="/calendar"
        className="inline-block font-display text-sm uppercase tracking-wider text-accent-cyan hover:text-foreground transition-colors"
      >
        ← Back to Calendar
      </Link>
      <div>
        <p className="font-display text-xs uppercase tracking-[0.2em] text-accent-red">
          Round {data.round}
        </p>
        <h2 className="mt-1 font-display text-3xl font-bold tracking-tight">
          {data.race_name}
        </h2>
        <p className="text-muted-foreground">
          {data.circuit_name} — {data.country_name}
        </p>
        {data.data_as_of_utc && (
          <p className="mt-1 text-sm text-muted-foreground font-mono">
            Data as of: {new Date(data.data_as_of_utc).toLocaleString()}
          </p>
        )}
      </div>

      {sortedSessions.length === 0 ? (
        <p className="text-muted-foreground">No session data available yet for this round.</p>
      ) : (
        <div className="space-y-4">
          {sortedSessions.map((session) => (
            <SessionCard key={session.session_type} session={session} />
          ))}
        </div>
      )}
    </section>
  );
}

const RACE_TYPES = new Set(['race', 'sprint']);
const QUALIFYING_TYPES = new Set(['qualifying', 'sprint_qualifying']);
const PRACTICE_TYPES = new Set(['practice1', 'practice2', 'practice3']);

function SessionCard({ session }: { session: SessionDetail }) {
  const statusColor =
    session.status === 'completed'
      ? 'bg-positive/20 text-positive'
      : session.status === 'live'
        ? 'bg-accent-red/20 text-accent-red'
        : 'bg-surface text-muted-foreground border border-border';

  return (
    <div className="rounded-lg border border-border bg-surface p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="font-display text-xl font-bold tracking-tight">{session.session_name}</h3>
        <span className="text-sm text-muted-foreground font-mono">
          {new Date(session.date_start_utc).toLocaleString()}
        </span>
      </div>
      <p className="mt-2">
        <span
          className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-display font-bold uppercase tracking-wider ${statusColor}`}
        >
          {session.status}
        </span>
      </p>
      <div className="mt-4">
        {session.status === 'upcoming' ? (
          <p className="text-muted-foreground">Not yet available</p>
        ) : session.results.length > 0 ? (
          RACE_TYPES.has(session.session_type) ? (
            <RaceResults results={session.results} />
          ) : QUALIFYING_TYPES.has(session.session_type) ? (
            <QualifyingResults results={session.results} />
          ) : PRACTICE_TYPES.has(session.session_type) ? (
            <PracticeResults results={session.results} />
          ) : (
            <p className="text-muted-foreground">No results available.</p>
          )
        ) : (
          <p className="text-muted-foreground">No results available.</p>
        )}
      </div>
    </div>
  );
}
