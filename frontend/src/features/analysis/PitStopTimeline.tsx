import type { PitStop } from './analysisTypes';

interface PitStopTimelineProps {
  pits: PitStop[];
  totalLaps: number;
  driverOrder?: Map<number, number>;
  teamColors?: Map<number, string>;
}

/**
 * Pit Stop Timeline — shows when each driver pitted with duration markers.
 * Each row = one driver, dots at pit lap positions with size indicating duration.
 */
export default function PitStopTimeline({
  pits,
  totalLaps,
  driverOrder,
  teamColors,
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

  // Sort drivers by finishing position (P1 at top) if driverOrder provided, else by number
  const drivers = Array.from(byDriver.entries()).sort((a, b) => {
    if (driverOrder) {
      const aPos = driverOrder.get(a[1][0]?.driver_number ?? 0) ?? 99;
      const bPos = driverOrder.get(b[1][0]?.driver_number ?? 0) ?? 99;
      return aPos - bPos;
    }
    const aNum = a[1][0]?.driver_number ?? 0;
    const bNum = b[1][0]?.driver_number ?? 0;
    return aNum - bNum;
  });

  // Determine if stop_duration (crew time) is available for this session
  const hasStopDuration = pits.some((p) => p.stop_duration != null && p.stop_duration > 0);
  const slowThreshold = hasStopDuration ? 5 : 30;
  const durationLabel = hasStopDuration ? 'Stop Duration' : 'Lane Duration';

  // Find max duration for relative sizing
  const maxDuration = Math.max(
    ...pits.map((p) => hasStopDuration ? (p.stop_duration ?? p.pit_duration) : p.pit_duration),
    3,
  );

  // Generate X-axis ticks at every 5-lap interval
  const xTicks: number[] = [1];
  for (let t = 5; t <= totalLaps; t += 5) {
    xTicks.push(t);
  }
  if (xTicks[xTicks.length - 1] !== totalLaps) {
    xTicks.push(totalLaps);
  }

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-4">
      <div className="space-y-0">
        {drivers.map(([acronym, driverPits], idx) => (
          <div
            key={acronym}
            className={`flex items-center gap-2 py-0.5 ${idx < drivers.length - 1 ? 'border-b border-border/70' : ''}`}
          >
            <span
              className="w-10 text-xs font-mono text-right shrink-0"
              style={{ color: teamColors?.get(driverPits[0]?.driver_number ?? 0) ? `#${teamColors.get(driverPits[0]?.driver_number ?? 0)}` : '#8888aa' }}
            >
              {acronym}
            </span>
            <div className="flex-1 relative h-6 bg-muted/30 rounded-sm min-w-0">
              {driverPits.map((pit, pidx) => {
                const left = ((pit.lap - 1) / totalLaps) * 100;
                // Use crew stop time when available, otherwise lane time
                const displayTime = hasStopDuration ? (pit.stop_duration ?? pit.pit_duration) : pit.pit_duration;
                // Scale dot size based on displayed time
                const size = Math.max(8, Math.min(20, (displayTime / maxDuration) * 20));
                const isSlow = displayTime > slowThreshold;
                // Tooltip: show primary time, include lane time as context when showing stop duration
                const tooltip = hasStopDuration
                  ? `${acronym} Lap ${pit.lap}: ${displayTime.toFixed(1)}s stop (Lane: ${pit.pit_duration.toFixed(1)}s)`
                  : `${acronym} Lap ${pit.lap}: ${pit.pit_duration.toFixed(1)}s lane`;
                return (
                  <div
                    key={`${acronym}-${pidx}`}
                    className={`absolute top-1/2 -translate-y-1/2 rounded-full ${
                      isSlow ? 'bg-accent-red' : 'bg-accent-cyan'
                    }`}
                    style={{
                      left: `${left}%`,
                      width: `${size}px`,
                      height: `${size}px`,
                    }}
                    title={tooltip}
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
        <div className="flex-1 relative h-4 min-w-0">
          {xTicks.map((tick) => (
            <span
              key={tick}
              className="absolute -translate-x-1/2 text-xs"
              style={{ left: `${((tick - 1) / (totalLaps - 1)) * 100}%`, color: '#ffffff' }}
            >
              {tick}
            </span>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="flex gap-4 mt-3 text-xs">
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-accent-cyan" />
          <span className="text-muted-foreground">Normal {durationLabel.toLowerCase()} (≤{slowThreshold}s)</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-accent-red" />
          <span className="text-muted-foreground">Slow {durationLabel.toLowerCase()} (&gt;{slowThreshold}s)</span>
        </div>
      </div>
    </div>
  );
}
