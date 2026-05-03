import LapTimeDisplay from '../design-system/LapTimeDisplay';
import { getTeamColor } from '../design-system/teamColors';
import type { SessionDetail } from './roundApi';

const eventLabel: Record<string, string> = {
  red_flag: '🚩 Red Flag',
  safety_car: '🚗 Safety Car',
  vsc: '🟡 Virtual SC',
  investigation: '🔍 Investigation',
};

export default function RaceRecapCard({ session }: { session: SessionDetail }) {
  const recap = session.recap_summary!;
  const winnerColor = getTeamColor(recap.winner_team ?? '');

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-surface p-4 w-full sm:w-[280px]">
      <p className="font-display text-xs uppercase tracking-wider text-muted-foreground">
        {session.session_name}
      </p>

      {recap.winner_name && (
        <div
          className="flex items-center gap-2 border-l-4 pl-2"
          style={{ borderColor: winnerColor }}
        >
          <div>
            <p className="font-display font-bold leading-tight">{recap.winner_name}</p>
            <p className="text-xs text-muted-foreground">{recap.winner_team}</p>
          </div>
        </div>
      )}

      {recap.gap_to_p2 && (
        <p className="text-sm font-mono text-muted-foreground">
          Gap to P2: <span className="text-foreground">{recap.gap_to_p2}</span>
        </p>
      )}

      {recap.fastest_lap_holder && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Fastest Lap</span>
          <span className="font-mono">
            {recap.fastest_lap_holder}
            {recap.fastest_lap_time_seconds != null && (
              <>
                {' '}
                <LapTimeDisplay time={recap.fastest_lap_time_seconds} className="text-accent-cyan" />
              </>
            )}
          </span>
        </div>
      )}

      {(recap.total_laps ?? 0) > 0 && (
        <p className="text-sm text-muted-foreground font-mono">{recap.total_laps} laps</p>
      )}

      {recap.top_event && (
        <p className="text-sm text-muted-foreground">
          {eventLabel[recap.top_event.event_type] ?? recap.top_event.event_type}
          {recap.top_event.count > 1 && ` ×${recap.top_event.count}`}
          {recap.top_event.lap_number > 0 && ` — Lap ${recap.top_event.lap_number}`}
        </p>
      )}
    </div>
  );
}
