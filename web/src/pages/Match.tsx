import { type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import {
  fetchMatch,
  fmtKick,
  fmtSigned,
  fmtValue,
  playRows,
  type EvalSide,
  type MatchDetail,
  type MarketQuote,
  type OddsBoard,
  type Snapshot,
} from "../api";
import Layout from "../Layout";
import {
  AI_COLORS,
  DualCompare,
  GateBoard,
  GroupBars,
  KELLY_SCALE,
  KELLY_ZONES,
  PlayOddsChart,
  ScoreHeat,
  ShapeHero,
  VALUE_SCALE,
  VALUE_ZONES,
  VolumeChart,
  ZoneLegend,
  type DualRow,
  type GateSide,
} from "../Charts";
import { IconBack, IconBall, IconClock } from "../Icons";
import { LineupBoard } from "../Lineup";

export default function Match() {
  const { id } = useParams();
  const [kind, setKind] = useState<string>("");
  const [data, setData] = useState<MatchDetail | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    if (!id) return;
    void fetchMatch(id, kind || undefined)
      .then(setData)
      .catch((e) => setErr(e instanceof Error ? e.message : "失败"));
  }, [id, kind]);

  useEffect(() => {
    if (!id) return;
    const takes = data?.snapshot?.takes?.length ?? 0;
    if (takes > 0 || data?.match?.finished) return;
    const t = setInterval(() => {
      void fetchMatch(id, kind || undefined).then(setData).catch(() => undefined);
    }, 15000);
    return () => clearInterval(t);
  }, [id, kind, data?.snapshot?.takes?.length, data?.match?.finished]);

  if (err) {
    return (
      <Layout>
        <Link className="back" to="/">
          <IconBack size={14} /> 返回今日
        </Link>
        <div className="err">{err}</div>
      </Layout>
    );
  }
  if (!data) {
    return (
      <Layout>
        <div className="pending">读取中…</div>
      </Layout>
    );
  }

  const m = data.match;
  const sn = data.snapshot;
  const hasBoth = data.available.includes("open") && data.available.includes("close");
  const now = data.oddsClose || data.oddsOpen || sn?.odds;
  const market = data.market || sn?.market;
  const rows = playRows(now);
  const had = (sn?.eval ?? []).filter((x) => ["home", "draw", "away"].includes(x.key));
  const ou = (sn?.eval ?? []).filter((x) => x.key === "over" || x.key === "under");
  const asia = sn?.handicap?.sides ?? [];
  const pickedHad = pickSides(had, 2);
  const pickedOu = pickSides(ou, 1);
  const pickedAsia = pickSides(asia, 1);
  const hadGates = toGates(had, pickedHad, market?.betfair);
  const ouGates = toGates(ou, pickedOu, undefined);
  const asiaGates = toGates(asia, pickedAsia, undefined);
  const picked = pickedHad.concat(pickedOu, pickedAsia);
  const best = [...hadGates, ...ouGates, ...asiaGates]
    .filter((x) => x.picked)
    .sort((a, b) => rank(b.verdict) - rank(a.verdict) || b.value - a.value)[0];

  return (
    <Layout>
      <Link className="back" to={m.finished ? "/results" : "/"}>
        <IconBack size={14} /> {m.finished ? "返回结果" : "返回今日"}
      </Link>
      <div className="page-head">
        <div>
          <div className="num">{m.numStr}</div>
          <div className="teams" style={{ marginTop: 8 }}>
            {m.home} {m.finished && m.homeGoals != null && m.awayGoals != null ? `${m.homeGoals}-${m.awayGoals}` : "vs"} {m.away}
          </div>
          <div className="kick">
            <IconBall size={12} /> {m.leagueAbb}
            <IconClock size={12} /> {fmtKick(m.kickoff)}
          </div>
        </div>
        {hasBoth ? (
          <div className="switch">
            <button className={kind === "open" ? "btn" : "btn ghost"} onClick={() => setKind("open")}>
              赛前
            </button>
            <button className={kind !== "open" ? "btn" : "btn ghost"} onClick={() => setKind("close")}>
              临场
            </button>
          </div>
        ) : (
          <div className={`pill ${data.status === "临场" ? "on" : ""}`}>{data.status}</div>
        )}
      </div>

      {!sn ? (
        <div className="pending">这场还在分析。</div>
      ) : (
        <>
          <div className="verdict-strip">
            <div>
              <h1 className="headline">{sn.headline}</h1>
              <p className="layer-talk">{basicTalk(sn, now)}</p>
            </div>
            <div className="verdict-side">
              {best ? (
                <>
                  <span className={`verdict v-${best.verdict}`}>{best.verdict}</span>
                  <b>{best.label}</b>
                </>
              ) : (
                <span className="pred muted">价值还没齐</span>
              )}
              <div className="ai-row" style={{ margin: 0 }}>
                {(sn.usedModels ?? []).map((n) => (
                  <span className="hud-chip" key={n}>
                    {n}
                  </span>
                ))}
                {sn.usedAI ? null : <span className="hud-chip dim">本地模板</span>}
              </div>
            </div>
          </div>

          <Layer n="01" title="走势" hint="先看这场更常怎么结束，再看阵容能不能撑住这个判断。">
            <ShapeHero home={sn.homeWin} draw={sn.draw} away={sn.awayWin} over={sn.over25} under={sn.under25} />
            <div className="block">
              <div className="block-h">比分分布 · 主进球 \ 客进球</div>
              <ScoreHeat tops={sn.topScores ?? []} grid={sn.grid ?? []} />
            </div>
            <div className="block">
              <div className="block-h">阵容 · 首发 · 近期评分</div>
              {data.preview?.home || data.preview?.away ? (
                <LineupBoard preview={data.preview} />
              ) : (
                <div className="pred muted">阵容还在采集。</div>
              )}
            </div>
          </Layer>

          <Layer n="02" title="票面" hint="这里只看竞彩怎么卖。价值不看这组 SP。">
            <PlayOddsChart rows={rows} />
            {market?.betfair ? (
              <VolumeChart
                home={market.betfair.homeVol}
                draw={market.betfair.drawVol}
                away={market.betfair.awayVol}
                thin={market.betfair.thin}
                note={market.betfair.note}
              />
            ) : (
              <div className="pred muted">必发还在采集。</div>
            )}
          </Layer>

          <Layer n="03" title="价值" hint="定方向后过三关。对照 Bet365，不用竞彩 SP。">
            <p className="layer-talk">{valueTalk(hadGates, ouGates, asiaGates, sn.handicap?.talk)}</p>
            <div className="block">
              <div className="block-h">定方向</div>
              <div className="dir-chips">
                {picked.map((s) => (
                  <span className="dir-chip" key={s.key}>
                    {s.label}
                  </span>
                ))}
                {!picked.length ? <span className="pred muted">Bet365 还没齐。</span> : null}
              </div>
            </div>
            <DualCompare title="胜平负 · 模型 vs Bet365" rows={toDual(had)} empty="Bet365 欧赔还在采集。" />
            <DualCompare title="大小球 · 模型 vs Bet365" rows={toDual(ou)} empty="Bet365 大小还在采集。" />
            <DualCompare title="亚赔 · 模型 vs Bet365" rows={toDual(asia)} empty="Bet365 亚赔还在采集。" />
            <div className="eval-guide">
              <ZoneLegend label="价值差" min={VALUE_SCALE.min} max={VALUE_SCALE.max} zones={VALUE_ZONES} />
              <ZoneLegend label="凯利" min={KELLY_SCALE.min} max={KELLY_SCALE.max} zones={KELLY_ZONES} />
            </div>
            <GateBoard title="胜平负 · 三关" sides={hadGates} empty="Bet365 欧赔还在采集。" />
            <GateBoard title="大小球 · 三关" sides={ouGates} empty="Bet365 大小还在采集。" />
            <GateBoard title="亚赔 · 三关" sides={asiaGates} empty="Bet365 亚赔还在采集。" />
          </Layer>

          <Layer n="04" title="专家会诊" hint="后台用基础数据和价值参数自动算完。每位专家给完整解读和怎么买。第二天到专家战绩对账。">
            <AICompare
              sn={sn}
              finished={!!m.finished}
              score={m.finished && m.homeGoals != null && m.awayGoals != null ? `${m.homeGoals}-${m.awayGoals}` : ""}
            />
          </Layer>
        </>
      )}
    </Layout>
  );
}

