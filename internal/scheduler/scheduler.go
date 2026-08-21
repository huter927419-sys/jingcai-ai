package scheduler

import (
	"log"
	"sync"
	"time"

	"jingcai-ai/internal/analyze"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	Store    *store.Store
	Client   *sporttery.Client
	Market   *market.Client
	Engine   *analyze.Engine
	Location *time.Location

	mu     sync.Mutex
	closes map[int64]chan struct{}
	cron   *cron.Cron
}

func New(st *store.Store, cl *sporttery.Client, mk *market.Client, eng *analyze.Engine, loc *time.Location) *Scheduler {
	c := cron.New(cron.WithLocation(loc), cron.WithSeconds())
	return &Scheduler{
		Store:    st,
		Client:   cl,
		Market:   mk,
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
		if err := s.RefreshSFC(); err != nil {
			log.Printf("sfc: %v", err)
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
		if err := s.RefreshSFC(); err != nil {
			log.Printf("sfc: %v", err)
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
	if s.Market == nil {
		return nil
	}
	board, err := s.Market.FetchSFC()
	if err != nil {
		return err
	}
	s.fillSFCAnalysis(board)
	if err := s.Store.SaveSFC(board); err != nil {
		return err
	}
	n := 0
	for _, m := range board.Matches {
		if m.AnalyzedHome+m.AnalyzedDraw+m.AnalyzedAway > 1 {
			n++
		}
	}
	log.Printf("sfc issue %s matches %d analyzed %d", board.Issue, len(board.Matches), n)
	return nil
}

func (s *Scheduler) fillSFCAnalysis(board *market.SFCBoard) {
	if board == nil {
		return
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
		if m := s.Store.MatchSFC(*row, now); m != nil {
			if sn, err := s.Store.PreferredSnapshot(m.ID); err == nil && sn != nil && sn.Result.HomeWin+sn.Result.Draw+sn.Result.AwayWin > 1 {
				continue
			}
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
	}
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
}
