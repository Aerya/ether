package plex

import (
	"context"
	"net/url"
	"strings"

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
	case "crime", "true-crime":
		return "crime", true
	case "reality":
		return "reality", true
	case "comedy":
		return "comedy", true
	case "adrenaline-sci-fi":
		return "adrenaline-sci-fi", true
	case "explore":
		return "explore", true
	case "food-home-culture":
		return "food-home-culture", true
	case "en-espanol":
		return "en-espanol", true
	default:
		return "", false
	}
}

var liveGenreAliases = map[string][]string{
	"crime":             {"crime", "true crime", "crime drama"},
	"reality":           {"reality", "competition reality"},
	"comedy":            {"comedy", "sitcom", "romantic comedy"},
	"adrenaline-sci-fi": {"adrenaline + sci-fi", "adrenaline & sci-fi", "science fiction", "action", "adventure", "thriller", "mystery"},
	"explore":           {"explore", "documentary", "outdoors", "animals", "paranormal"},
	"food-home-culture": {"food + home + culture", "food, home & culture", "cooking", "house/garden", "auction", "collectibles"},
	"en-espanol":        {"en español", "en espanol", "spanish"},
}

func matchesLiveCategory(genres []string, canonical, mediaType string) bool {
	if aliases, ok := liveGenreAliases[canonical]; ok {
		for _, g := range genres {
			gl := strings.ToLower(strings.TrimSpace(g))
			for _, a := range aliases {
				if gl == a {
					return true
				}
			}
		}
		return false
	}
	return frontend.LiveCategory(genres, canonical, mediaType)
}

func liveCategoryGuideFilters() []model.GridChannelFilter {
	cats := []struct {
		ident          string
		genreRatingKey string
		title          string
	}{
		{"bingeworthy", "genre_6a0c2728768cf9b31f6fe02a", "Bingeworthy"},
		{"movies", "genre_620143f98578b9238e1cdb89", "Movies"},
		{"crime", "genre_6006cc1a610ee2002c74f33f", "True Crime"},
		{"news", "genre_6006cc1d610ee2002c74f37a", "News"},
		{"sports", "genre_620143f98578b9238e1cdb8a", "Sports"},
		{"reality", "genre_6006cc1d610ee2002c74f38c", "Reality"},
		{"adrenaline-sci-fi", "genre_6a0c2728768cf9b31f6fe030", "Adrenaline & Sci-Fi"},
		{"comedy", "genre_6006cc18610ee2002c74f2f9", "Comedy"},
		{"explore", "genre_6074b802507c8d42cf7e22eb", "Explore"},
		{"food-home-culture", "genre_6a0c2729768cf9b31f6fe040", "Food, Home & Culture"},
		{"kids-family", "genre_6074b802507c8d42cf7e1678", "Kids & Family"},
		{"en-espanol", "genre_6171e01d4c451f5a44debdf6", "En Español"},
	}
	out := make([]model.GridChannelFilter, 0, len(cats))
	for _, c := range cats {
		out = append(out, model.GridChannelFilter{
			Identifier:     c.ident,
			Key:            liveProviderPath + "/lineups/dvr/channels?genre=" + c.ident,
			GenreRatingKey: c.genreRatingKey,
			Title:          c.title,
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
			return append(out, liveCategoryGuideFilters()...)
		}
		for _, g := range guides {
			out = append(out, model.GridChannelFilter{Identifier: "guide-" + g.ID, Key: liveProviderPath + "/guides/" + url.PathEscape(g.ID) + "/channels", GenreRatingKey: "ether.guide." + g.ID, Title: g.Name})
		}
	}
	return append(out, liveCategoryGuideFilters()...)
}
func liveChannelGenreKeysFromProgrammes(ps []frontend.LiveTVProgramme) map[string]model.StringList {
	filters := liveCategoryGuideFilters()
	byChannel := make(map[string]map[string]bool)
	for _, p := range ps {
		for _, f := range filters {
			canonical, ok := normalizeLiveCategory(f.Identifier)
			if !ok {
				continue
			}
			if matchesLiveCategory(p.Item.Genres, canonical, p.Item.MediaType) {
				m := byChannel[p.ChannelID]
				if m == nil {
					m = make(map[string]bool)
					byChannel[p.ChannelID] = m
				}
				m[f.GenreRatingKey] = true
			}
		}
	}
	out := make(map[string]model.StringList, len(byChannel))
	for cid, m := range byChannel {
		keys := make(model.StringList, 0, len(m))
		for _, f := range filters {
			if m[f.GenreRatingKey] {
				keys = append(keys, f.GenreRatingKey)
			}
		}
		if len(keys) > 0 {
			out[cid] = keys
		}
	}
	return out
}

func (s *Server) liveCategoryChannels(cs []frontend.LiveTVChannel, category string, ps []frontend.LiveTVProgramme) ([]frontend.LiveTVChannel, error) {
	canonical, ok := normalizeLiveCategory(category)
	if !ok {
		return nil, frontend.ErrNotFound
	}
	if len(cs) == 0 {
		return cs, nil
	}
	matches := map[string]bool{}
	for _, p := range ps {
		if matchesLiveCategory(p.Item.Genres, canonical, p.Item.MediaType) {
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
