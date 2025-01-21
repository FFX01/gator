package rss

import (
	"context"
	"encoding/xml"
	"html"
	"io"
	"net/http"
	"time"
)

type Feed struct {
	Channel struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		Items       []Item `xml:"item"`
	} `xml:"channel"`
}

type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Pubdate     time.Time `xml:"pubdate"`
}

func unescapeText(feed *Feed) *Feed {
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for _, i := range feed.Channel.Items {
		i.Title = html.UnescapeString(i.Title)
		i.Description = html.UnescapeString(i.Description)
	}

	return feed
}

func FetchFeed(ctx context.Context, feedURL string) (*Feed, error) {
	client := &http.Client{}
	request, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &Feed{}, err
	}
	request.Header.Add("Content-Type", "application/xml")
	request.Header.Add("User-Agent", "gator")

	resp, err := client.Do(request)
	if err != nil {
		return &Feed{}, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Feed{}, err
	}

	parsedData := Feed{}
	err = xml.Unmarshal(data, &parsedData)
	if err != nil {
		return &parsedData, err
	}

	unescapedData := unescapeText(&parsedData)

	return unescapedData, nil
}
