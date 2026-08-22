package scheduler

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"jingcai-ai/internal/analyze"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
	"jingcai-ai/internal/titan007"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Store    *store.Store
	Client   *sporttery.Client
	Market   *market.Client
	Titan    *titan007.Client
	Engine   *analyze.Engine
	Location *time.Location

	mu     sync.Mutex
	closes map[int64]chan struct{}
	cron   *cron.Cron
	sfcMu  sync.Mutex
}

func New(st *store.Store, cl *sporttery.Client, mk *market.Client, titan *titan007.Client, eng *analyze.Engine, loc *time.Location) *Scheduler {
	c := cron.New(cron.WithLocation(loc), cron.WithSeconds())
	return &Scheduler{
		Store:    st,
		Client:   cl,
		Market:   mk,
		Titan:    titan,
		Engine:   eng,
		Location: loc,
		closes:   map[int64]chan struct{}{},
		cron:     c,
	}
}

func (s *Scheduler) Start() error {
	if _, err := s.cron.AddFunc("0 0 10 * * *", func() {
		if err := s.DailyOpen(); err != nil {
			log.Printf("daily open: %v", err)
		}
	}); err != nil {
		return err
	}
	if _, err := s.cron.AddFunc("0 */10 * * * *", func() {
		if err := s.PollScores(); err != nil {
			log.Printf("scores: %v", err)
		}
	}); err != nil {
		return err
	}
	if _, err := s.cron.AddFunc("0 */20 * * * *", func() {
		if err := s.BackfillNew(); err != nil {
			log.Printf("backfill: %v", err)
		}
		if err := s.RefreshSFCAndExperts(); err != nil {
			log.Printf("sfc: %v", err)
		}
	}); err != nil {
		return err
	}
	if _, err := s.cron.AddFunc("0 5 12 * * *", func() {
		if err := s.ReviewMisses(); err != nil {
			log.Printf("miss review: %v", err)
		}
	}); err != nil {
		return err
	}
	s.cron.Start()
	go func() {
		if err := s.Store.PruneOlderThan(time.Now().Add(-14 * 24 * time.Hour)); err != nil {
			log.Printf("prune: %v", err)
		}
		if err := s.PollScores(); err != nil {
			log.Printf("scores: %v", err)
		}
		if err := s.BackfillWeekReview(); err != nil {
			log.Printf("week review: %v", err)
		}
		if err := s.Resume(); err != nil {
			log.Printf("resume: %v", err)
		}
		if err := s.BackfillExperts(); err != nil {
			log.Printf("experts: %v", err)
		}
		if err := s.RefreshSFCAndExperts(); err != nil {
			log.Printf("sfc: %v", err)
		}
		if err := s.ReviewMisses(); err != nil {
			log.Printf("miss review: %v", err)
		}
	}()
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, ch := range s.closes {
		close(ch)
		delete(s.closes, id)
	}
}

func (s *Scheduler) ReviewMisses() error {
	if s.Engine == nil {
		return nil
	}
	now := time.Now()
	if s.Location != nil {
		now = now.In(s.Location)
	}
	return s.Engine.ReviewMisses(now)
}

func (s *Scheduler) DailyOpen() error {
	return s.ingest(store.KindOpen, false)
}

func (s *Scheduler) BackfillNew() error {
	return s.ingest(store.KindOpen, true)
}

func (s *Scheduler) RefreshNow() error {
	return s.ingest(store.KindOpen, true)
}

func (s *Scheduler) RefreshSFC() error {
	return s.refreshSFC(false)
}

func (s *Scheduler) RefreshSFCAndExperts() error {
	return s.refreshSFC(true)
}