function AICompare({
  sn,
  finished,
  score,
}: {
  sn: Snapshot;
  finished?: boolean;
  score?: string;
}) {
  const takes = sn.takes ?? [];
  const cards = takes.slice();
  const series = cards.map((t) => ({
    name: t.role || t.name,
    color: AI_COLORS[t.name] || "var(--muted)",
  }));
  return (
    <>
      {finished && score ? <p className="pred muted">完场 {score}。本场对账如下，汇总在结果页。</p> : null}
      {!cards.length ? (
        <p className="pred muted">专家还在后台算。用的是阵容、票面和 Bet365 价值参数，算完会自动出现，不用点。</p>
      ) : (
        <>
          <GroupBars
            title="专家胜平负"
            series={series}
            categories={["胜", "平", "负"]}
            values={[
              cards.map((t) => t.homeWin),
              cards.map((t) => t.draw),
              cards.map((t) => t.awayWin),
            ]}
          />
          <GroupBars
            title="专家大 2.5"
            series={series}
            categories={["大 2.5"]}
            values={[cards.map((t) => t.over25)]}
          />
          <div className="ai-compare">
            {cards.map((t) => (
              <article className={`ai-card${t.verdict === "主推" ? " blend" : ""}`} key={t.name}>
                <div className="ai-card-head">
                  <span className="hud-chip">{t.role || t.name}</span>
                  <span className="hud-chip dim">{t.name}</span>
                  {t.verdict ? <span className={`verdict v-${t.verdict}`}>{t.verdict}</span> : null}
                  <b>{t.headline || "—"}</b>
                </div>
                <div className="dir-chips">
                  {t.pick1x2 ? <span className="dir-chip">胜平负 {t.pick1x2}</span> : null}
                  {t.pickOu ? <span className="dir-chip">大小 {t.pickOu}</span> : null}
                  {t.hit1x2 != null ? (
                    <span className={`verdict ${t.hit1x2 ? "v-主推" : "v-放弃"}`}>{t.hit1x2 ? "胜平负中" : "胜平负未中"}</span>
                  ) : null}
                  {t.hitOu != null ? (
                    <span className={`verdict ${t.hitOu ? "v-主推" : "v-放弃"}`}>{t.hitOu ? "大小中" : "大小未中"}</span>
                  ) : null}
                </div>
                <p>{t.plainTalk || "没有白话。"}</p>
                {t.buyTalk ? (
                  <p className="buy-talk">
                    <b>怎么买</b>
                    {t.buyTalk}
                  </p>
                ) : null}
              </article>
            ))}
          </div>
        </>
      )}
    </>
  );
}

