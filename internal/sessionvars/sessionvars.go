// Package sessionvars expands the small set of substitution variables the
// New Session form offers in a working directory and a launch command:
//
//	~/sessions/$name    -> ~/sessions/api-refactor
//	claude --name $name -> claude --name api-refactor
//	~/logs/$date        -> ~/logs/2026-07-31
//
// The whole design hangs on one constraint: `$` already means something on
// the other end. The launch command is delivered with send-keys, i.e. typed
// into an interactive shell, so `cd $HOME && claude` genuinely depends on the
// shell expanding `$HOME` itself. So the rule is: substitute ONLY the names
// listed here, and pass every other $-form through byte-for-byte. Anything
// looser breaks commands people are already running.
//
// The package is pure — no I/O, no globals — so both the HTTP handlers and
// the MCP server can call it without an import cycle, and there is exactly
// one implementation of the rules.
package sessionvars

import (
	"strings"
	"time"
)

// Vars are the values the known variables expand to.
type Vars struct {
	// Name is the SANITIZED session name — the real tmux name and the
	// registry key. Expanding to the name as typed would put a space in the
	// path, which is precisely what sanitizing exists to prevent.
	Name string
	// Now dates $date. Server-local rather than UTC: someone writing
	// ~/logs/$date means their own calendar day, not the registry's. A zero
	// value falls back to the current time, so a caller that forgets to set
	// it gets today rather than the year 1.
	Now time.Time
}

// dateLayout is the one format $date produces. Sortable, unambiguous, and
// safe in a path on every filesystem — which rules out anything with a slash
// or a colon in it.
const dateLayout = "2006-01-02"

// names are the variables we own. Keep this list short: every name added
// here is a name the remote shell can no longer have.
var names = []string{"name", "date"}

// Expand substitutes the known variables in s and leaves every other $-form
// exactly as typed, for the remote shell to deal with.
func Expand(s string, v Vars) string {
	out, _ := expand(s, v)
	return out
}

// HasVars reports whether s actually uses one of the known variables — as
// opposed to merely containing a `$`. Callers use it to decide whether a
// template is worth persisting next to its expansion; an escaped `\$name`
// is not a use, because the point of escaping it is that it stays literal.
func HasVars(s string) bool {
	_, used := expand(s, Vars{})
	return used
}

// expand returns the substituted string and whether any known variable was
// actually substituted.
func expand(s string, v Vars) (string, bool) {
	// The overwhelmingly common case: no `$` at all, so nothing can change
	// and the caller gets its own string back untouched.
	if !strings.ContainsRune(s, '$') {
		return s, false
	}
	if v.Now.IsZero() {
		v.Now = time.Now()
	}

	var b strings.Builder
	b.Grow(len(s))
	used := false
	for i := 0; i < len(s); {
		c := s[i]

		// `\$name` is the escape hatch: emit `$name` and drop the backslash,
		// so a literal `$name` reaches the shell. Only a name we would
		// otherwise have expanded is unescaped this way — `\$HOME` is the
		// shell's own escape and has to survive byte-for-byte.
		if c == '\\' && i+1 < len(s) && s[i+1] == '$' {
			if _, n := match(s[i+1:]); n > 0 {
				b.WriteString(s[i+1 : i+1+n])
				i += 1 + n
				continue
			}
		}

		if c == '$' {
			if name, n := match(s[i:]); n > 0 {
				b.WriteString(v.value(name))
				i += n
				used = true
				continue
			}
		}

		b.WriteByte(c)
		i++
	}
	return b.String(), used
}

// match identifies a known variable at the start of s (which must begin with
// '$'), returning its name and how many bytes the reference occupies —
// braces included. A zero length means "not one of ours"; the caller then
// copies the bytes through untouched.
func match(s string) (string, int) {
	if len(s) < 2 || s[0] != '$' {
		return "", 0
	}

	// ${name} — the spelling that makes ~/sessions/${name}-work possible,
	// where the bare form would run into the suffix.
	if s[1] == '{' {
		end := strings.IndexByte(s, '}')
		if end < 0 {
			return "", 0
		}
		name := s[2:end]
		if !known(name) {
			return "", 0
		}
		return name, end + 1
	}

	for _, name := range names {
		rest, ok := strings.CutPrefix(s[1:], name)
		if !ok {
			continue
		}
		// A word boundary is required: `$nameless` is one unknown word, not
		// `$name` followed by "less". Without this we would silently rewrite
		// the middle of somebody's shell variable.
		if rest != "" && isIdent(rest[0]) {
			return "", 0
		}
		return name, 1 + len(name)
	}
	return "", 0
}

func known(name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// isIdent reports whether c can continue a shell variable name.
func isIdent(c byte) bool {
	return c == '_' ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}

func (v Vars) value(name string) string {
	switch name {
	case "name":
		return v.Name
	case "date":
		return v.Now.Format(dateLayout)
	}
	return ""
}
