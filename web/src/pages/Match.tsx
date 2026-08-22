import { type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import {
  fetchMatch,
  fmtKick,
  fmtPrice,
  fmtSigned,
  playMarketRows,
  playRows,
  type EvalSide,
  type MatchDetail,
  type MarketQuote,
  type ModelTake,
  type OddsBoard,
  type Snapshot,
} from "../api";
import Layout from "../Layout";
import {
  GroupBars,
  GateBoard,
  KELLY_SCALE,
  KELLY_ZONES,
  PlayOddsChart,
  ScoreHeat,
  ShapeHero,
  VALUE_SCALE,
  VALUE_ZONES,
  VolumeChart,
  ZoneLegend,
  type GateSide,
} from "../Charts";
import { IconBack, IconBall, IconChart, IconClock, IconGauge, IconGrid, IconScale, IconTalk } from "../Icons";
import { LineupBoard } from "../Lineup";
import { VerdictBadge, VerdictHelp, verdictLabel } from "../VerdictHelp";

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
  const sfc = m.origin === "sfc";
  const backTo = sfc ? "/lab/sfc" : m.finished ? "/results" : "/";
  const backLabel = sfc ? "返回胜负彩" : m.finished ? "返回结果" : "返回今日";
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
      <Link className="back" to={backTo}>
        <IconBack size={14} /> {backLabel}
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
              <span className="strip-kicker">结构估计 · 不是专家结论</span>
              <h1 className="headline">{shapeHeadline(sn)}</h1>
              <p className="layer-talk">{basicTalk(sn, now)}这是结构估计，用来理解比赛更常怎么走。专家怎么解盘、价格值不值得做，分别在第 03、04 层，不要和这里合成一句预测。</p>
            </div>
            <div className="verdict-side">
              {best ? (
                <>
                  <VerdictBadge verdict={best.verdict} />
                  <b>{best.label}</b>
                </>
              ) : (
                <span className="pred muted">价值还没齐</span>
              )}
              <div className="ai-row" style={{ margin: 0 }}>
                <span className="hud-chip">价值等级 · 价格值不值得做</span>
              </div>
            </div>
          </div>

          <DetailOverview sn={sn} best={best} now={now} previewReady={Boolean(data.preview?.home || data.preview?.away)} />
          <EvidencePanel sn={sn} data={data} now={now} market={market} />
          <nav className="detail-index" aria-label="详情分析导航">
            <a href="#trend"><IconChart size={16} /><span><b>01 格局</b><small>分布 · 阵容 · 不是推荐</small></span></a>
            <a href="#ticket"><IconGrid size={16} /><span><b>02 票面</b><small>{sfc ? "欧赔 · 亚盘 · 大小" : "赔率 · 让球 · 变化"}</small></span></a>
            <a href="#analysis"><IconTalk size={16} /><span><b>03 专家解盘</b><small>独立判断 · 允许分歧</small></span></a>
            <a href="#value"><IconScale size={16} /><span><b>04 价值研判™</b><small>价格 · 保护 · 是否执行</small></span></a>
          </nav>

          <Layer id="trend" n="01" title="比赛格局" hint="这是结构分布：这场更常怎么走。用来建立形态，不是专家推荐，也不是买入结论。">
            <ShapeHero home={sn.homeWin} draw={sn.draw} away={sn.awayWin} over={sn.over25} under={sn.under25} />
            <div className="block">
              <div className="block-h">情景比分分布 · 主进球 \ 客进球</div>
              <div className="score-note">
                <span>路径参考，不是比分预测</span>
                <p>热区只说明哪些比分路径相对更常见，用来理解节奏和进球区间。不要把它读成“会出这个比分”，也不要和专家卡片里的情景比分一一对赌。</p>
              </div>
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

          <Layer id="ticket" n="02" title="票面" hint={sfc ? "本场没有竞彩票面，这里只看市场欧赔、亚盘和大小。" : "这里只看竞彩怎么卖。价值不看这组 SP。"}>
            {sfc ? (
              <PlayOddsChart rows={playMarketRows(market)} />
            ) : (
              <>
                {data.oddsOpen && data.oddsClose ? <OddsMovement open={data.oddsOpen} close={data.oddsClose} /> : null}
                <PlayOddsChart rows={rows} />
              </>
            )}
            {market?.betfair ? (
              <VolumeChart
                home={market.betfair.homeVol}
                draw={market.betfair.drawVol}
                away={market.betfair.awayVol}
                thin={market.betfair.thin}
                note={market.betfair.note}
              />
            ) : sfc ? null : (
              <div className="pred muted">必发还在采集。</div>
            )}
            {market?.books?.length || market?.asianMove || market?.ouMove || market?.asianBooks?.length || market?.ouBooks?.length ? <MarketBooks market={market} /> : null}
            <RiskNotice tone="data">盘口、赔率与成交数据可能存在更新延迟或不同来源口径差异，临场变化会改变原有判断，请以实际出票时信息为准。</RiskNotice>
            {sfc ? null : <TicketAdvice sn={sn} odds={now} />}
          </Layer>

          <Layer id="analysis" n="03" title="专家解盘" hint="先看基本盘给出的盘面基准，再看四位专家从盘口、进球、阵容和定价独立解读。允许和基本盘不一致；不一致时继续看价值，而不是选一个当标准答案。">
            <div className="layer-guide">
              <b>怎么读这一层</b>
              <span>基本盘分析师只回答“价格和分布现在站哪边”。盘口分析师看升降水和诱盘，另外三位看进球、阵容和定价。五套口径并存，基本盘不是标准答案。</span>
            </div>
            {data.expertKind === "open" && sn.kind === "close" ? (
              <div className="expert-source-note">
                <b>赛前研判留档</b>
                <span>本场临场快照未生成新的专家解盘，以下内容采用赛前留档；临场盘口与价值数据仍以上方当前快照为准。</span>
              </div>
            ) : null}
            {m.finished && sn.kind === "open" && data.available.includes("close") ? (
              <div className="expert-source-note">
                <b>这是赛前稿</b>
                <span>专家临场可能改口。复盘成绩按临场解盘计算，切到临场才是计入对账的那一版。</span>
              </div>
            ) : null}
            <AICompare
              sn={sn}
              numStr={m.numStr}
              odds={now}
              finished={!!m.finished}
              score={m.finished && m.homeGoals != null && m.awayGoals != null ? `${m.homeGoals}-${m.awayGoals}` : ""}
            />
            <RiskNotice strong>以上分析和参考买入仅供信息研究，不构成收益保证。请结合自身判断审慎决策，由此产生的相关风险与结果由使用者自行承担。</RiskNotice>
          </Layer>

          <Layer id="value" n="04" title="价值研判™" hint="看完格局和专家之后，这里只回答一件事：当前价格值不值得做。方向对也可能回避。">
            <ValueMatrix had={hadGates} ou={ouGates} asia={asiaGates} picked={picked} />
            <div className="public-metrics">
              <div className="public-metrics-head"><b>公开指标参考</b><span>价值差、凯利与判定区间用于辅助理解；专属权重和综合裁决逻辑不公开。</span></div>
              <div className="eval-guide">
                <ZoneLegend label="价值差" min={VALUE_SCALE.min} max={VALUE_SCALE.max} zones={VALUE_ZONES} />
                <ZoneLegend label="凯利" min={KELLY_SCALE.min} max={KELLY_SCALE.max} zones={KELLY_ZONES} />
              </div>
              <GateBoard title="胜平负 · 指标明细" sides={hadGates} empty="Bet365 欧赔还在采集。" />
              <GateBoard title="大小球 · 指标明细" sides={ouGates} empty="Bet365 大小还在采集。" />
              <GateBoard title="亚洲盘 · 指标明细" sides={asiaGates} empty="Bet365 亚盘还在采集。" />
            </div>
            <RiskNotice>价值等级是基于当前数据的条件性判断，不代表结果承诺；市场快速变盘、阵容变化或突发事件均可能使结论失效。</RiskNotice>
          </Layer>
        </>
      )}
    </Layout>
  );
}

