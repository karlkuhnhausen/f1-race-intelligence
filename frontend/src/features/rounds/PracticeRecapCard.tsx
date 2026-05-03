import LapTimeDisplay from '../design-system/LapTimeDisplay';
import { getTeamColor } from '../design-system/teamColors';
import type { SessionDetail } from './roundApi';

export default function PracticeRecapCard({ session }: { session: SessionDetail }) {
  const recap = session.recap_summary!;
  const driverColor = getTeamColor(recap.best_driver_team ?? '');

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-surface p-4 min-w-[260px] w-full md:w-[280px] shrink-0">
      <p className="font-display text-xs uppercase tracking-wider text-muted-foreground">
        {session.session_name}
      </p>

      {recap.best_driver_name && (
        <div
          className="flex items-center gap-2 border-l-4 pl-2"
          style={{ borderColor: driverColor }}
        >
          <div>
            <p className="font-display font-bold leading-tight">{recap.best_driver_name}</p>
            <p className="text-xs text-muted-foreground">{recap.best_driver_team}</p>
          </div>
        </div>
      )}

      {recap.best_lap_time != null && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Best Lap</span>
          <LapTimeDisplay time={recap.best_lap_time} className="font-mono" />
        </div>
      )}

      {(recap.total_laps ?? 0) > 0 && (
        <p className="text-sm text-muted-foreground font-mono">{recap.total_laps} laps</p>
      )}

      {(recap.red_flag_count ?? 0) > 0 && (
        <p className="text-sm text-muted-foreground">
          🚩 Red Flag{(recap.red_flag_count ?? 0) > 1 ? ` ×${recap.red_flag_count}` : ''}
        </p>
      )}
    </div>
  );
}
