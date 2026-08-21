export type MatchRow = {
  id: number;
  numStr: string;
  league: string;
  leagueAbb: string;
  home: string;
  away: string;
  kickoff: string;
  businessDate: string;
  hasOpen: boolean;
  hasClose: boolean;
  homeGoals?: number | null;
  awayGoals?: number | null;
  finished?: boolean;
  status: string;
  kind: string;
  odds?: OddsBoard;
  market?: MarketQuote;
  shape?: { homeWin: number; draw: number; awayWin: number; over25: number };
};

export type EvalSide = {
  key: string;
  label: string;
  model?: number;
  market?: number;
  odds?: number;
  value: number;
  valueBand?: "有价值" | "边缘" | "无价值" | string;
  kelly: number;
  kellyBand: "松" | "中" | "紧" | string;
};

export type OddsBoard = {
  had: { H: number; D: number; A: number };
  hasHad?: boolean;
  marketH: number;
  marketD: number;
  marketA: number;
  hhadLine: string;
  hhadText: string;
  hhad: { H: number; D: number; A: number };
  hhadMarketH?: number;
  hhadMarketD?: number;
  hhadMarketA?: number;
  over: number;
  under: number;
  marketOver: number;
  marketUnder: number;
};

export type MarketQuote = {
  company?: string;
  eu?: { h: number; d: number; a: number; pH: number; pD: number; pA: number; company?: string };
  asian?: { line: string; lineNum: number; home: number; away: number; pH: number; pA: number; company?: string };
  ou?: { line: number; over: number; under: number; pO: number; pU: number; company?: string };
  betfair?: { homeVol: number; drawVol: number; awayVol: number; total: number; thin: boolean; note?: string };
};