function DetailOverview({ sn, best, now, previewReady }: { sn: Snapshot; best?: GateSide; now?: OddsBoard | null; previewReady: boolean }) {
  const outcomes = [{ label: "主胜", p: sn.homeWin }, { label: "平局", p: sn.draw }, { label: "客胜", p: sn.awayWin }].sort((a, b) => b.p - a.p);
  const gap = outcomes[0].p - outcomes[1].p;
  const oddsReady = Number(now?.had?.H) > 1 && Number(now?.had?.D) > 1 && Number(now?.had?.A) > 1;
  const dataCount = Number(oddsReady) + Number(previewReady) + Number((sn.takes?.length ?? 0) > 0);
  return (
    <section className="detail-overview" aria-label="本场摘要指标">
      <div className="detail-overview-title"><span>本场摘要</span><b>先看格局，再核票面、专家与价格</b></div>
      <div><IconGauge size={17} /><span>概率重心<strong>{outcomes[0].label}</strong><small>{outcomes[0].p.toFixed(1)}% · 分布不是推荐</small></span></div>
      <div><IconChart size={17} /><span>方向强度<strong>{gap >= 18 ? "清晰" : gap >= 8 ? "倾向" : "胶着"}</strong><small>领先次选 {gap.toFixed(1)}%</small></span></div>
      <div><IconBall size={17} /><span>进球倾向<strong>{sn.over25 >= 52 ? "偏大" : sn.over25 <= 48 ? "偏小" : "均衡"}</strong><small>大 2.5 {sn.over25.toFixed(1)}%</small></span></div>
      <div><IconScale size={17} /><span>价值等级<strong>{best?.verdict ? <>{verdictLabel(best.verdict)} <VerdictHelp verdict={best.verdict} /></> : "待确认"}</strong><small>{best?.label || "等待市场数据"}</small></span></div>
      <div><IconGrid size={17} /><span>数据覆盖<strong>{dataCount}/3</strong><small>票面 · 阵容 · 研判</small></span></div>
    </section>
  );
}

