package source

import (
	"context"
	"strings"

	"github.com/SlyMarbo/rss"
	"github.com/samber/lo"

	"github.com/Eneliel/feed_news_tg_bot/internal/model"
)

type RSSSource struct {
	URL        string
	SourceID   int64
	SourceName string
}

func NewRSSSOurceFromModel(m model.Source) RSSSource {
	return RSSSource{
		URL:        m.FeedURL,
		SourceID:   m.ID,
		SourceName: m.Name,
	}
}

func (s RSSSource) Fetch(ctx context.Context) ([]model.Item, error) {
	feed, err := s.loadFeed(ctx, s.URL)
	if err != nil {
		return nil, err
	}

	return lo.Map(feed.Items, func(item *rss.Item, _ int) model.Item {
		return model.Item{
			Title:      item.Title,
			Categories: item.Categories,
			Link:       item.Link,
			Date:       item.Date,
			Summary:    strings.TrimSpace(item.Summary),
			SourceName: s.SourceName,
		}
	}), nil
}

func (s RSSSource) loadFeed(ctx context.Context, url string) (*rss.Feed, error) {
	var (
		feedch = make(chan *rss.Feed)
		errCh  = make(chan error)
	)

	go func() {
		feed, err := rss.Fetch(url)
		if err != nil {
			errCh <- err
			return
		}

		feedch <- feed
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case feed := <-feedch:
		return feed, nil
	}
}

func (s RSSSource) ID() int64 {
	return s.SourceID
}

func (s RSSSource) Name() string {
	return s.SourceName
}