function Layer({ n, title, hint, children }: { n: string; title: string; hint: string; children: ReactNode }) {
  return (
    <section className="layer">
      <div className="layer-head">
        <span className="layer-n">{n}</span>
        <div>
          <h2>{title}</h2>
          <p>{hint}</p>
        </div>
      </div>
      <div className="layer-body">{children}</div>
    </section>
  );
}

function toDual(sides: EvalSide[]): DualRow[] {
  return sides
    .filter((s) => (s.model ?? 0) > 0 || (s.market ?? 0) > 0)
    .map((s) => ({ label: s.label, model: s.model ?? 0, market: s.market ?? 0 }));
}

function pickSides(sides: EvalSide[], n: number): EvalSide[] {
  return [...sides].sort((a, b) => (b.model ?? 0) - (a.model ?? 0)).slice(0, n);
}

function hotOf(s: EvalSide, bf?: MarketQuote["betfair"]): { hot: boolean; note: string } {
  const marketHot = (s.market ?? 0) > (s.model ?? 0) + 1;
  let crowd = false;
  if (bf && (s.key === "home" || s.key === "draw" || s.key === "away")) {
    const tot = Math.max(1, bf.homeVol + bf.drawVol + bf.awayVol);
    const vol = s.key === "home" ? bf.homeVol : s.key === "draw" ? bf.drawVol : bf.awayVol;
    crowd = vol / tot >= 0.5;
  }
  if (marketHot && crowd) return { hot: true, note: "过热" };
  if (marketHot) return { hot: true, note: "盘口偏热" };
  if (crowd) return { hot: true, note: "必发一边倒" };
  return { hot: false, note: "不热" };
}

