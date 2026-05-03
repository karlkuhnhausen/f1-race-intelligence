import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import type { IntervalDriver } from './analysisTypes';

interface GapChartProps {
  intervals: IntervalDriver[];
}

/**
 * Gap to Leader Progression — shows how time gaps evolve over the race.
 * Lapped drivers (gap = -1 sentinel) are excluded from display.
 */
export default function GapChart({ intervals }: GapChartProps) {
  // Find max laps across all drivers
  const maxLaps = Math.max(
    ...intervals.map((d) => (d.laps.length > 0 ? d.laps[d.laps.length - 1].lap : 0)),
  );

  // Transform: array of { lap, [acronym]: gap }
  const lapData: Record<string, number | null>[] = [];
  for (let lap = 1; lap <= maxLaps; lap++) {
    const point: Record<string, number | null> = { lap };
    for (const driver of intervals) {
      const entry = driver.laps.find((l) => l.lap === lap);
      if (entry && entry.gap_to_leader >= 0) {
        point[driver.driver_acronym] = entry.gap_to_leader;
      } else {
        point[driver.driver_acronym] = null; // lapped or no data
      }
    }
    lapData.push(point);
  }

  // Only show top ~10 drivers to avoid chart clutter
  const driversToShow = intervals.slice(0, 10);

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-4">
      <ResponsiveContainer width="100%" height={400}>
        <LineChart
          data={lapData}
          margin={{ top: 10, right: 30, left: 10, bottom: 10 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
          <XAxis
            dataKey="lap"
            type="number"
            domain={[1, maxLaps]}
            label={{ value: 'Lap', position: 'insideBottom', offset: -5 }}
            stroke="hsl(var(--muted-foreground))"
            tick={{ fontSize: 11 }}
          />
          <YAxis
            label={{
              value: 'Gap to Leader (s)',
              angle: -90,
              position: 'insideLeft',
            }}
            stroke="hsl(var(--muted-foreground))"
            tick={{ fontSize: 11 }}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--surface))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '6px',
            }}
            labelStyle={{ color: 'hsl(var(--foreground))' }}
          />
          <Legend wrapperStyle={{ fontSize: '11px' }} />
          {driversToShow.map((driver) => (
            <Line
              key={driver.driver_number}
              type="monotone"
              dataKey={driver.driver_acronym}
              name={driver.driver_acronym}
              stroke={`#${driver.team_colour || '888888'}`}
              dot={false}
              strokeWidth={1.5}
              connectNulls
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
