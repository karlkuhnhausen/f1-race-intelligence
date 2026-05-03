import LapTimeDisplay from '../design-system/LapTimeDisplay';
import { getTeamColor } from '../design-system/teamColors';
import type { SessionDetail } from './roundApi';

export default function QualifyingRecapCard({ session }: { session: SessionDetail }) {
  const recap = session.recap_summary!;
  const poleColor = getTeamColor(recap.pole_sitter_team ?? '');

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-surface p-4 min-w-[260px] w-full md:w-[280px] shrink-0">
      <p className="font-display text-xs uppercase tracking-wider text-muted-foreground">
        {session.session_name}
      </p>

      {recap.pole_sitter_name && (
        <div
          className="flex items-center gap-2 border-l-4 pl-2"
          style={{ borderColor: poleColor }}
        >
          <div>
            <p className="font-display font-bold leading-tight">{recap.pole_sitter_name}</p>
            <p className="text-xs text-muted-foreground">{recap.pole_sitter_team}</p>
          </div>
        </div>
      )}

      {recap.pole_time != null && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Pole Time</span>
          <LapTimeDisplay time={recap.pole_time} className="font-mono" />
        </div>
      )}

      {recap.gap_to_p2 && (
        <p className="text-sm font-mono text-muted-foreground">
          Gap to P2: <span className="text-foreground">{recap.gap_to_p2}</span>
        </p>
      )}

      {recap.q1_cutoff_time != null && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Q1 Cutoff</span>
          <LapTimeDisplay time={recap.q1_cutoff_time} className="font-mono" />
        </div>
      )}

      {recap.q2_cutoff_time != null && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Q2 Cutoff</span>
          <LapTimeDisplay time={recap.q2_cutoff_time} className="font-mono" />
        </div>
      )}

      {(recap.red_flag_count ?? 0) > 0 && (
        <p className="text-sm text-muted-foreground">
          🚩 Red Flag{(recap.red_flag_count ?? 0) > 1 ? ` ×${recap.red_flag_count}` : ''}
        </p>
      )}
    </div>
  );
}