function EvidencePanel({ sn, data, now, market }: { sn: Snapshot; data: MatchDetail; now?: OddsBoard | null; market?: MarketQuote | null }) {
  const hasOdds = Boolean(now?.had?.H && now.had.H > 1);
  const hasLineup = Boolean(data.preview?.home || data.preview?.away);
  const hasMarket = Boolean(market?.eu || market?.asian || market?.ou || market?.betfair);
  const hasClose = Boolean(data.oddsOpen && data.oddsClose);
  const covered = [hasOdds, hasLineup, hasMarket, (sn.takes?.length ?? 0) > 0].filter(Boolean).length;
  const picks = (sn.takes ?? []).map((x) => x.pick1x2).filter(Boolean);
  const consensus = picks.length ? Math.max(...["胜", "平", "负"].map((v) => picks.filter((x) => x === v).length)) / picks.length : 0;
  const confidenceScore = Math.min(100, covered * 18 + (hasClose ? 10 : 0) + Math.round(consensus * 18));
  const confidence = confidenceScore >= 80 ? "较高" : confidenceScore >= 60 ? "中等" : confidenceScore >= 40 ? "偏低" : "谨慎";
  const main = sn.homeWin >= sn.draw && sn.homeWin >= sn.awayWin ? "主胜" : sn.awayWin >= sn.draw ? "客胜" : "平局";
  const second = [...[{ label: "主胜", p: sn.homeWin }, { label: "平局", p: sn.draw }, { label: "客胜", p: sn.awayWin }]].sort((a, b) => b.p - a.p)[1];
  const risks = [
    !hasLineup ? "首发/阵容资料未完整确认" : "阵容资料已接入，但正式首发仍可能变化",
    !hasClose ? "缺少开盘到临场的完整变化链" : "临场盘口变化仍可能改变方向等级",
    picks.length > 1 && consensus < 0.75 ? "不同研判维度对胜平负方向存在分歧" : "多维判断较一致，但一致不代表结果确定",
    second.p >= (main === "主胜" ? sn.homeWin : main === "客胜" ? sn.awayWin : sn.draw) - 8 ? `重心${main}与次档${second.label}差距有限` : "红牌、点球等偶发事件无法由赛前数据覆盖",
  ];
    return <section className="evidence-panel" aria-label="数据质量与反向校验"><div className="evidence-head"><div><span>DATA QUALITY · RED TEAM CHECK</span><h3>证据基础与反向风险</h3></div><b>{covered}/4 项覆盖 · 置信度{confidence} {confidenceScore}</b></div><div className="evidence-grid"><div><span>数据截止</span><strong>{new Date(sn.fetchedAt).toLocaleString("zh-CN", { hour12: false })}</strong><small>以当前快照时间为准</small></div><div><span>已覆盖</span><strong>{[hasOdds && "竞彩票面", hasMarket && "市场报价", hasLineup && "阵容预览", (sn.takes?.length ?? 0) > 0 && "专家解盘"].filter(Boolean).join(" · ") || "基础数据"}</strong><small>仅列入系统实际拿到的数据</small></div><div><span>概率重心</span><strong>{main} · {Math.max(sn.homeWin, sn.draw, sn.awayWin).toFixed(1)}%</strong><small>结构估计，不是专家推荐</small></div><div className="evidence-risk"><span>反向校验</span>{risks.map((x, i) => <p key={i}>· {x}</p>)}</div></div><div className="evidence-foot">置信度依据数据覆盖、临场快照和多维方向一致性计算，不等同于胜率；正式首发、盘口临场变化和突发事件仍需重新核验。</div></section>;
}

