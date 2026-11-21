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

// Edit returns the URL of a post's editor. The editor also saves the post to this URL.
func Edit(number int) string {
	return "/" + strconv.Itoa(number) + "/edit"
}

// EditPreview returns the URL that renders the editor's Markdown preview.
func EditPreview(number int) string {
	return Edit(number) + "/preview"
}

// EditUpload returns the URL that receives the editor's image uploads.
func EditUpload(number int) string {
	return Edit(number) + "/upload"
}

// Delete returns the URL that deletes a draft, like /670/delete.
func Delete(number int) string {
	return "/" + strconv.Itoa(number) + "/delete"
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
