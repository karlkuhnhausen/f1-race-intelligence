import type { PitStop } from './analysisTypes';

interface PitStopTimelineProps {
  pits: PitStop[];
  totalLaps: number;
}

/**
 * Pit Stop Timeline — shows when each driver pitted with duration markers.
 * Each row = one driver, dots at pit lap positions with size indicating duration.
 */
export default function PitStopTimeline({
  pits,
  totalLaps,
}: PitStopTimelineProps) {
  // Group by driver
  const byDriver = new Map<string, PitStop[]>();
  for (const pit of pits) {
    const key = pit.driver_acronym;
    if (!byDriver.has(key)) {
      byDriver.set(key, []);
    }
    byDriver.get(key)!.push(pit);
  }

  const drivers = Array.from(byDriver.entries()).sort((a, b) => {
    const aNum = a[1][0]?.driver_number ?? 0;
    const bNum = b[1][0]?.driver_number ?? 0;
    return aNum - bNum;
  });

  // Find max stop duration for relative sizing
  const maxDuration = Math.max(
    ...pits.map((p) => p.stop_duration || p.pit_duration),
    3,
  );

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-4 overflow-x-auto">
      <div className="space-y-1">
        {drivers.map(([acronym, driverPits]) => (
          <div key={acronym} className="flex items-center gap-2">
            <span className="w-10 text-xs font-mono text-muted-foreground text-right shrink-0">
              {acronym}
            </span>
            <div className="flex-1 relative h-6 bg-muted/30 rounded-sm">
              {driverPits.map((pit, idx) => {
                const left = ((pit.lap - 1) / totalLaps) * 100;
                const duration = pit.stop_duration || pit.pit_duration;
                // Scale dot size: normal stops ~2-3s = small, slow stops >5s = large
                const size = Math.max(8, Math.min(20, (duration / maxDuration) * 20));
                const isSlow = duration > 5;
                return (
                  <div
                    key={`${acronym}-${idx}`}
                    className={`absolute top-1/2 -translate-y-1/2 rounded-full ${
                      isSlow ? 'bg-accent-red' : 'bg-accent-cyan'
                    }`}
                    style={{
                      left: `${left}%`,
                      width: `${size}px`,
                      height: `${size}px`,
                    }}
                    title={`Lap ${pit.lap}: ${duration.toFixed(1)}s${pit.stop_duration ? ` (stop: ${pit.stop_duration.toFixed(1)}s)` : ''}`}
                  />
                );
              })}
            </div>
          </div>
        ))}
      </div>

      {/* Lap axis */}
      <div className="flex items-center gap-2 mt-2">
        <span className="w-10 shrink-0" />
        <div className="flex-1 flex justify-between text-xs text-muted-foreground">
          <span>1</span>
          <span>{Math.round(totalLaps / 4)}</span>
          <span>{Math.round(totalLaps / 2)}</span>
          <span>{Math.round((totalLaps * 3) / 4)}</span>
          <span>{totalLaps}</span>
        </div>
      </div>

      {/* Legend */}
      <div className="flex gap-4 mt-3 text-xs">
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-accent-cyan" />
          <span className="text-muted-foreground">Normal stop (≤5s)</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-accent-red" />
          <span className="text-muted-foreground">Slow stop (&gt;5s)</span>
        </div>
      </div>
    </div>
  );
}
