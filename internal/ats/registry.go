package ats

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type registryEntry struct {
	adapter Adapter
	slug    string
	name    string
}

type slugEntry struct {
	slug string
	norm string
}

// Registry is the read-only union of all adapter rosters and owns name resolution.
type Registry struct {
	bySlug map[string]registryEntry // key: normalize(slug)
	byName map[string]registryEntry // key: normalize(display name)
	slugs  []slugEntry              // sorted by slug, for suggestions
}

// NewRegistry builds a registry and rejects duplicate slugs or names.
func NewRegistry(adapters ...Adapter) (*Registry, error) {
	r := &Registry{
		bySlug: make(map[string]registryEntry),
		byName: make(map[string]registryEntry),
	}
	for _, a := range adapters {
		for _, c := range a.Roster() {
			e := registryEntry{adapter: a, slug: c.Slug, name: c.Name}
			slugKey := normalize(c.Slug)
			if prev, ok := r.bySlug[slugKey]; ok {
				return nil, fmt.Errorf("ats: company slug %q from %s collides with %q from %s",
					c.Slug, a.Name(), prev.slug, prev.adapter.Name())
			}
			r.bySlug[slugKey] = e
			nameKey := normalize(c.Name)
			if prev, ok := r.byName[nameKey]; ok {
				return nil, fmt.Errorf("ats: company name %q from %s collides with %q from %s",
					c.Name, a.Name(), prev.name, prev.adapter.Name())
			}
			r.byName[nameKey] = e
			r.slugs = append(r.slugs, slugEntry{slug: c.Slug, norm: slugKey})
		}
	}
	sort.Slice(r.slugs, func(i, j int) bool { return r.slugs[i].slug < r.slugs[j].slug })
	return r, nil
}

// Resolve maps a company string to an adapter and slug, or returns suggestions.
func (r *Registry) Resolve(company string) (Adapter, string, error) {
	key := normalize(company)
	if key == "" {
		return nil, "", fmt.Errorf("company is required")
	}
	if e, ok := r.bySlug[key]; ok {
		return e.adapter, e.slug, nil
	}
	if e, ok := r.byName[key]; ok {
		return e.adapter, e.slug, nil
	}
	return nil, "", fmt.Errorf("unknown company %q; closest matches: %s. %d companies are supported — pass one of the suggested slugs",
		company, strings.Join(r.suggest(key, 3), ", "), len(r.bySlug))
}

// normalize folds case and removes non-alphanumeric characters.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// suggest ranks substring matches first, then uses edit distance.
func (r *Registry) suggest(key string, n int) []string {
	type scored struct {
		slug string
		dist int
	}
	ranked := make([]scored, 0, len(r.slugs))
	for _, s := range r.slugs {
		// Skip edit distance for substring matches.
		dist := 0
		if !strings.Contains(s.norm, key) && !strings.Contains(key, s.norm) {
			dist = levenshtein(key, s.norm)
		}
		ranked = append(ranked, scored{slug: s.slug, dist: dist})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].dist != ranked[j].dist {
			return ranked[i].dist < ranked[j].dist
		}
		return ranked[i].slug < ranked[j].slug
	})
	if len(ranked) > n {
		ranked = ranked[:n]
	}
	out := make([]string, 0, len(ranked))
	for _, s := range ranked {
		out = append(out, s.slug)
	}
	return out
}

// levenshtein computes edit distance with two rows of storage.
func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}
