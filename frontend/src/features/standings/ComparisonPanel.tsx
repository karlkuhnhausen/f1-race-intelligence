import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import type { DriverComparisonResponse, ConstructorComparisonResponse } from './standingsApi';

interface ComparisonPanelProps {
  data: DriverComparisonResponse | ConstructorComparisonResponse | null;
  loading?: boolean;
}

function isDriverComparison(data: DriverComparisonResponse | ConstructorComparisonResponse): data is DriverComparisonResponse {
  return 'driver1' in data;
}

function DeltaBadge({ value, label }: { value: number; label: string }) {
  const color = value > 0 ? 'text-green-400' : value < 0 ? 'text-red-400' : 'text-muted-foreground';
  const prefix = value > 0 ? '+' : '';
  return (
    <span className={`inline-block rounded px-2 py-0.5 text-xs font-mono ${color} bg-background/50`}>
      {prefix}{value} {label}
    </span>
  );
}

export default function ComparisonPanel({ data, loading }: ComparisonPanelProps) {
  if (loading) return <p className="text-muted-foreground text-center py-4">Loading comparison…</p>;
  if (!data) return null;

  const isDriver = isDriverComparison(data);

  const name1 = isDriver ? data.driver1.driver_name : data.team1.team_name;
  const name2 = isDriver ? data.driver2.driver_name : data.team2.team_name;
  const color1 = isDriver ? data.driver1.team_color : data.team1.team_color;
  const color2 = isDriver ? data.driver2.team_color : data.team2.team_color;
  const points1 = isDriver ? data.driver1_points : data.team1_points;
  const points2 = isDriver ? data.driver2_points : data.team2_points;

  const chartData = data.rounds.map((round, i) => ({
    round,
    [name1]: points1[i] ?? 0,
    [name2]: points2[i] ?? 0,
  }));

  return (
    <div data-testid="comparison-panel" className="space-y-4 rounded-lg border border-border bg-surface p-4">
      <div className="flex justify-between items-center">
        <h4 className="font-display text-sm font-bold uppercase tracking-wider">
          {name1} vs {name2}
        </h4>
        <div className="flex gap-2 flex-wrap">
          <DeltaBadge value={data.deltas.points} label="pts" />
          <DeltaBadge value={data.deltas.wins} label="wins" />
          <DeltaBadge value={data.deltas.podiums} label="podiums" />
        </div>
      </div>

      {data.rounds.length > 1 && (
        <div className="h-[250px]">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
              <XAxis dataKey="round" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip />
              <Legend />
              <Line
                type="monotone"
                dataKey={name1}
                stroke={color1 ? `#${color1}` : '#3b82f6'}
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey={name2}
                stroke={color2 ? `#${color2}` : '#ef4444'}
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
