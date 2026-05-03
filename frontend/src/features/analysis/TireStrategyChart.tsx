import type { Stint } from './analysisTypes';

interface TireStrategyChartProps {
  stints: Stint[];
  totalLaps: number;
}

const COMPOUND_COLORS: Record<string, string> = {
  SOFT: '#FF3333',
  MEDIUM: '#FFC700',
  HARD: '#EBEBEB',
  INTERMEDIATE: '#43B02A',
  WET: '#0072C6',
};

/**
 * Tire Strategy Swimlane — horizontal bars per driver showing compound usage.
 * Each row = one driver, each block = one stint colored by compound.
 */
export default function TireStrategyChart({
  stints,
  totalLaps,
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

  // Sort drivers by their first stint's driver_number for consistent ordering
  const drivers = Array.from(byDriver.entries()).sort((a, b) => {
    const aNum = a[1][0]?.driver_number ?? 0;
    const bNum = b[1][0]?.driver_number ?? 0;
    return aNum - bNum;
  });

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-4 overflow-x-auto">
      {/* Legend */}
      <div className="flex gap-4 mb-4 text-xs">
        {Object.entries(COMPOUND_COLORS).map(([compound, color]) => (
          <div key={compound} className="flex items-center gap-1">
            <div
              className="w-3 h-3 rounded-sm"
              style={{ backgroundColor: color }}
            />
            <span className="text-muted-foreground">{compound}</span>
          </div>
        ))}
      </div>

      {/* Swimlane rows */}
      <div className="space-y-1">
        {drivers.map(([acronym, driverStints]) => (
          <div key={acronym} className="flex items-center gap-2">
            <span className="w-10 text-xs font-mono text-muted-foreground text-right shrink-0">
              {acronym}
            </span>
            <div className="flex-1 relative h-5 bg-muted/30 rounded-sm">
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
                          COMPOUND_COLORS[stint.compound] || '#666',
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
        <div className="flex-1 flex justify-between text-xs text-muted-foreground">
          <span>1</span>
          <span>{Math.round(totalLaps / 4)}</span>
          <span>{Math.round(totalLaps / 2)}</span>
          <span>{Math.round((totalLaps * 3) / 4)}</span>
          <span>{totalLaps}</span>
        </div>
      </div>
    </div>
  );
}