function AICompare({
  sn,
  numStr,
  odds,
  finished,
  score,
}: {
  sn: Snapshot;
  numStr?: string;
  odds?: OddsBoard | null;
  finished?: boolean;
  score?: string;
}) {
  const takes = sn.takes ?? [];
  const cards = takes.slice();
  const voices = cards.filter((t) => t.roleKey !== "shape");
  const series = cards.map((t) => ({
    name: t.role || "专业研判",
    color: roleColor(t.roleKey),
  }));
  return (
    <>
      {finished && score ? <p className="pred muted">完场 {score}。本场对账如下，汇总在结果页。</p> : null}
      {!cards.length ? (
        <p className="pred muted">专家解盘正在生成，将综合阵容、盘口与市场数据自动更新。</p>
      ) : (
        <>
          {voices.length ? <TradePlan sn={sn} cards={voices} odds={odds} /> : <p className="pred muted">四位专家还在生成。下方先看盘面基准。</p>}
          <GroupBars
            title="专家各自的胜平负判断"
            series={series}
            categories={["胜", "平", "负"]}
            values={[
              cards.map((t) => t.homeWin),
              cards.map((t) => t.draw),
              cards.map((t) => t.awayWin),
            ]}
          />
          <GroupBars
            title="专家各自的进球判断"
            series={series}
            categories={["大 2.5"]}
            values={[cards.map((t) => t.over25)]}
          />
          <div className="ai-compare">
            {cards.map((t, index) => {
              const meta = roleMeta(t.roleKey);
              return (
              <article className={`ai-card role-${t.roleKey || "general"}${t.verdict === "主推" ? " blend" : ""}`} key={t.name}>
                <div className="ai-card-head">
                  <div className="expert-identity"><span>{String(index + 1).padStart(2, "0")}</span><div><b>{roleTitle(t.roleKey, t.role)}</b><small>{meta.focus}</small></div></div>
                  {t.verdict ? <VerdictBadge verdict={t.verdict} /> : null}
                </div>
                <h3>{t.headline || "研判结论待确认"}</h3>
                <div className="dir-chips">
                  {t.pattern ? <span className="dir-chip">格局 {t.pattern}</span> : null}
                  {t.pick1x2 ? <span className="dir-chip">胜平负 {t.pick1x2}</span> : null}
                  {t.pickHandicap ? <span className="dir-chip">让球 {t.pickHandicap}{handicapSp(t.pickHandicap, odds)}</span> : null}
                  {t.pickOu ? <span className="dir-chip">大小 {t.pickOu}</span> : null}
                  {t.scores?.length ? <span className="dir-chip">比分 {t.scores.join(" / ")}</span> : null}
                  {t.hit1x2 != null ? (
                    <span className={`verdict ${t.hit1x2 ? "v-主推" : "v-放弃"}`}>{t.hit1x2 ? "胜平负中" : "胜平负未中"}</span>
                  ) : null}
                  {t.hitOu != null ? (
                    <span className={`verdict ${t.hitOu ? "v-主推" : "v-放弃"}`}>{t.hitOu ? "大小中" : "大小未中"}</span>
                  ) : null}
                  {t.hitHc != null ? (
                    <span className={`verdict ${t.hitHc ? "v-主推" : "v-放弃"}`}>{t.hitHc ? "让球中" : "让球未中"}</span>
                  ) : null}
                  {t.hitScore != null ? (
                    <span className={`verdict ${t.hitScore ? "v-主推" : "v-放弃"}`}>{t.hitScore ? "比分中" : "比分未中"}</span>
                  ) : null}
                </div>
                <div className="expert-quickread">
                  <div><span>解盘结构</span><b>{t.pattern || "盘口与概率交叉验证"}</b></div>
                  <div><span>方向温度</span><b>{Math.max(t.homeWin, t.draw, t.awayWin).toFixed(0)}°</b><small>基于胜平负概率，仅作强弱参考</small></div>
                  <div><span>竞彩参考</span><b>{numStr ? `${numStr} ` : ""}{t.pickHandicap ? `${t.pickHandicap}${handicapSp(t.pickHandicap, odds)}` : t.pick1x2 ? t.pick1x2 : "待确认"}</b></div>
                </div>
                <div className="expert-analysis"><b>专业解盘</b><p>{t.plainTalk || "研判内容暂未生成。"}</p>{ticketInterpretation(t.pickHandicap, odds)}</div>
                {t.buyTalk ? (
                  <div className="buy-talk">
                    <b>{t.roleKey === "shape" ? "基准说明" : "参考买入"}</b>
                    <p>{t.roleKey === "shape" ? t.buyTalk : normalizeAdvice(t.buyTalk)}</p>
                    {t.roleKey === "shape" ? null : (
                    <span className="buy-risk">风险提示：本观点依赖当前盘口、阵容与市场信息，临场变化可能导致判断失效。请独立决策，相关风险由使用者自行承担。</span>
                    )}
                  </div>
                ) : null}
              </article>
              );
            })}
          </div>
        </>
      )}
    </>
  );
}

function TradePlan({ sn, cards, odds }: { sn: Snapshot; cards: ModelTake[]; odds?: OddsBoard | null }) {
  const active = cards.filter((t) => t.verdict !== "放弃");
  const pool = active.length ? active : cards;
  const pick = majority(pool.map((t) => t.pick1x2).filter(Boolean) as string[]);
  const ou = majority(pool.map((t) => t.pickOu).filter(Boolean) as string[]);
  const handicap = majority(pool.map((t) => t.pickHandicap).filter((x) => x && x !== "放弃") as string[]);
  const scores = cards.flatMap((t) => t.scores ?? []).filter(Boolean).slice(0, 2);
  const fallbackScores = (sn.topScores ?? []).slice(0, 2).map((x) => x.score);
  const scoreText = (scores.length ? scores : fallbackScores).join(" / ") || "待确认";
  const pattern = cards.find((t) => t.pattern)?.pattern || patternOf(sn);
  return (
    <section className="trade-plan">
      <div className="trade-plan-head"><span>专家共识摘要</span><b>{active.some((t) => t.verdict === "主推") ? "有专家给出主推" : "专家整体偏谨慎"}</b></div>
      <div className="trade-plan-grid">
        <div><span>多数格局</span><strong>{pattern}</strong></div>
        <div className="primary"><span>多数方向</span><strong>胜平负 {pick || "待确认"}</strong></div>
        <div><span>让球多数</span><strong>{handicap ? `${handicap}${handicapSp(handicap, odds)}` : sn.handicap?.pick ? `${sn.handicap.pick}${handicapSp(sn.handicap.pick, odds)}` : "待确认"}</strong></div>
        <div><span>大小球多数</span><strong>{ou ? `${ou} 2.5` : "待确认"}</strong></div>
        <div><span>情景比分</span><strong>{scoreText}</strong></div>
      </div>
      <p>这是四位专家独立判断的多数口径，可能与上方结构分布不一致。是否执行看第 04 层价值等级，不要把这里当成买入指令。</p>
      <div className="trade-risk">风险提示：以上是专家多数口径的摘要，不代表确定赛果或收益承诺；请独立判断并自行承担相关决策风险。</div>
    </section>
  );
}

