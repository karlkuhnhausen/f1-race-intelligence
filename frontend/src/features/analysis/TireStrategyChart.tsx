import type { Stint } from './analysisTypes';

interface TireStrategyChartProps {
  stints: Stint[];
  totalLaps: number;
  driverOrder?: Map<number, number>;
  teamColors?: Map<number, string>;
}

const COMPOUND_COLORS: Record<string, string> = {
  SOFT: '#FF3333',
  MEDIUM: '#FFC700',
  HARD: '#EBEBEB',
  INTERMEDIATE: '#43B02A',
  WET: '#0072C6',
  UNKNOWN: '#555555',
};

/**
 * Tire Strategy Swimlane — horizontal bars per driver showing compound usage.
 * Each row = one driver, each block = one stint colored by compound.
 */
export default function TireStrategyChart({
  stints,
  totalLaps,
  driverOrder,
  teamColors,
}: TireStrategyChartProps) {
  // Group stints by driver
  const byDriver = new Map<string, Stint[]>();
  for (const stint of stints) {
    const key = stint.driver_acronym;
    if (!byDriver.has(key)) {
      byDriver.set(key, []);
    }
    byDriver.get(key)!.push(stint);
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
      {/* Legend */}
      <div className="flex gap-4 mb-4 text-xs">
        {Object.entries(COMPOUND_COLORS)
          .filter(([compound]) => compound !== 'UNKNOWN')
          .map(([compound, color]) => (
          <div key={compound} className="flex items-center gap-1">
            <div
              className="w-3 h-3 rounded-sm"
              style={{ backgroundColor: color }}
            />
            <span className="text-muted-foreground">{compound}</span>
          </div>
        ))}
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded-sm" style={{ backgroundColor: '#555555' }} />
          <span className="text-muted-foreground">UNKNOWN</span>
        </div>
      </div>

      {/* Swimlane rows */}
      <div className="space-y-1">
        {drivers.map(([acronym, driverStints]) => (
          <div key={acronym} className="flex items-center gap-2">
            <span
              className="w-10 text-xs font-mono text-right shrink-0"
              style={{ color: teamColors?.get(driverStints[0]?.driver_number ?? 0) ? `#${teamColors.get(driverStints[0]?.driver_number ?? 0)}` : '#8888aa' }}
            >
              {acronym}
            </span>
            <div className="flex-1 relative h-5 bg-muted/30 rounded-sm min-w-0">
              {driverStints
                .sort((a, b) => a.stint_number - b.stint_number)
                .map((stint) => {
                  const left = ((stint.lap_start - 1) / totalLaps) * 100;
                  const width =
                    ((stint.lap_end - stint.lap_start + 1) / totalLaps) * 100;
                  return (
                    <div
                      key={`${acronym}-${stint.stint_number}`}
                      className="absolute top-0 h-full rounded-sm opacity-90"
                      style={{
                        left: `${left}%`,
                        width: `${width}%`,
                        backgroundColor:
                          COMPOUND_COLORS[stint.compound] || COMPOUND_COLORS['UNKNOWN'],
                      }}
                      title={`${stint.compound} (Laps ${stint.lap_start}–${stint.lap_end})`}
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
    </div>
  );
}
