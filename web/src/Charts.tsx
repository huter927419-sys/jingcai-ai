type Bar = { label: string; value: number; color: string };

export function BarChart({
  title,
  bars,
  unit = "%",
  max,
}: {
  title: string;
  bars: Bar[];
  unit?: string;
  max?: number;
}) {
  const hi = max ?? Math.max(1, ...bars.map((b) => Math.abs(b.value)));
  return (
    <div className="chart">
      <div className="chart-title">{title}</div>
      <div className="chart-bars">
        {bars.map((b) => {
          const w = (Math.abs(b.value) / hi) * 100;
          return (
            <div className="chart-row" key={b.label}>
              <span className="chart-lab">{b.label}</span>
              <div className="chart-track">
                <div className="chart-fill" style={{ width: `${w}%`, background: b.color }} />
              </div>
              <span className="chart-val">
                {b.value.toFixed(1)}
                {unit}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

const RING = 2 * Math.PI * 36;

export function Donut1X2({
  home,
  draw,
  away,
  size = 156,
}: {
  home: number;
  draw: number;
  away: number;
  size?: number;
}) {
  const tot = Math.max(1, home + draw + away);
  const segs = [
    { key: "胜", p: home, color: "var(--win)" },
    { key: "平", p: draw, color: "var(--draw)" },
    { key: "负", p: away, color: "var(--lose)" },
  ];
  const top = [...segs].sort((a, b) => b.p - a.p)[0];
  let offset = 0;
  return (
    <div className={`donut${size < 120 ? " sm" : ""}`} style={{ width: size, height: size }}>
      <svg viewBox="0 0 100 100" aria-hidden>
        <circle cx="50" cy="50" r="36" fill="none" stroke="var(--track)" strokeWidth="12" />
        <g transform="rotate(-90 50 50)">
          {segs.map((s) => {
            const len = (s.p / tot) * RING;
            const el = (
              <circle
                key={s.key}
                cx="50"
                cy="50"
                r="36"
                fill="none"
                stroke={s.color}
                strokeWidth="12"
                strokeDasharray={`${len} ${RING - len}`}
                strokeDashoffset={-offset}
                strokeLinecap="butt"
              />
            );
            offset += len;
            return el;
          })}
        </g>
      </svg>
      <div className="donut-center">
        <b>{top.p.toFixed(0)}%</b>
        <span>{top.key}</span>
      </div>
    </div>
  );
}

export function ShapeHero({
  home,
  draw,
  away,
  over,
  under,
}: {
  home: number;
  draw: number;
  away: number;
  over: number;
  under: number;
}) {
  const rows = [
    { lab: "胜", p: home, color: "var(--win)" },
    { lab: "平", p: draw, color: "var(--draw)" },
    { lab: "负", p: away, color: "var(--lose)" },
  ];
  const hi = Math.max(1, ...rows.map((r) => r.p));
  return (
    <div className="shape-hero">
      <Donut1X2 home={home} draw={draw} away={away} />
      <div className="shape-hero-side">
        <div className="block-h">胜平负</div>
        <div className="vbars">
          {rows.map((r) => (
            <div className="vbar-row" key={r.lab}>
              <span>{r.lab}</span>
              <div className="vbar-track">
                <i style={{ width: `${(r.p / hi) * 100}%`, background: r.color }} />
              </div>
              <b>{r.p.toFixed(1)}%</b>
            </div>
          ))}
        </div>
        <div className="block-h" style={{ marginTop: 14 }}>
          大小 2.5
        </div>
        <div className="play-prices two">
          <div>
            <b>{over.toFixed(1)}%</b>
            <span>大</span>
          </div>
          <div>
            <b>{under.toFixed(1)}%</b>
            <span>小</span>
          </div>
        </div>
        <StackedP pH={over} pA={under} twoWay />
      </div>
    </div>
  );
}

export function PredPanel(props: {
  home: number;
  draw: number;
  away: number;
  over: number;
  under: number;
}) {
  return <ShapeHero {...props} />;
}

export function ScanShape({
  home,
  draw,
  away,
  over,
}: {
  home: number;
  draw: number;
  away: number;
  over: number;
}) {
  return (
    <div className="scan-shape">
      <Donut1X2 home={home} draw={draw} away={away} size={92} />
      <div className="scan-legend">
        <span>
          <i className="h" /> 胜 {home.toFixed(0)}%
        </span>
        <span>
          <i className="d" /> 平 {draw.toFixed(0)}%
        </span>
        <span>
          <i className="a" /> 负 {away.toFixed(0)}%
        </span>
        <em>大 2.5 {over.toFixed(0)}%</em>
      </div>
    </div>
  );
}

export type DualRow = { label: string; model: number; market: number };

export function DualCompare({ title, rows, empty }: { title: string; rows: DualRow[]; empty?: string }) {
  if (!rows.length) {
    return (
      <div className="gbar-wrap">
        <div className="block-h">{title}</div>
        <div className="pred muted">{empty || "还没齐。"}</div>
      </div>
    );
  }
  return (
    <GroupBars
      title={title}
      series={[
        { name: "模型", color: "var(--win)" },
        { name: "Bet365", color: "var(--draw)" },
      ]}
      categories={rows.map((r) => r.label)}
      values={rows.map((r) => [r.model, r.market])}
    />
  );
}

export type BarSeries = { name: string; color: string };

export function GroupBars({
  title,
  series,
  categories,
  values,
}: {
  title: string;
  series: BarSeries[];
  categories: string[];
  values: number[][];
}) {
  const max = Math.max(1, ...values.flat());
  return (
    <div className="gbar-wrap">
      <div className="gbar-head">
        <div className="block-h" style={{ margin: 0 }}>
          {title}
        </div>
        <div className="gbar-legend">
          {series.map((s) => (
            <span key={s.name}>
              <i style={{ background: s.color }} />
              {s.name}
            </span>
          ))}
        </div>
      </div>
      <div className="gbar-plot">
        {categories.map((lab, i) => (
          <div className="gbar-cat" key={lab}>
            <div className="gbar-cols">
              {(values[i] ?? []).map((v, j) => (
                <div
                  className="gbar-col"
                  key={`${lab}-${series[j]?.name ?? j}`}
                  title={`${series[j]?.name ?? ""} ${v.toFixed(1)}%`}
                  style={{
                    height: `${(Math.max(0, v) / max) * 100}%`,
                    background: series[j]?.color,
                  }}
                />
              ))}
            </div>
            <em>{lab}</em>
          </div>
        ))}
      </div>
    </div>
  );
}

export function ScoreHeat({
  tops,
  grid,
}: {
  tops: { score: string; p: number }[];
  grid: number[][];
}) {
  const max = Math.max(1, ...grid.flat());
  return (
    <div className="score-panel">
      <div className="tops">
        {tops.map((x) => (
          <div className="score-chip" key={x.score}>
            {x.score}
            <b>{x.p.toFixed(1)}%</b>
          </div>
        ))}
      </div>
      <div className="heat-wrap">
        <div className="heat-axis">
          <span>主进球 ↓</span>
          <span>客进球 →</span>
        </div>
        <table className="heat">
          <thead>
            <tr>
              <th />
              {[0, 1, 2, 3, 4].map((a) => (
                <th key={a}>{a}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {grid.slice(0, 5).map((row, h) => (
              <tr key={h}>
                <th>{h}</th>
                {row.slice(0, 5).map((v, a) => {
                  const t = v / max;
                  return (
                    <td key={a} style={{ background: `rgba(34, 211, 238, ${0.04 + t * 0.42})` }}>
                      {v.toFixed(1)}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export type GateSide = {
  key: string;
  label: string;
  value: number;
  valueBand?: string;
  kelly: number;
  kellyBand: string;
  hot: boolean;
  hotNote: string;
  verdict: "主推" | "可看" | "放弃";
  picked?: boolean;
};

export function GateBoard({
  title,
  sides,
  empty,
}: {
  title: string;
  sides: GateSide[];
  empty?: string;
}) {
  if (!sides.length) {
    return (
      <div className="gate-board">
        <div className="block-h">{title}</div>
        <div className="pred muted">{empty || "机构盘还在采集。"}</div>
      </div>
    );
  }
  return (
    <div className="gate-board">
      <div className="block-h">{title}</div>
      <div className={`analyze-grid${sides.length <= 2 ? " two" : ""}`}>
        {sides.map((s) => {
          const vb = s.valueBand || (s.value >= 3 ? "有价值" : s.value >= 0 ? "边缘" : "无价值");
          return (
            <div className={`outcome${s.picked ? " on" : ""}`} key={s.key}>
              <div className="outcome-top">
                <strong>{s.label}</strong>
                <span className={`verdict v-${s.verdict}`}>{s.verdict === "放弃" ? "回避" : s.verdict}</span>
              </div>
              <div className={`outcome-diff ${s.value >= 0 ? "pos" : "neg"}`}>
                {s.value > 0 ? "+" : ""}
                {s.value.toFixed(1)}% · {vb}
              </div>
              <div className="outcome-meters">
                <ZoneMeter
                  value={s.value}
                  min={VALUE_SCALE.min}
                  max={VALUE_SCALE.max}
                  zones={VALUE_ZONES}
                  text={`${s.value > 0 ? "+" : ""}${s.value.toFixed(1)}%`}
                  band={vb}
                />
                <ZoneMeter
                  value={s.kelly}
                  min={KELLY_SCALE.min}
                  max={KELLY_SCALE.max}
                  zones={KELLY_ZONES}
                  text={s.kelly.toFixed(2)}
                  band={s.kellyBand}
                />
              </div>
              <div className={`hot-tag${s.hot ? " on" : ""}`}>{s.hotNote}</div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

type Zone = { from: number; to: number; label: string; tone: "bad" | "mid" | "good" };

export const VALUE_SCALE = { min: -8, max: 8 };
export const VALUE_ZONES: Zone[] = [
  { from: -8, to: 0, label: "无价值", tone: "bad" },
  { from: 0, to: 3, label: "边缘", tone: "mid" },
  { from: 3, to: 8, label: "有价值", tone: "good" },
];

export const KELLY_SCALE = { min: 0.88, max: 1.12 };
export const KELLY_ZONES: Zone[] = [
  { from: 0.88, to: 0.96, label: "紧", tone: "bad" },
  { from: 0.96, to: 1.02, label: "中", tone: "mid" },
  { from: 1.02, to: 1.12, label: "松", tone: "good" },
];

function clampPct(v: number, min: number, max: number) {
  return Math.max(1.5, Math.min(98.5, ((v - min) / (max - min)) * 100));
}

function ZoneTrack({
  min,
  max,
  zones,
  mark,
  tall,
}: {
  min: number;
  max: number;
  zones: Zone[];
  mark?: number;
  tall?: boolean;
}) {
  return (
    <div className={`meter-track${tall ? " tall" : ""}`}>
      <div className="meter-zones">
        {zones.map((z) => {
          const left = ((z.from - min) / (max - min)) * 100;
          const width = ((z.to - min) / (max - min)) * 100 - left;
          return (
            <div
              key={z.label}
              className={`meter-zone ${z.tone}`}
              style={{ left: `${left}%`, width: `${width}%` }}
            />
          );
        })}
        {zones.slice(1).map((z) => (
          <i
            key={`tick-${z.from}`}
            className="meter-tick"
            style={{ left: `${((z.from - min) / (max - min)) * 100}%` }}
          />
        ))}
      </div>
      {mark != null ? <b className="meter-mark" style={{ left: `${clampPct(mark, min, max)}%` }} /> : null}
    </div>
  );
}

export function ZoneLegend({
  label,
  min,
  max,
  zones,
}: {
  label: string;
  min: number;
  max: number;
  zones: Zone[];
}) {
  return (
    <div className="meter-legend">
      <div className="meter-kicker">{label}</div>
      <ZoneTrack min={min} max={max} zones={zones} tall />
      <div className="meter-labs">
        {zones.map((z) => (
          <span key={z.label} style={{ flex: Math.max(0.18, z.to - z.from) }}>
            {z.label}
          </span>
        ))}
      </div>
    </div>
  );
}

function toneOf(band: string): "bad" | "mid" | "good" {
  if (band === "有价值" || band === "松") return "good";
  if (band === "无价值" || band === "紧") return "bad";
  return "mid";
}

export function ZoneMeter({
  value,
  min,
  max,
  zones,
  text,
  band,
}: {
  value: number;
  min: number;
  max: number;
  zones: Zone[];
  text: string;
  band: string;
}) {
  return (
    <div className="meter">
      <ZoneTrack min={min} max={max} zones={zones} mark={value} />
      <div className={`meter-read ${toneOf(band)}`}>
        <b>{text}</b>
        <em>{band}</em>
      </div>
    </div>
  );
}

export function MiniMeter({
  value,
  min,
  max,
  zones,
}: {
  value: number;
  min: number;
  max: number;
  zones: Zone[];
}) {
  return (
    <div className="mini-meter">
      <ZoneTrack min={min} max={max} zones={zones} mark={value} />
    </div>
  );
}

export type PlayRowView = {
  key: string;
  label: string;
  line?: string;
  note?: string;
  h?: number;
  d?: number;
  a?: number;
  pH?: number;
  pD?: number;
  pA?: number;
  twoWay?: boolean;
  empty?: string;
  ends?: [string, string];
};

export function PlayOddsChart({ rows }: { rows: PlayRowView[] }) {
  return (
    <div className="play-odds">
      {rows.map((row, i) => (
        <div className="play-block" key={row.key} style={{ animationDelay: `${i * 70}ms` }}>
          <div className="play-block-head">
            <b>{row.label}</b>
            {row.note ? <em>{row.note}</em> : null}
          </div>
          {row.empty && !row.h && !row.a ? (
            <div className="play-empty">{row.empty}</div>
          ) : (
            <>
              <div className={`play-prices${row.twoWay ? " two" : ""}`}>
                <div>
                  <b>{fmtCell(row.h)}</b>
                  <span>{row.twoWay ? row.ends?.[0] ?? "主" : "胜"}</span>
                </div>
                {row.twoWay ? null : (
                  <div>
                    <b>{fmtCell(row.d)}</b>
                    <span>平</span>
                  </div>
                )}
                <div>
                  <b>{fmtCell(row.a)}</b>
                  <span>{row.twoWay ? row.ends?.[1] ?? "客" : "负"}</span>
                </div>
              </div>
              <StackedP pH={row.pH} pD={row.twoWay ? 0 : row.pD} pA={row.pA} twoWay={row.twoWay} />
            </>
          )}
        </div>
      ))}
    </div>
  );
}

function fmtCell(v?: number) {
  if (!v || v <= 0) return "—";
  return v.toFixed(2);
}

function StackedP({ pH = 0, pD = 0, pA = 0, twoWay }: { pH?: number; pD?: number; pA?: number; twoWay?: boolean }) {
  const tot = (pH || 0) + (twoWay ? 0 : pD || 0) + (pA || 0);
  if (tot <= 0) return <div className="play-bar empty" />;
  return (
    <div className="play-bar" title={`主 ${pH?.toFixed(0)}%${twoWay ? "" : ` 平 ${pD?.toFixed(0)}%`} 客 ${pA?.toFixed(0)}%`}>
      {pH > 0 ? <i className="h live" style={{ width: `${pH}%` }} /> : null}
      {!twoWay && pD > 0 ? <i className="d live" style={{ width: `${pD}%` }} /> : null}
      {pA > 0 ? <i className="a live" style={{ width: `${pA}%` }} /> : null}
    </div>
  );
}

export function SplitChart({
  title,
  left,
  right,
  hint,
}: {
  title: string;
  left: { label: string; value: number; text?: string; color?: string };
  right: { label: string; value: number; text?: string; color?: string };
  hint?: string;
}) {
  const tot = Math.max(1, left.value + right.value);
  return (
    <div className="play-block">
      <div className="play-block-head">
        <b>{title}</b>
        {hint ? <em>{hint}</em> : null}
      </div>
      <div className="play-prices two">
        <div>
          <b>{left.text ?? `${left.value.toFixed(0)}%`}</b>
          <span>{left.label}</span>
        </div>
        <div>
          <b>{right.text ?? `${right.value.toFixed(0)}%`}</b>
          <span>{right.label}</span>
        </div>
      </div>
      <div className="split-bar">
        <i className="live" style={{ width: `${(left.value / tot) * 100}%`, background: left.color || "var(--win)" }} />
        <i className="live" style={{ width: `${(right.value / tot) * 100}%`, background: right.color || "var(--lose)" }} />
      </div>
    </div>
  );
}

export function VolumeChart({
  home,
  draw,
  away,
  thin,
  note,
}: {
  home: number;
  draw: number;
  away: number;
  thin?: boolean;
  note?: string;
}) {
  const tot = Math.max(1, home + draw + away);
  const fmt = (v: number) => (v >= 10000 ? `${(v / 10000).toFixed(1)}万` : String(Math.round(v)));
  return (
    <div className="play-block">
      <div className="play-block-head">
        <b>必发成交</b>
        {thin ? <em className="warn">{note || "样本偏小"}</em> : null}
      </div>
      <div className="play-prices">
        <div>
          <b>{fmt(home)}</b>
          <span>主</span>
        </div>
        <div>
          <b>{fmt(draw)}</b>
          <span>平</span>
        </div>
        <div>
          <b>{fmt(away)}</b>
          <span>客</span>
        </div>
      </div>
      <div className="split-bar">
        <i className="live" style={{ width: `${(home / tot) * 100}%`, background: "var(--win)" }} />
        <i className="live" style={{ width: `${(draw / tot) * 100}%`, background: "var(--draw)" }} />
        <i className="live" style={{ width: `${(away / tot) * 100}%`, background: "var(--lose)" }} />
      </div>
    </div>
  );
}