function majority(values: string[]): string {
  const counts = new Map<string, number>();
  values.forEach((v) => counts.set(v, (counts.get(v) ?? 0) + 1));
  return [...counts].sort((a, b) => b[1] - a[1])[0]?.[0] || "";
}

function patternOf(sn: Snapshot): string {
  const side = sn.homeWin >= sn.awayWin ? "主队掌握更多主动权" : "客队反击空间更值得关注";
  return sn.over25 >= 52 ? `${side}，对攻倾向` : `${side}，节奏偏谨慎`;
}

function MarketBooks({ market }: { market: MarketQuote }) {
  const books = market.books ?? [];
  const ah = market.asianMove;
  const ou = market.ouMove;
  const asianBooks = market.asianBooks ?? [];
  const ouBooks = market.ouBooks ?? [];
  return (
    <div className="block">
      <div className="block-h">机构对照 · 初盘到即时</div>
      {books.length ? (
        <div className="book-table" role="table">
          <div className="book-row head" role="row">
            <span>公司</span><span>初赔 主/平/客</span><span>即时 主/平/客</span>
          </div>
          {books.map((b) => (
            <div className="book-row" role="row" key={b.companyId}>
              <span>{b.company}</span>
              <span>{fmtTrio(b.opening)}</span>
              <span>{fmtTrio(b.current)}</span>
            </div>
          ))}
        </div>
      ) : null}
      {asianBooks.length ? (
        <div className="book-table" role="table">
          <div className="book-row head" role="row">
            <span>亚盘</span><span>初盘</span><span>即时</span>
          </div>
          {asianBooks.map((r, i) => (
            <div className="book-row" role="row" key={`ah-${r.companyId || i}`}>
              <span>{r.company}</span>
              <span>{fmtLine(r.openingLine, r.openingLeft, r.openingRight)}</span>
              <span>{fmtLine(r.currentLine, r.currentLeft, r.currentRight)}</span>
            </div>
          ))}
        </div>
      ) : ah?.openingLine || ah?.currentLine ? (
        <p className="pred">澳门亚盘对照：初 {ah.openingLine} {fmtPair(ah.openingLeft, ah.openingRight)} → 即 {ah.currentLine} {fmtPair(ah.currentLeft, ah.currentRight)}。价值仍看上方 Bet365 亚盘。</p>
      ) : null}
      {ouBooks.length ? (
        <div className="book-table" role="table">
          <div className="book-row head" role="row">
            <span>大小</span><span>初盘</span><span>即时</span>
          </div>
          {ouBooks.map((r, i) => (
            <div className="book-row" role="row" key={`ou-${r.companyId || i}`}>
              <span>{r.company}</span>
              <span>{fmtLine(r.openingLine, r.openingLeft, r.openingRight)}</span>
              <span>{fmtLine(r.currentLine, r.currentLeft, r.currentRight)}</span>
            </div>
          ))}
        </div>
      ) : ou?.openingLine || ou?.currentLine ? (
        <p className="pred">澳门大小对照：初 {ou.openingLine} {fmtPair(ou.openingLeft, ou.openingRight)} → 即 {ou.currentLine} {fmtPair(ou.currentLeft, ou.currentRight)}。</p>
      ) : null}
      {asianBooks.length ? <p className="pred">亚盘价值仍看上方 Bet365；以上为多家机构盘路对照，给盘口解读用。</p> : null}
    </div>
  );
}

function fmtTrio(t?: { h?: number; d?: number; a?: number } | null): string {
  if (!t || !t.h) return "—";
  return `${t.h.toFixed(2)} / ${t.d?.toFixed(2)} / ${t.a?.toFixed(2)}`;
}

function fmtPair(a?: number, b?: number): string {
  if (a == null || b == null || a <= 0) return "";
  return `${a.toFixed(2)}/${b.toFixed(2)}`;
}

