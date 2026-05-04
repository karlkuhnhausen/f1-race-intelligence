import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

export interface ProgressionEntry {
  name: string;
  color: string;
  pointsByRound: number[];
}

export interface ProgressionChartProps {
  rounds: string[];
  entries: ProgressionEntry[];
}

export default function ProgressionChart({ rounds, entries }: ProgressionChartProps) {
  if (rounds.length <= 1) {
    return (
      <p className="text-muted-foreground text-center py-8">
        Not enough data for progression chart (need at least 2 rounds).
      </p>
    );
  }

  // Transform into recharts data format: [{round: "Round 1", "Verstappen": 25, "Norris": 18}, ...]
  const data = rounds.map((round, i) => {
    const point: Record<string, string | number> = { round };
    for (const entry of entries) {
      point[entry.name] = entry.pointsByRound[i] ?? 0;
    }
    return point;
  });

  return (
    <div data-testid="progression-chart" className="w-full h-[400px]">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <XAxis
            dataKey="round"
            tick={{ fontSize: 12 }}
            className="text-muted-foreground"
          />
          <YAxis
            tick={{ fontSize: 12 }}
            className="text-muted-foreground"
          />
          <Tooltip
            contentStyle={{ backgroundColor: 'hsl(var(--background))', border: '1px solid hsl(var(--border))' }}
            labelStyle={{ color: 'hsl(var(--foreground))' }}
          />
          <Legend />
          {entries.map((entry) => (
            <Line
              key={entry.name}
              type="monotone"
              dataKey={entry.name}
              stroke={entry.color ? `#${entry.color}` : '#888'}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
