package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/urls"
)

const (
	feedItems         = 50
	feedSummaryLength = 200
)

var xmlText = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
var xmlAttribute = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

// feed serves the RSS feed of the latest posts, with the same elements and
// layout as the SvelteKit app's.
func (s *Server) feed(w http.ResponseWriter, r *http.Request) error {
	posts, err := s.store.Posts(r.Context(), db.PostFilter{}, feedItems, 0)
	if err != nil {
		return err
	}
	items, err := s.postItems(r.Context(), posts)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Note di Matteo</title>
    <link>` + xmlText.Replace(s.siteURL) + `</link>
    <description>Note di Matteo</description>
    <language>it-IT</language>
    <atom:link href="` + xmlAttribute.Replace(s.siteURL+"/feed.xml") + `" rel="self" type="application/rss+xml"></atom:link>
`)

	for _, item := range items {
		permalink := s.siteURL + urls.Post(item.Number, item.Slug)
		title := "#" + strconv.Itoa(item.Number)
		if item.Title != "" {
			title += " / " + item.Title
		}
		description := truncate(item.Body, feedSummaryLength)
		if len([]rune(item.Body)) > feedSummaryLength {
			description += "…"
		}

		b.WriteString("    <item>\n")
		b.WriteString("      <title>" + xmlText.Replace(title) + "</title>\n")
		b.WriteString("      <link>" + xmlText.Replace(permalink) + "</link>\n")
		b.WriteString("      <guid>" + xmlText.Replace(permalink) + "</guid>\n")
		b.WriteString("      <pubDate>" + item.PublishedAt.UTC().Format(http.TimeFormat) + "</pubDate>\n")
		for _, category := range append([]string{item.Category}, item.Tags...) {
			b.WriteString("      <category>" + xmlText.Replace(category) + "</category>\n")
		}
		b.WriteString("      <description>" + xmlText.Replace(description) + "</description>\n")
		b.WriteString("      <content:encoded>\n        " + cdata(item.HTML) + "\n      </content:encoded>\n")
		b.WriteString("    </item>\n")
	}

	b.WriteString("  </channel>\n</rss>\n")

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_, err = w.Write([]byte(b.String()))
	return err
}

// cdata wraps text in a CDATA section, splitting it where the text contains
// the section's end marker.
func cdata(text string) string {
	return "<![CDATA[" + strings.ReplaceAll(text, "]]>", "]]]]><![CDATA[>") + "]]>"
}