function fmtLine(line?: string, left?: number, right?: number): string {
  if (!line) return "—";
  const w = fmtPair(left, right);
  return w ? `${line} ${w}` : line;
}

function OddsMovement({ open, close }: { open: OddsBoard; close: OddsBoard }) {
  const moves = [
    { label: "主胜", from: open.had?.H, to: close.had?.H },
    { label: "平局", from: open.had?.D, to: close.had?.D },
    { label: "客胜", from: open.had?.A, to: close.had?.A },
  ];
  return (
    <div className="odds-movement">
      <div className="block-h">票面变化 · 赛前 → 临场</div>
      <div className="movement-grid">
        {moves.map((m) => <MoveCell key={m.label} {...m} />)}
        <div className="movement-cell"><span>让球线</span><b>{fmtSigned(open.hhadLine) || "—"} → {fmtSigned(close.hhadLine) || "—"}</b><em>{open.hhadLine === close.hhadLine ? "维持盘口" : "盘口调整"}</em></div>
        <MoveCell label="大球" from={open.over} to={close.over} />
        <MoveCell label="小球" from={open.under} to={close.under} />
      </div>
    </div>
  );
}

function TicketAdvice({ sn, odds }: { sn: Snapshot; odds?: OddsBoard | null }) {
  const had = [
    { label: "胜", p: sn.homeWin },
    { label: "平", p: sn.draw },
    { label: "负", p: sn.awayWin },
  ].sort((a, b) => b.p - a.p);
  const hadGap = had[0].p - had[1].p;
  const hadText = hadGap >= 10 ? `重心 ${had[0].label}` : `重心 ${had[0].label}，次档 ${had[1].label}`;
  const hhad = [
    { label: "让胜", p: odds?.hhadMarketH ?? 0 },
    { label: "让平", p: odds?.hhadMarketD ?? 0 },
    { label: "让负", p: odds?.hhadMarketA ?? 0 },
  ].sort((a, b) => b.p - a.p);
  const hhadReady = Boolean(odds?.hhadLine && odds?.hhad?.H > 1 && hhad[0].p > 0);
  const handicap = fmtSigned(odds?.hhadLine);
  const handicapSpRows = [
    { label: "让胜", price: odds?.hhad?.H },
    { label: "让平", price: odds?.hhad?.D },
    { label: "让负", price: odds?.hhad?.A },
  ];
  const goalPicks = totalGoalPicks(sn.grid ?? []);
  const scorePicks = (sn.topScores ?? []).slice(0, 2);

  return (
    <section className="ticket-advice" aria-label="竞彩基础参考">
      <div className="ticket-advice-head">
        <div><span>SPORTTERY BASICS</span><h3>竞彩基础参考</h3></div>
        <b>对照玩法，不是买入</b>
      </div>
      <div className="ticket-advice-grid">
        <div className="ticket-pick primary">
          <span>胜平负</span>
          <strong>{hadText}</strong>
          <small>{had[0].label} {had[0].p.toFixed(1)}% · {had[1].label} {had[1].p.toFixed(1)}%</small>
        </div>
        <div className="ticket-pick">
          <span>让球胜平负 {handicap || ""}</span>
          <strong>{hhadReady ? `重心 ${hhad[0].label}${hhad[0].p - hhad[1].p < 8 ? `，次档 ${hhad[1].label}` : ""}` : "票面未齐，暂不对照"}</strong>
          <small>{hhadReady ? <>SP {handicapSpRows.map((x) => `${x.label} ${fmtSp(x.price)}`).join(" · ")} · 让球数按主队口径</> : "等待官方让球数与 SP 完整更新"}</small>
        </div>
        <div className="ticket-pick">
          <span>总进球</span>
          <strong>{goalPicks.length ? `对照 ${goalPicks.map((x) => x.goals).join(" / ")} 球` : "分布不足，暂不对照"}</strong>
          <small>{goalPicks.length ? goalPicks.map((x) => `${x.goals}球 ${x.p.toFixed(1)}%`).join(" · ") : "按竞彩 0–7+ 口径统计"}</small>
        </div>
        <div className="ticket-pick">
          <span>比分</span>
          <strong>{scorePicks.length ? `路径 ${scorePicks.map((x) => x.score).join(" / ")}` : "分布不足，暂不对照"}</strong>
          <small>高波动玩法，仅用于辅助理解比赛路径</small>
        </div>
      </div>
      <div className="ticket-rule-note">
        <b>怎么读</b>
        <span>这里只是把结构分布对照到竞彩玩法，方便核对官方怎么卖。它不是专家结论，也不是价值等级。总进球与比分波动更高，只作路径理解。</span>
      </div>
      <div className="ticket-risk">风险提示：以上为当前票面与概率分布下的基础参考，不构成投注指令、赛果或收益保证。请以实际出票规则和票面为准，独立决策，相关风险与结果由使用者自行承担。</div>
    </section>
  );
}

