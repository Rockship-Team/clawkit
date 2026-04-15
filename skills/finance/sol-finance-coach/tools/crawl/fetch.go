package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	rateMu     sync.Mutex
	lastFetch  time.Time
)

const minInterval = 1500 * time.Millisecond // polite rate: ~0.7 req/sec

// fetch retrieves a URL with proper headers and rate limiting.
func fetch(url string) (*http.Response, error) {
	rateMu.Lock()
	elapsed := time.Since(lastFetch)
	if elapsed < minInterval {
		time.Sleep(minInterval - elapsed)
	}
	lastFetch = time.Now()
	rateMu.Unlock()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en;q=0.8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return resp, nil
}

// fetchDoc fetches a URL and parses it into an HTML document.
func fetchDoc(url string) (*html.Node, error) {
	info("fetching " + url)
	resp, err := fetch(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return html.Parse(resp.Body)
}

// fetchBody fetches a URL and returns the body as a string.
func fetchBody(url string) (string, error) {
	info("fetching " + url)
	resp, err := fetch(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// --- HTML traversal helpers ---

// findAll returns all descendant nodes matching a predicate.
func findAll(n *html.Node, match func(*html.Node) bool) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if match(node) {
			results = append(results, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return results
}

// findFirst returns the first descendant matching a predicate, or nil.
func findFirst(n *html.Node, match func(*html.Node) bool) *html.Node {
	var result *html.Node
	var walk func(*html.Node) bool
	walk = func(node *html.Node) bool {
		if match(node) {
			result = node
			return true
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(n)
	return result
}

// hasClass checks if an element has a given CSS class.
func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

// getAttr returns the value of an attribute, or "".
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContent returns the concatenated text of all descendant text nodes.
func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

// isElem checks if node is an element with the given tag.
func isElem(n *html.Node, tag string) bool {
	return n != nil && n.Type == html.ElementNode && n.Data == tag
}

// isElemWithClass checks if node is an element with given tag and class.
func isElemWithClass(n *html.Node, tag, class string) bool {
	return isElem(n, tag) && hasClass(n, class)
}

// findElemsByTag returns all descendant elements with the given tag.
func findElemsByTag(n *html.Node, tag string) []*html.Node {
	return findAll(n, func(node *html.Node) bool {
		return isElem(node, tag)
	})
}

// directChildren returns direct child elements (not text nodes).
func directChildren(n *html.Node) []*html.Node {
	var children []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			children = append(children, c)
		}
	}
	return children
}
