import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { fetchSFC, type SFCMatch } from "../api";
import Layout from "../Layout";
import { IconClock } from "../Icons";

function pct(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return "—";
  return `${n.toFixed(1)}%`;
}

export default function SFC() {
  const [issue, setIssue] = useState("");
  const [rows, setRows] = useState<SFCMatch[]>([]);
  const [analyzed, setAnalyzed] = useState(0);
  const [err, setErr] = useState("");

  async function load() {
    try {
      setErr("");
      const j = await fetchSFC();
      setIssue(j.issue);
      setRows(j.matches);
      setAnalyzed(j.analyzed);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "加载失败");
    }
  }

  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), 30000);
    return () => clearInterval(t);
  }, []);

  const pickCount = { 胜: 0, 平: 0, 负: 0 };
  for (const m of rows) {
    if (m.pick === "胜" || m.pick === "平" || m.pick === "负") pickCount[m.pick] += 1;
  }

  return (
    <Layout>
      <div className="page-head today-head">
        <div>
          <div className="eyebrow"><span /> TRADITIONAL 14</div>
          <h1>胜负彩 · 14 场</h1>
          <div className="sub">
            当期 14 场胜、平、负概率。点进任一场可看详情；对得上竞彩的会进入完整研判页。
          </div>
        </div>
        <div className="refresh-note"><i /><span>数据自动更新<small>每 30 秒同步一次</small></span></div>
      </div>
      <div className="overview-grid sfc-overview">
        <div className="overview-item"><span>当期期号</span><strong>{issue || "—"}</strong><small>胜负彩</small></div>
        <div className="overview-item"><span>研判覆盖</span><strong>{analyzed}</strong><small>/ {rows.length || 14} 场</small></div>
        <div className="overview-item"><span>倾向统计</span><strong>{pickCount.胜}-{pickCount.平}-{pickCount.负}</strong><small>胜 / 平 / 负</small></div>
      </div>
      {err ? <div className="err">{err}</div> : null}
      {rows.length === 0 && !err ? (
        <div className="empty"><b>等待当期对阵</b><span>正在同步 14 场赛程与概率</span></div>
      ) : (
        <div className="sfc-list">
          {rows.map((m) => (
            <SFCRow key={m.no} m={m} />
          ))}
        </div>
      )}
    </Layout>
  );
}

function SFCRow({ m }: { m: SFCMatch }) {
  const inner = (
    <>
      <div className="sfc-meta">
        <span className="sfc-no">{String(m.no).padStart(2, "0")}</span>
        <span className="league">{m.league}</span>
        {m.numStr ? <span className="sfc-jc">{m.numStr}</span> : null}
        <span className={`sfc-src ${m.source === "研判" ? "on" : ""}`}>{m.source === "研判" ? "研判" : "均赔"}</span>
      </div>
      <div className="teams">
        <span>{m.home}</span><em>VS</em><span>{m.away}</span>
      </div>
      <div className="kick"><IconClock size={12} />{m.kickoff || "待定"}</div>
      <div className="sfc-odds" aria-label="胜平负概率">
        <Prob label="胜" value={m.homeWin} tone="win" active={m.pick === "胜"} />
        <Prob label="平" value={m.draw} tone="draw" active={m.pick === "平"} />
        <Prob label="负" value={m.awayWin} tone="lose" active={m.pick === "负"} />
      </div>
      <div className={`sfc-pick pick-${m.pick}`}>{m.pick}</div>
      <span className="sfc-more">详情</span>
    </>
  );
  const to = m.matchId ? `/matches/${m.matchId}` : `/sfc/${m.no}`;
  return <Link className="sfc-card linked" to={to}>{inner}</Link>;
}

function Prob({ label, value, tone, active }: { label: string; value: number; tone: string; active: boolean }) {
  const w = Math.max(0, Math.min(100, value || 0));
  return (
    <div className={`sfc-prob ${tone}${active ? " active" : ""}`}>
      <span>{label}</span>
      <div className="sfc-track"><i style={{ width: `${w}%` }} /></div>
      <b>{pct(value)}</b>
    </div>
  );
}