function totalGoalPicks(grid: number[][]): Array<{ goals: string; p: number }> {
  const totals = Array.from({ length: 8 }, () => 0);
  grid.forEach((row, home) => row.forEach((p, away) => {
    totals[Math.min(7, home + away)] += Number(p) || 0;
  }));
  return totals
    .map((p, goals) => ({ goals: goals === 7 ? "7+" : String(goals), p }))
    .filter((x) => x.p > 0)
    .sort((a, b) => b.p - a.p)
    .slice(0, 2);
}

function MoveCell({ label, from, to }: { label: string; from?: number; to?: number }) {
  const valid = !!from && !!to;
  const delta = valid ? (to as number) - (from as number) : 0;
  return <div className="movement-cell"><span>{label}</span><b>{fmtPrice(from)} → {fmtPrice(to)}</b><em className={delta < 0 ? "down" : delta > 0 ? "up" : ""}>{!valid ? "数据待齐" : delta < 0 ? `下调 ${Math.abs(delta).toFixed(2)}` : delta > 0 ? `上调 ${delta.toFixed(2)}` : "持平"}</em></div>;
}

function roleColor(role?: string): string {
  if (role === "shape") return "#475569";
  if (role === "market") return "#0b9668";
  if (role === "goals") return "#d28b16";
  if (role === "lineup") return "#3b82f6";
  if (role === "value") return "#dc5267";
  return "var(--muted)";
}

function roleMeta(role?: string): { focus: string } {
  if (role === "shape") return { focus: "欧赔重心 · 让球分布 · 进球路径" };
  if (role === "market") return { focus: "欧亚盘联动 · 水位变化 · 成交热度" };
  if (role === "goals") return { focus: "比赛节奏 · 进球路径 · 大小球" };
  if (role === "lineup") return { focus: "阵型对位 · 首发结构 · 伤停影响" };
  if (role === "value") return { focus: "市场定价 · 价格保护 · 执行等级" };
  return { focus: "多维数据交叉研判" };
}

function roleTitle(role?: string, fallback?: string): string {
  if (role === "shape") return "基本盘分析师";
  if (role === "market") return "盘口分析师";
  if (role === "goals") return "进球分析师";
  if (role === "lineup") return "阵容分析师";
  if (role === "value") return "价值研判师";
  if (fallback === "价值猎手") return "价值研判师";
  if (fallback === "进球专家") return "进球分析师";
  if (fallback === "盘口专家") return "盘口分析师";
  if (fallback === "阵容专家") return "阵容分析师";
  return fallback || "专业研判";
}

function normalizeAdvice(text: string): string {
  return text.replace(/^参考买入[：:]?\s*/, "").replace(/^竞彩/, "");
}

function fmtSp(value?: number | null): string {
  return value != null && value > 1 ? value.toFixed(2) : "待开售";
}

function handicapSp(pick?: string, odds?: OddsBoard | null): string {
  const raw = (pick || "").replace(/^主选\s*/, "").trim();
  if (!raw || raw === "放弃") return "";
  const key = raw.startsWith("让") ? raw : `让${raw}`;
  const price = key === "让胜" ? odds?.hhad?.H : key === "让平" ? odds?.hhad?.D : key === "让负" ? odds?.hhad?.A : undefined;
  return ` · SP ${fmtSp(price)}`;
}

function ticketInterpretation(pick?: string, odds?: OddsBoard | null) {
  const line = fmtSigned(odds?.hhadLine);
  const ready = Boolean(line && Number(odds?.hhad?.H) > 1 && Number(odds?.hhad?.D) > 1 && Number(odds?.hhad?.A) > 1);
  if (!ready) return <p className="ticket-interpretation muted"><b>让球票面：</b>官方让球数或 SP 尚未完整开售，当前不补充让胜、让平、让负结论。</p>;
  const direction = pick && pick !== "放弃" ? (pick.startsWith("让") ? pick : `让${pick}`) : "暂不介入";
  return <p className="ticket-interpretation"><b>让球票面：</b>主队 {line}，当前参考 {direction}{direction !== "暂不介入" ? handicapSp(direction, odds) : ""}；完整 SP 为让胜 {fmtSp(odds?.hhad?.H)}、让平 {fmtSp(odds?.hhad?.D)}、让负 {fmtSp(odds?.hhad?.A)}。SP 反映官方票面定价，仅用于选择玩法，不能单独视为赛果依据。</p>;
}

function RiskNotice({ children, tone, strong }: { children: ReactNode; tone?: "data"; strong?: boolean }) {
  return <div className={`risk-notice${tone === "data" ? " data" : ""}${strong ? " strong" : ""}`}><b>{tone === "data" ? "数据说明" : "风险提示"}</b><span>{children}</span></div>;
}

