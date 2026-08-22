import { useState } from "react";
import { IconHelpCircle } from "./Icons";

export type VerdictLevel = "主推" | "关注" | "谨慎" | "可看" | "观望" | "放弃" | "回避" | string;

const EXPLANATIONS: Record<string, string> = {
  主推: "多项关键信号一致，当前具备优先参考条件，但不代表确定赛果。",
  谨慎: "可以留意，不要急着买。方向有依据，但价格或条件还不够。",
  回避: "当前价格、保护或拥挤风险不利，不建议作为参考买入方向。",
};

export function verdictLabel(verdict: VerdictLevel): string {
  if (verdict === "放弃") return "回避";
  if (verdict === "可看" || verdict === "观望" || verdict === "关注") return "谨慎";
  return verdict;
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
  const label = verdictLabel(verdict);
  return (
    <span className="verdict-pair">
      <span className={`verdict v-${label}`}>{label}</span>
      <VerdictHelp verdict={verdict} />
    </span>
  );
}
