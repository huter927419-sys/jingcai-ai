import type { RiskHint } from "./api";

const TRIAL_UNTIL = "2026-08-27";

export function trialUntilLabel(until?: string): string {
  const day = (until || TRIAL_UNTIL).slice(0, 10);
  const [y, m, d] = day.split("-");
  if (!y || !m || !d) return "下周四";
  return `${Number(m)}月${Number(d)}日`;
}

export function trialActive(until?: string, now = new Date()): boolean {
  const day = (until || TRIAL_UNTIL).slice(0, 10);
  const end = new Date(`${day}T23:59:59+08:00`);
  return now.getTime() <= end.getTime();
}

export function UpsetHintList({
  hints,
  until,
  compact,
}: {
  hints?: RiskHint[] | null;
  until?: string;
  compact?: boolean;
}) {
  if (!hints?.length) return null;
  const active = trialActive(until);
  const when = trialUntilLabel(until);
  if (compact) {
    return (
      <div className="upset-hint-chips">
        {hints.map((h) => (
          <span key={h.key} title={h.detail}>{h.title}</span>
        ))}
      </div>
    );
  }
  return (
    <section className="upset-hints" aria-label="冷门风险提示">
      <div className="upset-hints-head">
        <span>冷门风险提示</span>
        <b>{active ? `试验观察至${when}` : "试验已截止"}</b>
      </div>
      <ul>
        {hints.map((h) => (
          <li key={h.key}>
            <strong>{h.title}</strong>
            <p>{h.detail}</p>
          </li>
        ))}
      </ul>
      <p className="upset-hints-note">
        只标风险，不改专家选项，也不改价值档。出现提示不代表会出冷门；{active ? `先观察到${when}，再按完场对账看准不准。` : "对账样本将另行汇总。"}
      </p>
    </section>
  );
}
