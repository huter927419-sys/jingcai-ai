import { Link, useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { fetchSFC, fmtPrice, type SFCMatch } from "../api";
import Layout from "../Layout";
import { Donut1X2 } from "../Charts";
import { IconBack, IconClock } from "../Icons";

function pct(n?: number): string {
  if (!Number.isFinite(n) || !n || n <= 0) return "—";
  return `${n.toFixed(1)}%`;
}

export default function SFCDetail() {
  const { no } = useParams();
  const [issue, setIssue] = useState("");
  const [row, setRow] = useState<SFCMatch | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    let alive = true;
    void fetchSFC()
      .then((j) => {
        if (!alive) return;
        setIssue(j.issue);
        const n = Number(no);
        setRow(j.matches.find((m) => m.no === n) ?? null);
      })
      .catch((e) => {
        if (alive) setErr(e instanceof Error ? e.message : "加载失败");
      });
    return () => {
      alive = false;
    };
  }, [no]);

  if (err) {
    return (
      <Layout>
        <Link className="back" to="/sfc"><IconBack size={14} /> 返回胜负彩</Link>
        <div className="err">{err}</div>
      </Layout>
    );
  }
  if (!row) {
    return (
      <Layout>
        <Link className="back" to="/sfc"><IconBack size={14} /> 返回胜负彩</Link>
        <div className="pending">{no ? "没有这场对阵" : "读取中…"}</div>
      </Layout>
    );
  }

  const asian = row.market?.asian;
  const ou = row.market?.ou;
  const eu = row.market?.eu;
  const hasMarket = Boolean(asian || ou || eu);

  return (
    <Layout>
      <Link className="back" to="/sfc"><IconBack size={14} /> 返回胜负彩</Link>
      <div className="page-head">
        <div>
          <div className="eyebrow"><span /> SFC {issue || "—"} · {String(row.no).padStart(2, "0")}</div>
          <h1>{row.home} vs {row.away}</h1>
          <div className="sub">
            <span className="league">{row.league}</span>
            <span className="kick"><IconClock size={12} />{row.kickoff || "待定"}</span>
            <span className={`sfc-src ${row.source === "研判" ? "on" : ""}`}>{row.source === "研判" ? "研判" : "均赔"}</span>
          </div>
        </div>
        <div className={`sfc-pick lg pick-${row.pick}`}>{row.pick}</div>
      </div>
      {row.talk ? <p className="pred sfc-talk">{row.talk}</p> : null}
      <section className="sfc-detail-hero">
        <Donut1X2 home={row.homeWin} draw={row.draw} away={row.awayWin} />
        <div className="sfc-detail-bars" aria-label="胜平负研判">
          <div className="block-h">胜平负</div>
          <DetailBar label="胜" value={row.homeWin} tone="win" active={row.pick === "胜"} />
          <DetailBar label="平" value={row.draw} tone="draw" active={row.pick === "平"} />
          <DetailBar label="负" value={row.awayWin} tone="lose" active={row.pick === "负"} />
        </div>
      </section>
      {asian ? (
        <section className="sfc-board">
          <div className="block-h">亚洲盘</div>
          <div className="sfc-board-line">
            <strong>{asian.line || "—"}</strong>
            <span>{row.handicap?.pick ? `倾向 ${row.handicap.pick}` : "依据盘口对照研判"}</span>
          </div>
          <div className="sfc-board-grid">
            <div><span>主队水位</span><b>{fmtPrice(asian.home, true)}</b><small>覆盖 {pct(row.handicap?.home ?? asian.pH)}</small></div>
            <div><span>客队水位</span><b>{fmtPrice(asian.away, true)}</b><small>覆盖 {pct(row.handicap?.away ?? asian.pA)}</small></div>
          </div>
          {row.handicap?.talk ? <p className="sfc-detail-note">{row.handicap.talk}</p> : null}
        </section>
      ) : null}
      {ou ? (
        <section className="sfc-board">
          <div className="block-h">大小球 {ou.line || ""}</div>
          <div className="sfc-board-grid">
            <div><span>大</span><b>{fmtPrice(ou.over, true)}</b><small>{pct(ou.pO)}</small></div>
            <div><span>小</span><b>{fmtPrice(ou.under, true)}</b><small>{pct(ou.pU)}</small></div>
          </div>
        </section>
      ) : null}
      {eu || (row.marketHome ?? 0) > 0 ? (
        <section className="sfc-compare">
          <div className="block-h">和市场对照</div>
          <div className="sfc-compare-row head"><span /><b>研判</b><b>市场</b></div>
          <div className="sfc-compare-row"><span>胜</span><b>{pct(row.homeWin)}</b><b>{pct(eu?.pH ?? row.marketHome)}</b></div>
          <div className="sfc-compare-row"><span>平</span><b>{pct(row.draw)}</b><b>{pct(eu?.pD ?? row.marketDraw)}</b></div>
          <div className="sfc-compare-row"><span>负</span><b>{pct(row.awayWin)}</b><b>{pct(eu?.pA ?? row.marketAway)}</b></div>
        </section>
      ) : null}
      {hasMarket ? (
        <p className="sfc-detail-note">本场不在竞彩开售范围内，盘口来自市场报价，没有竞彩票面和专家点评。</p>
      ) : (
        <p className="sfc-detail-note">市场盘口还在同步。有盘后会补上亚盘、大小球与欧赔对照。</p>
      )}
    </Layout>
  );
}

function DetailBar({ label, value, tone, active }: { label: string; value: number; tone: string; active: boolean }) {
  const w = Math.max(0, Math.min(100, value || 0));
  return (
    <div className={`sfc-prob ${tone}${active ? " active" : ""}`}>
      <span>{label}</span>
      <div className="sfc-track"><i style={{ width: `${w}%` }} /></div>
      <b>{pct(value)}</b>
    </div>
  );
}
