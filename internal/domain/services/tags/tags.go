// Package tags evaluates the Python CLI's boolean and hierarchical tag language.
package tags

import (
	"errors"
	"strings"
	"unicode"
)

var ErrExpression = errors.New("invalid tag expression")

type predicate func(map[string]bool) bool

// Match requires every expression to match. An empty list selects every skill.
func Match(values, expressions []string) (bool, error) {
	set := map[string]bool{}
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			set[v] = true
		}
	}
	result := true
	for _, expr := range expressions {
		p := parser{tokens: tokenize(expr)}
		fn, err := p.or()
		if err != nil || p.pos != len(p.tokens) {
			return false, ErrExpression
		}
		result = fn(set) && result
	}
	return result, nil
}
func tokenize(s string) []string {
	var out []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			out = append(out, word.String())
			word.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsSpace(r) {
			flush()
		} else if strings.ContainsRune("()!&|", r) {
			flush()
			out = append(out, string(r))
		} else {
			word.WriteRune(r)
		}
	}
	flush()
	return out
}

type parser struct {
	tokens []string
	pos    int
}

func (p *parser) take(s string) bool {
	if p.pos < len(p.tokens) && p.tokens[p.pos] == s {
		p.pos++
		return true
	}
	return false
}
func (p *parser) or() (predicate, error) {
	left, err := p.and()
	if err != nil {
		return nil, err
	}
	for p.take("|") {
		right, e := p.and()
		if e != nil {
			return nil, e
		}
		l, r := left, right
		left = func(s map[string]bool) bool { return l(s) || r(s) }
	}
	return left, nil
}
func (p *parser) and() (predicate, error) {
	left, err := p.atom()
	if err != nil {
		return nil, err
	}
	for p.take("&") {
		right, e := p.atom()
		if e != nil {
			return nil, e
		}
		l, r := left, right
		left = func(s map[string]bool) bool { return l(s) && r(s) }
	}
	return left, nil
}
func (p *parser) atom() (predicate, error) {
	if p.take("!") {
		next, err := p.atom()
		if err != nil {
			return nil, err
		}
		return func(s map[string]bool) bool { return !next(s) }, nil
	}
	if p.take("(") {
		next, err := p.or()
		if err != nil || !p.take(")") {
			return nil, ErrExpression
		}
		return next, nil
	}
	if p.pos == len(p.tokens) || strings.ContainsAny(p.tokens[p.pos], "()&|") {
		return nil, ErrExpression
	}
	term := p.tokens[p.pos]
	p.pos++
	return func(s map[string]bool) bool { return matchTerm(s, term) }, nil
}
func matchTerm(set map[string]bool, term string) bool {
	if term == "*" || term == "**" || term == "/**" {
		return len(set) > 0
	}
	if term == "/*" {
		for t := range set {
			if strings.Contains(t, "/") {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(term, "/**") {
		prefix := strings.TrimSuffix(term, "/**")
		for t := range set {
			if t == prefix || strings.HasPrefix(t, prefix+"/") {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(term, "/*") {
		prefix := strings.TrimSuffix(term, "/*")
		for t := range set {
			if t == prefix || strings.HasPrefix(t, prefix+"/") && !strings.Contains(strings.TrimPrefix(t, prefix+"/"), "/") {
				return true
			}
		}
		return false
	}
	if strings.Contains(term, "*") {
		for t := range set {
			if pattern(strings.Split(term, "/"), strings.Split(t, "/")) {
				return true
			}
		}
		return false
	}
	// Compatibility: the query a/b/c accepts a skill tagged b, b/c, or a/b/c.
	parts := strings.FieldsFunc(term, func(r rune) bool { return r == '/' })
	for i := range parts {
		for j := i + 1; j <= len(parts); j++ {
			if set[strings.Join(parts[i:j], "/")] {
				return true
			}
		}
	}
	return false
}
func pattern(p, t []string) bool {
	// Dynamic programming keeps multiple ** terms from causing exponential work.
	dp := make([]bool, len(t)+1)
	dp[0] = true
	for _, part := range p {
		next := make([]bool, len(t)+1)
		for j := 0; j <= len(t); j++ {
			if part == "**" {
				next[j] = dp[j] || (j > 0 && next[j-1])
			} else if j > 0 {
				next[j] = dp[j-1] && (part == "*" || part == t[j-1])
			}
		}
		dp = next
	}
	return dp[len(t)]
}