function verdictOf(s: EvalSide, hot: boolean): GateSide["verdict"] {
  if (s.kellyBand === "紧" || s.value < 0) return "放弃";
  if (hot && s.kellyBand === "紧") return "放弃";
  if (s.value >= 3 && s.kellyBand !== "紧" && !hot) return "主推";
  if (s.value >= 3 && s.kellyBand === "松") return "可看";
  if (s.value >= 0 && s.kellyBand !== "紧" && !hot) return "可看";
  return "放弃";
}

function toGates(sides: EvalSide[], picked: EvalSide[], bf?: MarketQuote["betfair"]): GateSide[] {
  const keys = new Set(picked.map((x) => x.key));
  return sides.map((s) => {
    const hot = hotOf(s, bf);
    return {
      key: s.key,
      label: s.label,
      value: s.value,
      valueBand: s.valueBand,
      kelly: s.kelly,
      kellyBand: s.kellyBand,
      hot: hot.hot,
      hotNote: hot.note,
      verdict: verdictOf(s, hot.hot),
      picked: keys.has(s.key),
    };
  });
}

function basicTalk(sn: Snapshot, now?: OddsBoard | null): string {
  const sides = [
    { n: "主胜", p: sn.homeWin },
    { n: "平局", p: sn.draw },
    { n: "客胜", p: sn.awayWin },
  ].sort((a, b) => b.p - a.p);
  let s = `更常出现的是${sides[0].n}。`;
  s += sn.over25 >= sn.under25 ? "进球会稍多一点。" : "进球不会太多。";
  const hc = fmtSigned(now?.hhadLine);
  if (hc) s += `竞彩让球是${hc}。`;
  if (now && now.over > 1 && now.under > 1) s += "竞彩大小看 2.5。";
  if (sn.topScores?.[0]) s += `比分里 ${sn.topScores[0].score} 更常见。`;
  return s;
}

function valueTalk(had: GateSide[], ou: GateSide[], asia: GateSide[], asiaTalk?: string): string {
  const all = [...had, ...ou, ...asia].filter((x) => x.picked);
  const best = [...all].sort((a, b) => rank(b.verdict) - rank(a.verdict) || b.value - a.value)[0];
  if (!best) return "Bet365 盘还没齐，价值三关暂时走不完。";
  const parts = [`方向看${best.label}，结论${best.verdict}。`];
  parts.push(`价值差 ${fmtValue(best.value)}，凯利 ${best.kelly.toFixed(2)} ${best.kellyBand}，${best.hotNote}。`);
  if (asiaTalk) parts.push(asiaTalk);
  return parts.join("");
}

function rank(v: GateSide["verdict"]): number {
  if (v === "主推") return 2;
  if (v === "可看") return 1;
  return 0;
}