func (s *Scheduler) refreshSFC(runExperts bool) error {
	if s.Market == nil {
		return nil
	}
	board, err := s.Market.FetchSFC()
	if err != nil {
		return err
	}
	ids := s.fillSFCAnalysis(board)
	if err := s.Store.SaveSFC(board); err != nil {
		return err
	}
	n := 0
	for _, m := range board.Matches {
		if m.AnalyzedHome+m.AnalyzedDraw+m.AnalyzedAway > 1 {
			n++
		}
	}
	log.Printf("sfc issue %s matches %d analyzed %d experts %d", board.Issue, len(board.Matches), n, len(ids))
	if runExperts && len(ids) > 0 && s.Engine != nil {
		go s.completeSFC(ids)
	}
	return nil
}

func (s *Scheduler) completeSFC(ids []int64) {
	if !s.sfcMu.TryLock() {
		return
	}
	defer s.sfcMu.Unlock()
	seen := map[int64]bool{}
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		if err := s.Engine.CompleteSFC(id); err != nil {
			log.Printf("sfc experts %d: %v", id, err)
		}
	}
}

func (s *Scheduler) fillSFCAnalysis(board *market.SFCBoard) []int64 {
	if board == nil {
		return nil
	}
	prev, _ := s.Store.LatestSFC()
	prevByNo := map[int]market.SFCMatch{}
	if prev != nil && prev.Issue == board.Issue {
		for _, m := range prev.Matches {
			prevByNo[m.No] = m
		}
	}
	now := time.Now()
	if s.Location != nil {
		now = now.In(s.Location)
	}
	var ids []int64
	for i := range board.Matches {
		row := &board.Matches[i]
		if p, ok := prevByNo[row.No]; ok {
			row.AnalyzedHome, row.AnalyzedDraw, row.AnalyzedAway = p.AnalyzedHome, p.AnalyzedDraw, p.AnalyzedAway
			if row.EUHome <= 1 {
				row.EUHome, row.EUDraw, row.EUAway = p.EUHome, p.EUDraw, p.EUAway
			}
			if row.Quote == nil {
				row.Quote = p.Quote
			}
		}
		if m := s.Store.MatchSFC(*row, now); m != nil && m.Origin != "sfc" {
			continue
		}
		needMarkets := row.Fid > 0 && (row.Quote == nil || (row.Quote.Asian == nil && row.Quote.EU == nil))
		if needMarkets {
			if q, err := s.Market.FetchMarkets(row.Fid); err == nil && q != nil {
				analyze.ApplyQuote(row, q)
			} else if err != nil {
				log.Printf("sfc markets %d: %v", row.No, err)
			}
			time.Sleep(250 * time.Millisecond)
		} else if row.Quote != nil {
			analyze.ApplyQuote(row, row.Quote)
		}
		h, d, a := row.EUHome, row.EUDraw, row.EUAway
		if row.AnalyzedHome+row.AnalyzedDraw+row.AnalyzedAway <= 1 && h > 1 && d > 1 && a > 1 {
			res := analyze.ProbsFrom1X2(h, d, a)
			row.AnalyzedHome, row.AnalyzedDraw, row.AnalyzedAway = res.HomeWin, res.Draw, res.AwayWin
		}
		if s.Engine == nil {
			continue
		}
		m := sfcMatch(board.Issue, *row, s.Location)
		if err := s.Store.UpsertSFCMatch(m); err != nil {
			log.Printf("sfc upsert %d: %v", row.No, err)
			continue
		}
		if row.Quote != nil {
			q := *row.Quote
			q.MatchID = m.ID
			if err := s.Store.SaveQuote(&q); err != nil {
				log.Printf("sfc quote %d: %v", row.No, err)
			}
		}
		if prev, _ := s.Store.GetPreview(m.ID); prev == nil && row.Fid > 0 && s.Market != nil {
			if p, err := s.Market.FetchPreview(m.ID, row.Fid); err == nil && p != nil {
				_ = s.Store.SavePreview(p)
			} else if err != nil {
				log.Printf("sfc preview %d: %v", row.No, err)
			}
			time.Sleep(250 * time.Millisecond)
		}
		if err := s.Engine.SeedFromMarket(m); err != nil {
			log.Printf("sfc seed %d: %v", row.No, err)
			continue
		}
		ids = append(ids, m.ID)
		if sn, err := s.Store.PreferredSnapshot(m.ID); err == nil && sn != nil && sn.Result.HomeWin+sn.Result.Draw+sn.Result.AwayWin > 1 {
			row.AnalyzedHome, row.AnalyzedDraw, row.AnalyzedAway = sn.Result.HomeWin, sn.Result.Draw, sn.Result.AwayWin
		}
	}
	return ids
}

