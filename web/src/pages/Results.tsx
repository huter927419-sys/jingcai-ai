import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { fetchExperts, fmtKick, type ExpertBoardRow, type ModelTake, type SettledItem } from "../api";
import Layout from "../Layout";
import { GroupBars } from "../Charts";
import { IconChart, IconPulse, IconShield } from "../Icons";

export default function Results() {
  const [board, setBoard] = useState<ExpertBoardRow[]>([]);
  const [yesterday, setYesterday] = useState<SettledItem[]>([]);
  const [settled, setSettled] = useState<SettledItem[]>([]);
  const [pending, setPending] = useState(0);
  const [err, setErr] = useState("");

  useEffect(() => {
    let alive = true;
    const load = () =>
      fetchExperts()
        .then((j) => {
          if (!alive) return;
          setBoard(j.board);
          setYesterday(j.yesterday);
          setSettled(j.settled);
          setPending(j.pending);
        })
        .catch((e) => {
          if (!alive) return;
          setErr(e instanceof Error ? e.message : "加载失败");
        });
    void load();
    const t = setInterval(() => void load(), 12000);
    return () => {
      alive = false;
      clearInterval(t);
    };
  }, []);

  const ranked = board.filter((r) => r.games > 0);
  const leader = ranked[0];
  const verified = settled.length;

  return (
    <Layout>
      <div className="page-head results-head">
        <div>
          <div className="eyebrow"><span /> PERFORMANCE AUDIT</div>
          <h1>赛后复盘中心</h1>
          <div className="sub">所有赛前判断自动留档，以真实赛果持续检验研判表现</div>
        </div>
      </div>
      <section className="proof-strip" aria-label="分析验证概览">
        <div className="proof-lead">
          <IconShield size={21} />
          <span><b>结论可追溯，表现可验证</b><small>拒绝只展示正确案例，每场预测统一纳入对账</small></span>
        </div>
        <div className="proof-stat">
          <IconChart size={17} />
          <span>已对账场次<strong>{verified}</strong></span>
        </div>
        <div className="proof-stat">
          <IconPulse size={17} />
          <span>领先胜平负<strong>{leader?.games ? `${leader.rate1x2.toFixed(0)}%` : "—"}</strong></span>
        </div>
        <div className="proof-stat">
          <span>领先大小球<strong>{leader?.games ? `${leader.rateOu.toFixed(0)}%` : "—"}</strong></span>
        </div>
      </section>
      {err ? <div className="err">{err}</div> : null}
      {pending > 0 ? (
        <p className="layer-talk" style={{ marginBottom: 16 }}>
          还有 {pending} 场正在用赛前资料补预测，算完会自动对账。
        </p>
      ) : null}
      {!settled.length && !ranked.length ? (
        <div className="empty">还没有完场。赛后将自动对账，并计入各研判维度的表现统计。</div>
      ) : (
        <>
          {ranked.length ? (
            <>
              {leader ? (
                <p className="layer-talk" style={{ marginBottom: 16 }}>
                  目前表现领先的是{leader.role}，胜平负 {leader.hit1x2}/{leader.games}，大小{" "}
                  {leader.hitOu}/{leader.games}。
                </p>
              ) : null}
              <GroupBars
                title="研判维度命中率"
                series={[
                  { name: "胜平负", color: "var(--win)" },
                  { name: "大小 2.5", color: "var(--draw)" },
                ]}
                categories={board.map((r) => r.role)}
                values={board.map((r) => [r.rate1x2, r.rateOu])}
              />
              <div className="analyze-grid two" style={{ marginTop: 14 }}>
                {board.map((r, i) => (
                  <div className={`outcome${i === 0 && r.games ? " on" : ""}`} key={r.name}>
                    <div className="outcome-top">
                      <strong>{r.role}</strong>
                      <span className="hud-chip">{r.games} 场样本</span>
                    </div>
                    <div className="play-prices two">
                      <div>
                        <b>{r.games ? `${r.rate1x2.toFixed(0)}%` : "—"}</b>
                        <span>
                          胜平负 {r.hit1x2}/{r.games}
                        </span>
                      </div>
                      <div>
                        <b>{r.games ? `${r.rateOu.toFixed(0)}%` : "—"}</b>
                        <span>
                          大小 {r.hitOu}/{r.games}
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <p className="layer-talk" style={{ marginBottom: 16 }}>
              完场比分已记下。有完整赛前研判的场次才会进入命中率评比。
            </p>
          )}
          {yesterday.length ? (
            <ResultList title="昨日核对" items={yesterday} />
          ) : null}
          <ResultList title={yesterday.length ? "最近完场" : "完场对账"} items={settled} />
        </>
      )}
    </Layout>
  );
}

function ResultList({ title, items }: { title: string; items: SettledItem[] }) {
  return (
    <div className="block" style={{ marginTop: 18 }}>
      <div className="block-h">{title}</div>
      <div className="list">
        {items.map((it) => (
          <ResultCard key={it.match.id} it={it} />
        ))}
      </div>
    </div>
  );
}

function ResultCard({ it }: { it: SettledItem }) {
  const actual1x2 = actualOf(it.score);
  return (
    <Link className="match-card result-card" to={`/matches/${it.match.id}`}>
      <div className="match-card-head">
        <div>
          <div className="match-id">
            <span className="num">{it.match.numStr}</span>
            <span className="league">{it.match.leagueAbb || it.match.league}</span>
          </div>
          <div className="teams">
            {it.match.home} {it.score} {it.match.away}
          </div>
          <div className="kick">{fmtKick(it.match.kickoff)}</div>
        </div>
        <div className="pill on">完场</div>
      </div>
      <div className="result-actual">
        实际 胜平负 {actual1x2.had} · 大小 {actual1x2.ou}
      </div>
      <div className="result-picks">
        {it.takes.length ? (
          it.takes.map((t) => <PickRow key={t.name} t={t} />)
        ) : (
          <div className="pred muted">这场还没有完整赛前研判，只记录比分。</div>
        )}
      </div>
    </Link>
  );
}

function PickRow({ t }: { t: ModelTake }) {
  const both = t.hit1x2 && t.hitOu;
  return (
    <div className={`result-pick${both ? " hit-all" : ""}`}>
      <span className="result-who">
        {t.role || "专业研判"}
      </span>
      <span className={t.hit1x2 ? "hit" : "miss"}>胜平负 {t.pick1x2 || "—"}</span>
      <span className={t.hitOu ? "hit" : "miss"}>大小 {t.pickOu || "—"}</span>
      {t.verdict ? <span className={`verdict v-${t.verdict}`}>{t.verdict}</span> : null}
    </div>
  );
}

function actualOf(score: string): { had: string; ou: string } {
  const [h, a] = score.split("-").map((n) => Number.parseInt(n, 10));
  if (Number.isNaN(h) || Number.isNaN(a)) return { had: "—", ou: "—" };
  const had = h > a ? "胜" : h < a ? "负" : "平";
  const ou = h + a >= 3 ? "大" : "小";
  return { had, ou };
}
