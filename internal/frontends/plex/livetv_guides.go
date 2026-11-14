package plex

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/braedonsaunders/ether/internal/frontend"
	"github.com/braedonsaunders/ether/internal/frontends/plex/model"
)

func normalizeLiveCategory(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "movies", "movie":
		return "movie", true
	case "sports", "sport":
		return "sports", true
	case "news":
		return "news", true
	case "kids-family", "kids":
		return "kids", true
	case "entertainment", "series", "tv-shows", "bingeworthy":
		return "series", true
	default:
		return "", false
	}
}

func (s *Server) livePromotedGuideFilters() []model.GridChannelFilter {
	cats := []struct {
		ident string
		title string
	}{
		{"movies", "Movies"},
		{"sports", "Sports"},
		{"news", "News"},
		{"kids-family", "Kids & Family"},
		{"entertainment", "Entertainment"},
	}
	out := make([]model.GridChannelFilter, 0, len(cats))
	for _, c := range cats {
		out = append(out, model.GridChannelFilter{
			Identifier: c.ident,
			Key:        liveProviderPath + "/lineups/dvr/channels?genre=" + c.ident,
			Title:      c.title,
		})
	}
	return out
}

func (s *Server) liveGuideFilters(ctx context.Context, u *frontend.User) []model.GridChannelFilter {
	out := []model.GridChannelFilter{}
	if g, ok := s.svc().liveTV.(frontend.LiveTVGuides); ok {
		guides, err := g.Guides(ctx, u)
		if err != nil {
			s.log.Warn("live guides unavailable", "error", err)
			return append(out, s.livePromotedGuideFilters()...)
		}
		for _, g := range guides {
			out = append(out, model.GridChannelFilter{Identifier: "guide-" + g.ID, Key: liveProviderPath + "/guides/" + url.PathEscape(g.ID) + "/channels", GenreRatingKey: "ether.guide." + g.ID, Title: g.Name})
		}
	}
	return append(out, s.livePromotedGuideFilters()...)
}
func (s *Server) liveCategoryChannels(r *http.Request, cs []frontend.LiveTVChannel, category string) ([]frontend.LiveTVChannel, error) {
	canonical, ok := normalizeLiveCategory(category)
	if !ok {
		return nil, frontend.ErrNotFound
	}
	ids := make([]string, 0, len(cs))
	for _, c := range cs {
		ids = append(ids, c.ID)
	}
	// A bounded window, no catalogue lookup and no provider/network refresh.
	if len(ids) == 0 {
		return cs, nil
	}
	now := time.Now()
	ps, err := s.svc().liveTV.GuideForChannels(r.Context(), FromContext(r.Context()).User, now, now.Add(2*time.Hour), ids)
	if err != nil {
		return nil, err
	}
	matches := map[string]bool{}
	for _, p := range ps {
		if frontend.LiveCategory(p.Item.Genres, canonical, p.Item.MediaType) {
			matches[p.ChannelID] = true
		}
	}
	out := []frontend.LiveTVChannel{}
	for _, c := range cs {
		if matches[c.ID] {
			out = append(out, c)
		}
	}
	return out, nil
}