func sfcMatch(issue string, row market.SFCMatch, loc *time.Location) sporttery.Match {
	kick := parseSFCKick(row.Kickoff, loc)
	abb := strings.TrimSpace(row.League)
	rs := []rune(abb)
	if len(rs) > 2 {
		abb = string(rs[:2])
	}
	m := sporttery.Match{
		ID:           store.SFCMatchID(issue, row.No),
		NumStr:       fmt.Sprintf("胜负%02d", row.No),
		League:       row.League,
		LeagueAbb:    abb,
		Home:         row.Home,
		Away:         row.Away,
		Kickoff:      kick,
		BusinessDate: kick.Format("2006-01-02"),
	}
	if row.EUHome > 1 && row.EUDraw > 1 && row.EUAway > 1 {
		m.HasHAD = true
		m.HAD = sporttery.Odds{H: row.EUHome, D: row.EUDraw, A: row.EUAway}
	}
	return m
}

func parseSFCKick(s string, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	t, err := time.ParseInLocation("01-02 15:04", strings.TrimSpace(s), loc)
	if err != nil {
		return now.Add(24 * time.Hour)
	}
	t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
	if t.Before(now.Add(-36 * time.Hour)) {
		t = t.AddDate(1, 0, 0)
	}
	return t
}

func (s *Scheduler) ingest(kind store.SnapshotKind, onlyMissing bool) error {
	matches, err := s.Client.Fetch(s.Location)
	if err != nil {
		return err
	}
	log.Printf("sporttery fetched %d matches", len(matches))
	for _, m := range matches {
		if err := s.Store.UpsertMatch(m); err != nil {
			log.Printf("upsert %s: %v", m.NumStr, err)
		}
	}
	s.refreshQuotes(matches)
	for _, m := range matches {
		if onlyMissing {
			has, err := s.Store.HasSnapshot(m.ID, kind)
			if err != nil || has {
				s.armClose(m)
				continue
			}
		}
		out, err := s.Engine.Run(m, kind)
		if err != nil {
			log.Printf("analyze %s %s: %v", m.NumStr, kind, err)
		} else if out.Skipped {
			log.Printf("skip %s %s (already in sqlite)", m.NumStr, kind)
		} else {
			log.Printf("saved %s %s ai=%v", m.NumStr, kind, out.UsedAI)
		}
		s.armClose(m)
	}
	_ = s.Store.PruneOlderThan(time.Now().Add(-14 * 24 * time.Hour))
	return nil
}

func (s *Scheduler) PollScores() error {
	if s.Market == nil {
		return nil
	}
	fids, err := s.Store.FidMap()
	if err != nil || len(fids) == 0 {
		return err
	}
	scores, err := s.Market.FetchFTScores()
	if err != nil {
		return err
	}
	n := 0
	now := time.Now()
	if s.Location != nil {
		now = now.In(s.Location)
	}
	for fid, sc := range scores {
		id := fids[fid]
		if id == 0 {
			continue
		}
		m, err := s.Store.GetMatch(id)
		if err != nil || m == nil {
			continue
		}
		if !market.ReadyForFullTime(m.Kickoff, now) {
			continue
		}
		if m.Finished && m.HomeGoals != nil && m.AwayGoals != nil && *m.HomeGoals == sc[0] && *m.AwayGoals == sc[1] {
			continue
		}
		if err := s.Store.SaveFT(id, sc[0], sc[1]); err != nil {
			log.Printf("save ft %d: %v", id, err)
			continue
		}
		n++
		if m.Finished {
			log.Printf("比分更新 %s %d-%d -> %d-%d", m.NumStr, *m.HomeGoals, *m.AwayGoals, sc[0], sc[1])
		} else {
			log.Printf("完场 %s %d-%d", m.NumStr, sc[0], sc[1])
		}
	}
	if n > 0 {
		log.Printf("scores saved %d", n)
	}
	return nil
}

