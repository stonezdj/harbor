package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/goharbor/harbor/src/lib/errors"
	libhttp "github.com/goharbor/harbor/src/lib/http"
	"github.com/goharbor/harbor/src/server/router"
)

// SetLinkHeader ...
func SetLinkHeader(origURL string, n int, last string) (string, error) {
	passedURL, err := url.Parse(origURL)
	if err != nil {
		return "", err
	}
	passedURL.Fragment = ""

	v := url.Values{}
	v.Add("n", strconv.Itoa(n))
	v.Add("last", last)
	passedURL.RawQuery = v.Encode()
	urlStr := fmt.Sprintf("<%s>; rel=\"next\"", passedURL.String())

	return urlStr, nil
}

// IndexString returns the index of X in a sorts string array
// If the array is not sorted, sort it.
func IndexString(strs []string, x string) int {
	if !sort.StringsAreSorted(strs) {
		sort.Strings(strs)
	}
	i := sort.Search(len(strs), func(i int) bool { return x <= strs[i] })
	if i < len(strs) && strs[i] == x {
		return i
	}
	return -1
}

const emptyN = -1

// ParseNAndLastParameters parse the n and last parameters from the query of the http request
func ParseNAndLastParameters(r *http.Request) (int, string, error) {
	q := r.URL.Query()

	n := emptyN

	if q.Get("n") != "" {
		value, err := strconv.Atoi(q.Get("n"))
		if err != nil || value < 0 {
			return 0, "", errors.New(err).WithCode(errors.BadRequestCode).WithMessage("the N must be a positive int type")
		}

		n = value
	}

	return n, q.Get("last"), nil
}

// SendListTagsResponse sends the response for list tags API
func SendListTagsResponse(w http.ResponseWriter, r *http.Request, tags []string) {
	n, last, err := ParseNAndLastParameters(r)
	if err != nil {
		libhttp.SendError(w, err)
		return
	}

	items, nextLast := pickItems(sortedAndUniqueItems(tags), n, last)

	if nextLast != "" {
		link, err := SetLinkHeader(r.URL.String(), n, nextLast)
		if err != nil {
			libhttp.SendError(w, err)
			return
		}

		w.Header().Set("Link", link)
	}

	body := struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{
		Name: router.Param(r.Context(), ":splat"),
		Tags: items,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		libhttp.SendError(w, err)
	}
}

func sortedAndUniqueItems(items []string) []string {
	n := len(items)
	if n <= 1 {
		return items[0:n]
	}

	sort.Strings(items)

	j := 1
	for i := 1; i < n; i++ {
		if items[i] != items[i-1] {
			items[j] = items[i]
			j++
		}
	}

	return items[0:j]
}

func pickItems(tags []string, n int, last string) ([]string, string) {
	if len(tags) == 0 || n == 0 {
		return []string{}, ""
	}

	if n == emptyN {
		n = len(tags)
	}

	i := 0
	if last != "" {
		lastIndex := sort.Search(len(tags), func(i int) bool { return strings.Compare(tags[i], last) >= 0 })
		if tags[lastIndex] == last {
			i = lastIndex + 1
		} else {
			i = lastIndex
		}
	}

	j := i + n

	if j >= len(tags) {
		j = len(tags)
	}

	result := tags[i:j]

	nextLast := ""
	if len(result) > 0 && tags[len(tags)-1] != result[len(result)-1] {
		nextLast = result[len(result)-1]
	}

	return result, nextLast
}