function Layer({ id, n, title, hint, children }: { id: string; n: string; title: string; hint: string; children: ReactNode }) {
  return (
    <section className="layer" id={id}>
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

function shapeHeadline(sn: Snapshot): string {
  const sides = [
    { n: "主胜", p: sn.homeWin },
    { n: "平局", p: sn.draw },
    { n: "客胜", p: sn.awayWin },
  ].sort((a, b) => b.p - a.p);
  const gap = sides[0].p - sides[1].p;
  const strength = gap >= 18 ? "格局较清晰" : gap >= 8 ? "略有倾向" : "双方胶着";
  const goals = sn.over25 >= 52 ? "进球略偏多" : sn.over25 <= 48 ? "进球略偏少" : "大小相对均衡";
  return `${sides[0].n}更常见，${strength}，${goals}`;
}

function basicTalk(sn: Snapshot, now?: OddsBoard | null): string {
  const parts: string[] = [];
  const hc = fmtSigned(now?.hhadLine);
  if (hc) parts.push(`竞彩让球是${hc}`);
  if (now && now.over > 1 && now.under > 1) parts.push("竞彩大小看 2.5");
  if (sn.topScores?.[0]) parts.push(`相对更常见的比分路径是 ${sn.topScores[0].score}`);
  return parts.length ? `${parts.join("。")}。` : "";
}

function ValueMatrix({ had, ou, asia, picked }: { had: GateSide[]; ou: GateSide[]; asia: GateSide[]; picked: EvalSide[] }) {
  const all = [...had, ...ou, ...asia];
  const candidates = all.filter((x) => x.picked);
  const best = [...candidates].sort((a, b) => rank(b.verdict) - rank(a.verdict) || b.value - a.value)[0];
  const protectedMarket = best ? best.kellyBand !== "紧" : false;
  const priceMatch = best ? best.value >= 3 : false;
  return (
    <section className="value-matrix">
      <div className="matrix-head">
        <div><span>JC VALUE MATRIX</span><h3>专属价值研判结果</h3><p>综合比赛概率、机构态度、资金分布与临场变化。仅展示结论，不公开权重、阈值及组合逻辑。</p></div>
        <div className={`matrix-verdict v-${best?.verdict || "放弃"}`}><small>综合等级</small><strong>{best?.verdict ? <>{verdictLabel(best.verdict)} <VerdictHelp verdict={best.verdict} /></> : "等待数据"}</strong></div>
      </div>
      <div className="verdict-guide" aria-label="价值等级说明">
        <div className="main"><span>主推 <VerdictHelp verdict="主推" /></span><b>方向与价格共振</b><small>多项关键信号一致，当前具备优先执行条件</small></div>
        <div className="watch"><span>可看 <VerdictHelp verdict="可看" /></span><b>方向成立，条件未齐</b><small>保留关注，等待更好价格或临场信号确认</small></div>
        <div className="avoid"><span>回避 <VerdictHelp verdict="回避" /></span><b>当前不具备价值</b><small>价格、保护或拥挤风险不利，不建议执行</small></div>
      </div>
      <div className="matrix-signals">
        <div><span>价格匹配度</span><b>{priceMatch ? "具备空间" : "空间有限"}</b><i className={priceMatch ? "good" : "mid"} /></div>
        <div><span>市场保护</span><b>{protectedMarket ? "信号稳定" : "保护不足"}</b><i className={protectedMarket ? "good" : "bad"} /></div>
        <div><span>拥挤风险</span><b>{best?.hot ? "需要防热" : "风险可控"}</b><i className={best?.hot ? "bad" : "good"} /></div>
      </div>
      <div className="matrix-output">
        <div><span>优先方向</span><strong>{best?.label || "等待市场数据"}</strong></div>
        <div><span>候选方向</span><strong>{picked.map((s) => s.label).join(" · ") || "暂未形成"}</strong></div>
        <div><span>执行提示</span><strong>{best?.verdict === "主推" ? "价格与方向共振，可优先考虑" : best?.verdict === "可看" ? "方向成立，等待更好价格" : "定价不利，建议回避"}</strong></div>
      </div>
      <div className="matrix-cards">
        <PrivateMarket title="胜平负" sides={had} />
        <PrivateMarket title="大小球" sides={ou} />
        <PrivateMarket title="亚洲盘" sides={asia} />
      </div>
      <div className="matrix-foot"><span>PROPRIETARY ANALYSIS</span> 公开基础指标；专属权重、冲突裁决和综合评级逻辑不公开。</div>
    </section>
  );
}

function PrivateMarket({ title, sides }: { title: string; sides: GateSide[] }) {
  const ordered = [...sides].sort((a, b) => rank(b.verdict) - rank(a.verdict) || b.value - a.value);
  return <div className="private-market"><span>{title}</span>{ordered.length ? ordered.map((s) => <div key={s.key}><b>{s.label}</b><em className={`v-${s.verdict}`}>{verdictLabel(s.verdict)} <VerdictHelp verdict={s.verdict} /></em><small>{s.hot ? "市场偏热" : s.verdict === "主推" ? "价值共振" : "等待确认"}</small></div>) : <p>市场数据采集中</p>}</div>;
}

function rank(v: GateSide["verdict"]): number {
  if (v === "主推") return 2;
  if (v === "可看") return 1;
  return 0;
}