func (s *Scheduler) BackfillExperts() error {
	if s.Engine == nil {
		return nil
	}
	from := time.Now().Add(-4 * time.Hour)
	list, err := s.Store.ListUpcoming(from)
	if err != nil {
		return err
	}
	n := 0
	for _, m := range list {
		sn, err := s.Store.PreferredSnapshot(m.ID)
		if err != nil || sn == nil || len(sn.Takes) > 0 || sn.ExpertDone {
			continue
		}
		if _, err := s.Engine.FillTakes(m.ID, sn.Kind); err != nil {
			log.Printf("expert %s: %v", m.NumStr, err)
			continue
		}
		n++
		log.Printf("expert saved %s", m.NumStr)
	}
	if n > 0 {
		log.Printf("experts backfilled %d", n)
	}
	return nil
}

func (s *Scheduler) BackfillWeekReview() error {
	if s.Engine == nil {
		return nil
	}
	from := store.WeekStart(time.Now().In(s.Location))
	list, err := s.Store.ListFinishedSince(from)
	if err != nil {
		return err
	}
	n := 0
	for _, m := range list {
		sn, err := s.Store.GetSnapshot(m.ID, store.KindOpen)
		if err != nil || sn == nil || len(sn.Takes) > 0 {
			continue
		}
		if _, err := s.Engine.FillMissingTakes(m.ID, store.KindOpen); err != nil {
			log.Printf("week review %s: %v", m.NumStr, err)
			continue
		}
		n++
		log.Printf("week review saved %s", m.NumStr)
	}
	if n > 0 {
		log.Printf("week review backfilled %d", n)
	}
	return nil
}

func (s *Scheduler) Resume() error {
	from := time.Now().Add(-4 * time.Hour)
	list, err := s.Store.ListUpcoming(from)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return s.DailyOpen()
	}
	live, err := s.Client.Fetch(s.Location)
	if err != nil {
		log.Printf("resume fetch: %v", err)
		return nil
	}
	byID := map[int64]sporttery.Match{}
	for _, m := range live {
		_ = s.Store.UpsertMatch(m)
		byID[m.ID] = m
	}
	s.refreshQuotes(live)
	for _, row := range list {
		m, ok := byID[row.ID]
		if !ok {
			continue
		}
		if !row.HasOpen {
			if _, err := s.Engine.Run(m, store.KindOpen); err != nil {
				log.Printf("resume open %s: %v", m.NumStr, err)
			}
		}
		s.armClose(m)
	}
	return nil
}

func (s *Scheduler) armClose(m sporttery.Match) {
	if m.Kickoff.IsZero() {
		return
	}
	has, err := s.Store.HasSnapshot(m.ID, store.KindClose)
	if err != nil || has {
		return
	}
	when := m.Kickoff.Add(-30 * time.Minute)
	s.mu.Lock()
	if _, ok := s.closes[m.ID]; ok {
		s.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	s.closes[m.ID] = stop
	s.mu.Unlock()

	delay := time.Until(when)
	if delay < 0 {
		delay = 0
	}
	go func(m sporttery.Match, delay time.Duration, stop chan struct{}) {
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-stop:
			return
		case <-t.C:
		}
		if time.Now().After(m.Kickoff) {
			return
		}
		live, err := s.Client.Fetch(s.Location)
		if err != nil {
			log.Printf("close fetch %s: %v", m.NumStr, err)
			return
		}
		var latest sporttery.Match
		found := false
		for _, x := range live {
			if x.ID == m.ID {
				latest = x
				found = true
				break
			}
		}
		if !found {
			latest = m
		}
		s.refreshQuotes([]sporttery.Match{latest})
		out, err := s.Engine.Run(latest, store.KindClose)
		if err != nil {
			log.Printf("close analyze %s: %v", m.NumStr, err)
			return
		}
		if out.Skipped {
			log.Printf("close skip %s (sqlite already has it)", m.NumStr)
			return
		}
		log.Printf("close saved %s ai=%v", m.NumStr, out.UsedAI)
	}(m, delay, stop)
}

