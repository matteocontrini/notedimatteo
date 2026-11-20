// Package urls builds the site's URLs, so the URL scheme can change in one place.
package urls

import (
	"fmt"
	"net/url"
	"strconv"
)

func Post(number int, slug string) string {
	if slug == "" {
		return "/" + strconv.Itoa(number)
	}
	return "/" + strconv.Itoa(number) + "/" + url.PathEscape(slug)
}

func Edit(number int) string {
	return "/" + strconv.Itoa(number) + "/edit"
}

func Category(slug string) string {
	return "/" + url.PathEscape(slug)
}

func Tag(tag string) string {
	return "/archivio/" + url.PathEscape(tag)
}

func Month(year, month int) string {
	return fmt.Sprintf("/archivio/%d/%02d", year, month)
}

// Page links to a page of a paginated list; the first page has no parameter.
func Page(path string, page int) string {
	if page <= 1 {
		return path
	}
	return path + "?page=" + strconv.Itoa(page)
}