export type PlayRow = {
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

export function fmtSigned(n?: number | string | null): string {
  const v = typeof n === "string" ? Number.parseFloat(n) : n;
  if (v == null || Number.isNaN(v)) return "";
  if (Math.abs(v) < 1e-9) return "0";
  const abs = Math.abs(v);
  const body = abs === Math.floor(abs) ? String(abs) : String(Math.round(abs * 100) / 100);
  return (v > 0 ? "+" : "-") + body;
}

export function playRows(odds?: OddsBoard | null): PlayRow[] {
  const std: PlayRow = { key: "std", label: "标准" };
  if (odds?.hasHad && odds.had?.H > 1) {
    std.h = odds.had.H;
    std.d = odds.had.D;
    std.a = odds.had.A;
    std.pH = odds.marketH;
    std.pD = odds.marketD;
    std.pA = odds.marketA;
    std.note = "竞彩";
  } else {
    std.empty = "未开售";
  }
  const hcNum = fmtSigned(odds?.hhadLine);
  const hc: PlayRow = { key: "hc", label: hcNum ? `让球 ${hcNum}` : "让球" };
  if (odds?.hhadLine && odds.hhad?.H > 1) {
    hc.h = odds.hhad.H;
    hc.d = odds.hhad.D;
    hc.a = odds.hhad.A;
    hc.pH = odds.hhadMarketH;
    hc.pD = odds.hhadMarketD;
    hc.pA = odds.hhadMarketA;
    hc.note = "竞彩";
  } else {
    hc.empty = "—";
  }
  const ou: PlayRow = { key: "ou", label: "大小 2.5", twoWay: true, ends: ["大", "小"] };
  if (odds && odds.over > 1 && odds.under > 1) {
    ou.h = odds.over;
    ou.a = odds.under;
    ou.pH = odds.marketOver;
    ou.pA = odds.marketUnder;
    ou.note = "竞彩";
  } else {
    ou.empty = "未开售";
  }
  return [std, hc, ou];
}

export type ScoreProb = { home: number; away: number; score: string; p: number };

export type Snapshot = {
  kind: "open" | "close";
  fetchedAt: string;
  headline: string;
  plainTalk: string;
  homeWin: number;
  draw: number;
  awayWin: number;
  over25: number;
  under25: number;
  topScores: ScoreProb[];
  grid: number[][];
  eval?: EvalSide[];
  handicap?: {
    line: string;
    lineText: string;
    pick: string;
    talk: string;
    home: number;
    draw: number;
    away: number;
    sides: EvalSide[];
  };
  odds?: OddsBoard;
  market?: MarketQuote;
  usedAI?: boolean;
  usedModels?: string[];
  takes?: ModelTake[];
};

export type ModelTake = {
  name: string;
  role?: string;
  roleKey?: string;
  headline: string;
  plainTalk: string;
  buyTalk?: string;
  pattern?: string;
  scores?: string[];
  pickHandicap?: string;
  homeWin: number;
  draw: number;
  awayWin: number;
  over25: number;
  under25: number;
  pick1x2?: string;
  pickOu?: string;
  verdict?: string;
  hit1x2?: boolean;
  hitOu?: boolean;
};

export type ExpertBoardRow = {
  name: string;
  role: string;
  roleKey: string;
  games: number;
  hit1x2: number;
  hitOu: number;
  rate1x2: number;
  rateOu: number;
  points: number;
};

export type SettledItem = {
  match: MatchRow;
  score: string;
  takes: ModelTake[];
};

export async function fetchExperts(): Promise<{ board: ExpertBoardRow[]; yesterday: SettledItem[]; settled: SettledItem[]; pending: number }> {
  const r = await fetch("/api/experts");
  if (!r.ok) throw new Error("experts failed");
  const j = await r.json();
  return { board: j.board ?? [], yesterday: j.yesterday ?? [], settled: j.settled ?? [], pending: Number(j.pending ?? 0) };
}

export type MatchDetail = {
  match: MatchRow;
  available: Array<"open" | "close">;
  status: string;
  source: string;
  snapshot?: Snapshot;
  oddsOpen?: OddsBoard | null;
  oddsClose?: OddsBoard | null;
  market?: MarketQuote | null;
  preview?: TeamPreview | null;
  expertKind?: "open" | "close";
};

export type PlayerXI = { no?: string; name: string; pos?: string };
export type RecentMatch = {
  date: string;
  league: string;
  home: string;
  away: string;
  score: string;
  result: string;
  rating: number;
};
export type SidePreview = {
  name: string;
  formation: string;
  starters: PlayerXI[];
  bench: PlayerXI[];
  form: RecentMatch[];
  avgRating: number;
};
export type TeamPreview = {
  matchId?: number;
  fid?: number;
  home: SidePreview;
  away: SidePreview;
};

export function valueBandOf(v: number, band?: string): string {
  if (band) return band;
  if (v >= 3) return "有价值";
  if (v >= 0) return "边缘";
  return "无价值";
}

export async function fetchToday(): Promise<{ matches: MatchRow[]; finished: number }> {
  const r = await fetch("/api/today");
  if (!r.ok) throw new Error("today failed");
  const j = await r.json();
  const matches = ((j.matches ?? []) as MatchRow[]).filter((m) => !m.finished && m.status !== "完场");
  return { matches, finished: Number(j.finished ?? 0) };
}

export type SFCMatch = {
  no: number;
  league: string;
  kickoff: string;
  home: string;
  away: string;
  homeWin: number;
  draw: number;
  awayWin: number;
  marketHome?: number;
  marketDraw?: number;
  marketAway?: number;
  pick: "胜" | "平" | "负" | string;
  source: "研判" | "均赔" | string;
  matchId?: number;
  numStr?: string;
  talk?: string;
  market?: MarketQuote;
  handicap?: {
    line?: string;
    lineText?: string;
    pick?: string;
    talk?: string;
    home?: number;
    away?: number;
  };
  eval?: EvalSide[];
};

export async function fetchSFC(): Promise<{ issue: string; matches: SFCMatch[]; analyzed: number; total: number }> {
  const r = await fetch("/api/sfc");
  if (!r.ok) throw new Error("sfc failed");
  const j = await r.json();
  return {
    issue: String(j.issue ?? ""),
    matches: (j.matches ?? []) as SFCMatch[],
    analyzed: Number(j.analyzed ?? 0),
    total: Number(j.total ?? 0),
  };
}

export async function fetchMatch(id: string, kind?: string): Promise<MatchDetail> {
  const q = kind ? `?kind=${kind}` : "";
  const r = await fetch(`/api/matches/${id}${q}`);
  if (!r.ok) throw new Error("match failed");
  return r.json();
}

export function fmtKick(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function fmtOdd(v?: number): string {
  return v && v > 1 ? v.toFixed(2) : "—";
}

export function fmtPrice(v?: number, water?: boolean): string {
  if (!v || v <= 0) return "—";
  if (water || v < 1.01) return v.toFixed(2);
  return v.toFixed(2);
}

export function fmtValue(v: number): string {
  const n = v.toFixed(1);
  return (v > 0 ? `+${n}` : n) + "%";
}

export function fmtVol(v?: number): string {
  if (!v) return "—";
  if (v >= 10000) return `${(v / 10000).toFixed(1)}万`;
  return String(Math.round(v));
}

export type MatchSort = "num" | "kick";

export type WeekMatch = MatchRow & {
  analysisCount: number;
  hasMarket: boolean;
  hasPreview: boolean;
};

export async function fetchWeek(): Promise<{ from: string; to: string; total: number; matches: WeekMatch[] }> {
  const r = await fetch("/api/week");
  if (!r.ok) throw new Error("week failed");
  const j = await r.json();
  return { from: j.from, to: j.to, total: Number(j.total ?? 0), matches: j.matches ?? [] };
}

function seqNo(numStr: string): number {
  const m = numStr.match(/(\d+)\s*$/);
  return m ? Number.parseInt(m[1], 10) : 0;
}

export function sortMatches(rows: MatchRow[], by: MatchSort): MatchRow[] {
  return [...rows].sort((a, b) => {
    if (by === "kick") {
      const t = new Date(a.kickoff).getTime() - new Date(b.kickoff).getTime();
      if (t) return t;
      return seqNo(a.numStr) - seqNo(b.numStr);
    }
    const day = (a.businessDate || "").localeCompare(b.businessDate || "");
    if (day) return day;
    const n = seqNo(a.numStr) - seqNo(b.numStr);
    if (n) return n;
    return a.numStr.localeCompare(b.numStr, "zh");
  });
}