func (s *Scheduler) refreshQuotes(matches []sporttery.Match) {
	if s.Market == nil || len(matches) == 0 {
		return
	}
	ids, err := s.Market.MapIDs()
	if err != nil {
		log.Printf("500 map: %v", err)
		return
	}
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for _, m := range matches {
		fid := ids[m.ID]
		if fid == 0 {
			continue
		}
		wg.Add(1)
		go func(m sporttery.Match, fid int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			q, err := s.Market.Fetch(m.ID, fid)
			if err != nil {
				log.Printf("quote %s: %v", m.NumStr, err)
			} else if err := s.Store.SaveQuote(q); err != nil {
				log.Printf("save quote %s: %v", m.NumStr, err)
			}
			p, err := s.Market.FetchPreview(m.ID, fid)
			if err != nil {
				log.Printf("preview %s: %v", m.NumStr, err)
				return
			}
			if p.Home.Name == "" {
				p.Home.Name = m.Home
			}
			if p.Away.Name == "" {
				p.Away.Name = m.Away
			}
			if err := s.Store.SavePreview(p); err != nil {
				log.Printf("save preview %s: %v", m.NumStr, err)
			}
		}(m, fid)
	}
	wg.Wait()
	s.refreshTitan(matches)
}

func (s *Scheduler) refreshTitan(matches []sporttery.Match) {
	if s.Titan == nil || len(matches) == 0 {
		return
	}
	now := time.Now()
	if s.Location != nil {
		now = now.In(s.Location)
	}
	rows, err := s.Titan.FetchJingzu(now)
	if err != nil {
		log.Printf("titan schedule: %v", err)
		return
	}
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	n := 0
	for _, m := range matches {
		hit := titan007.FindMatch(rows, m.NumStr, m.Home, m.Away)
		if hit == nil {
			continue
		}
		wg.Add(1)
		n++
		go func(m sporttery.Match, id int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			odds, err := s.Titan.FetchOdds(id)
			if err != nil {
				log.Printf("titan odds %s: %v", m.NumStr, err)
				return
			}
			q, _ := s.Store.GetQuote(m.ID)
			if q == nil {
				q = &market.Quote{MatchID: m.ID, FetchedAt: time.Now(), Company: "Bet365"}
			}
			titan007.Apply(q, odds)
			q.MatchID = m.ID
			q.FetchedAt = time.Now()
			if err := s.Store.SaveQuote(q); err != nil {
				log.Printf("titan save %s: %v", m.NumStr, err)
			}
		}(m, hit.ID)
	}
	wg.Wait()
	if n > 0 {
		log.Printf("titan odds %d/%d", n, len(matches))
	}
	s.refreshMarketTakes(matches)
}

func (s *Scheduler) refreshMarketTakes(matches []sporttery.Match) {
	if s.Engine == nil || len(matches) == 0 {
		return
	}
	now := time.Now()
	n := 0
	for _, m := range matches {
		if n >= 8 {
			break
		}
		if !m.Kickoff.IsZero() && now.After(m.Kickoff.Add(20*time.Minute)) {
			continue
		}
		ok, err := s.Engine.RefreshMarketTake(m.ID)
		if err != nil {
			log.Printf("market take %s: %v", m.NumStr, err)
			continue
		}
		if ok {
			n++
		}
	}
	if n > 0 {
		log.Printf("market takes %d", n)
	}
}
