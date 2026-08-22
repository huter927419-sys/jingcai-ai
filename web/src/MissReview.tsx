import type { MissReview } from "./api";

export function MissReviewCard({ review }: { review?: MissReview | null }) {
  if (!review?.plainTalk) return null;
  return (
    <section className="miss-review" aria-label="赛后复盘">
      <div className="miss-review-head">
        <span>复盘分析师</span>
        <b>{review.kind || "胜平负看错"}</b>
      </div>
      {review.headline ? <h3>{review.headline}</h3> : null}
      <p>{review.plainTalk}</p>
      {review.visibleBefore?.length ? (
        <div className="miss-review-list">
          <span>事前能看到</span>
          <ul>{review.visibleBefore.map((x) => <li key={x}>{x}</li>)}</ul>
        </div>
      ) : null}
      {review.overread?.length ? (
        <div className="miss-review-list">
          <span>可能读过头</span>
          <ul>{review.overread.map((x) => <li key={x}>{x}</li>)}</ul>
        </div>
      ) : null}
      {review.lesson ? <p className="miss-review-lesson">{review.lesson}</p> : null}
      <p className="miss-review-note">这是完场后的归因，不改赛前专家选项，也不构成下一次买入建议。</p>
    </section>
  );
}
