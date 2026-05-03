import type { SessionDetail } from './roundApi';
import RaceRecapCard from './RaceRecapCard';
import QualifyingRecapCard from './QualifyingRecapCard';
import PracticeRecapCard from './PracticeRecapCard';

interface SessionRecapStripProps {
  sessions: SessionDetail[];
}

export default function SessionRecapStrip({ sessions }: SessionRecapStripProps) {
  const completedWithRecap = [...sessions]
    .filter((s) => s.status === 'completed' && s.recap_summary != null)
    .sort(
      (a, b) =>
        new Date(b.date_start_utc).getTime() - new Date(a.date_start_utc).getTime(),
    );

  if (completedWithRecap.length === 0) return null;

  return (
    <div
      className="flex flex-wrap gap-3"
      aria-label="Session recap"
    >
      {completedWithRecap.map((session) => {
        const type = session.session_type;
        if (type === 'race' || type === 'sprint') {
          return <RaceRecapCard key={session.session_type} session={session} />;
        }
        if (type === 'qualifying' || type === 'sprint_qualifying') {
          return <QualifyingRecapCard key={session.session_type} session={session} />;
        }
        // practice1, practice2, practice3
        return <PracticeRecapCard key={session.session_type} session={session} />;
      })}
    </div>
  );
}
