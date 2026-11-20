package server

import (
	"fmt"
	"net/url"
	"strconv"
)

// The site's URLs are built only here, so the URL scheme can change in one place.

func postURL(number int, slug string) string {
	if slug == "" {
		return "/" + strconv.Itoa(number)
	}
	return "/" + strconv.Itoa(number) + "/" + url.PathEscape(slug)
}

func editURL(number int) string {
	return "/" + strconv.Itoa(number) + "/edit"
}

func categoryURL(slug string) string {
	return "/" + url.PathEscape(slug)
}

func tagURL(tag string) string {
	return "/archivio/" + url.PathEscape(tag)
}

func monthURL(year, month int) string {
	return fmt.Sprintf("/archivio/%d/%02d", year, month)
}

// pageURL links to a page of a paginated list; the first page has no parameter.
func pageURL(path string, page int) string {
	if page <= 1 {
		return path
	}
	return path + "?page=" + strconv.Itoa(page)
}
