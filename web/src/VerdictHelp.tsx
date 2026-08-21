import { useState } from "react";
import { IconHelpCircle } from "./Icons";

export type VerdictLevel = "主推" | "可看" | "放弃" | "回避" | string;

const EXPLANATIONS: Record<string, string> = {
  主推: "多项关键信号一致，当前具备优先参考条件，但不代表确定赛果。",
  可看: "方向有依据但条件未完全确认，建议继续观察临场价格、阵容或盘口。",
  回避: "当前价格、保护或拥挤风险不利，不建议作为参考买入方向。",
};

export function verdictLabel(verdict: VerdictLevel): string {
  return verdict === "放弃" ? "回避" : verdict;
}

export function VerdictHelp({ verdict }: { verdict: VerdictLevel }) {
  const [open, setOpen] = useState(false);
  const label = verdictLabel(verdict);
  const explanation = EXPLANATIONS[label];
  if (!explanation) return null;
  return (
    <span className={`verdict-help${open ? " open" : ""}`} onBlur={(event) => {
      if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false);
    }}>
      <button type="button" aria-label={`解释${label}`} aria-expanded={open} onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        setOpen((value) => !value);
      }}>
        <IconHelpCircle size={13} />
      </button>
      <span className="verdict-tooltip" role="tooltip">
        <b>{label}</b>
        {explanation}
      </span>
    </span>
  );
}

export function VerdictBadge({ verdict }: { verdict: VerdictLevel }) {
  return (
    <span className="verdict-pair">
      <span className={`verdict v-${verdict}`}>{verdictLabel(verdict)}</span>
      <VerdictHelp verdict={verdict} />
    </span>
  );
}
