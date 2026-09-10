// Package scriptset implements section 5 of UTS #39, Unicode Security
// Mechanisms: mixed-script detection and restriction levels.
//
// The unit of the algorithm is the augmented script set of a character: its
// Script_Extensions value, plus the writing systems Hanb, Jpan and Kore that
// UTS #39 adds on top of Han, Hiragana, Katakana, Hangul and Bopomofo. The
// resolved script set of a string is the intersection of those sets over all
// its characters, and a string is single-script when that intersection is
// non-empty.
//
// Two properties are easy to confuse and give different answers. Script
// (which the standard library exposes as unicode.Scripts) says U+30FC
// KATAKANA-HIRAGANA PROLONGED SOUND MARK is Common; Script_Extensions, which
// this package uses and UTS #39 requires, says it is {Hira, Kana}.
//
// This package does not implement confusable detection: skeletons and the
// confusables mapping of UTS #39 section 4 are in
// github.com/mtibben/confusables. This is the other half.
package scriptset

import (
	"math/bits"
	"sort"
	"strings"
	"unicode/utf8"
)

// A Script is an ISO 15924 script code. Beyond the Script property values it
// includes Hanb, Jpan and Kore, the three writing systems of UTS #39 section
// 5.1, which are not Script values in the UCD.
type Script uint16

// String returns the four-letter ISO 15924 code, or "" for an invalid Script.
func (s Script) String() string {
	if int(s) >= numScripts {
		return ""
	}
	return scriptNames[s]
}

// ParseScript returns the Script with the given four-letter code.
func ParseScript(code string) (Script, bool) {
	// scriptNames is sorted except for the three writing systems appended at
	// the end, so a linear scan is the honest thing here; the table has fewer
	// than 200 entries.
	for i, n := range scriptNames {
		if n == code {
			return Script(i), true
		}
	}
	return 0, false
}

// A Set is a set of scripts.
//
// The zero Set is empty. Sets are comparable with ==.
type Set struct {
	w [setWords]uint64
}

// All returns ALL, the set of every script value. UTS #39 section 5.1 gives
// this set to any character whose Script_Extensions contains Common or
// Inherited.
func All() Set { return allSet }

// NewSet returns the set containing exactly the given scripts.
func NewSet(scripts ...Script) Set {
	var s Set
	for _, sc := range scripts {
		if int(sc) < numScripts {
			s.w[sc/64] |= 1 << uint(sc%64)
		}
	}
	return s
}

// Contains reports whether the set contains sc.
func (s Set) Contains(sc Script) bool {
	if int(sc) >= numScripts {
		return false
	}
	return s.w[sc/64]&(1<<uint(sc%64)) != 0
}

// Intersect returns the intersection of s and t.
func (s Set) Intersect(t Set) Set {
	var out Set
	for i := range s.w {
		out.w[i] = s.w[i] & t.w[i]
	}
	return out
}

// Union returns the union of s and t.
func (s Set) Union(t Set) Set {
	var out Set
	for i := range s.w {
		out.w[i] = s.w[i] | t.w[i]
	}
	return out
}

// Empty reports whether the set contains no scripts.
func (s Set) Empty() bool {
	for _, w := range s.w {
		if w != 0 {
			return false
		}
	}
	return true
}

// IsAll reports whether the set is ALL.
func (s Set) IsAll() bool { return s == allSet }

// Len returns the number of scripts in the set.
func (s Set) Len() int {
	n := 0
	for _, w := range s.w {
		n += bits.OnesCount64(w)
	}
	return n
}

// Scripts returns the scripts in the set, in ascending Script order.
func (s Set) Scripts() []Script {
	out := make([]Script, 0, s.Len())
	for i := 0; i < numScripts; i++ {
		if s.Contains(Script(i)) {
			out = append(out, Script(i))
		}
	}
	return out
}

// String returns the set in brace notation, as UTS #39 writes it: "{Hira,
// Kana}". ALL is rendered as "ALL" and the empty set as "{}". Codes are
// sorted alphabetically so the output is stable.
func (s Set) String() string {
	if s.IsAll() {
		return "ALL"
	}
	names := make([]string, 0, s.Len())
	for _, sc := range s.Scripts() {
		names = append(names, sc.String())
	}
	sort.Strings(names)
	return "{" + strings.Join(names, ", ") + "}"
}

// Augmented returns the augmented script set of r, as defined in UTS #39
// section 5.1: the Script_Extensions of r, with Hanb, Jpan and Kore added by
// the five augmentation rules, or ALL if Script_Extensions contains Common or
// Inherited.
//
// Unassigned code points and surrogates have the Script value Unknown, so
// Augmented returns {Zzzz} for them.
func Augmented(r rune) Set {
	if r < 0 || r > 0x10FFFF {
		return NewSet(Zzzz)
	}
	i := sort.Search(len(scriptRanges), func(i int) bool {
		return scriptRanges[i].hi >= r
	})
	if i == len(scriptRanges) || scriptRanges[i].lo > r {
		return NewSet(Zzzz)
	}
	return Set{setPool[scriptRanges[i].set]}
}

// SOSS returns the set of script sets of s: the augmented script set of each
// character, with duplicates removed. UTS #39 section 5.1 introduces it as a
// way to compute the resolved script set and the restriction level without
// revisiting every character.
//
// The result is in first-appearance order. For the empty string it is nil.
func SOSS(s string) []Set {
	var out []Set
	for _, r := range s {
		set := Augmented(r)
		seen := false
		for _, have := range out {
			if have == set {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, set)
		}
	}
	return out
}

// Resolve returns the resolved script set of s: the intersection of the
// augmented script sets of all its characters.
//
// For the empty string it returns ALL, which is the intersection over no
// characters.
func Resolve(s string) Set {
	out := allSet
	for _, r := range s {
		out = out.Intersect(Augmented(r))
		if out.Empty() {
			return out
		}
	}
	return out
}

// IsSingleScript reports whether s is single-script, that is whether its
// resolved script set is non-empty.
//
// The name is the standard's. It does not mean the string uses exactly one
// script: "〆切" is single-script with the resolved set {Hanb, Hani, Jpan,
// Kore}.
func IsSingleScript(s string) bool { return !Resolve(s).Empty() }

// IsMixedScript reports whether s is mixed-script, that is whether its
// resolved script set is empty.
func IsMixedScript(s string) bool { return Resolve(s).Empty() }

// Covers reports whether the script set cover covers s: whether every
// character of s shares at least one script with cover.
func Covers(cover Set, s string) bool {
	for _, r := range s {
		if cover.Intersect(Augmented(r)).Empty() {
			return false
		}
	}
	return true
}

// Allowed reports whether r is in the General Security Profile for
// Identifiers, that is whether its Identifier_Status is Allowed (UTS #39
// section 3.1). Every code point that is not Allowed is Restricted.
func Allowed(r rune) bool {
	i := sort.Search(len(allowedRanges), func(i int) bool {
		return allowedRanges[i].hi >= r
	})
	return i < len(allowedRanges) && allowedRanges[i].lo <= r
}

// AllAllowed reports whether every character of s is Allowed. Invalid UTF-8
// makes it false: an identifier that cannot be decoded is not in any profile.
func AllAllowed(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !Allowed(r) {
			return false
		}
	}
	return true
}

// A Level is a UTS #39 restriction level.
type Level int

// The restriction levels of UTS #39 section 5.2, in increasing order of
// permissiveness. Comparing levels with <= is meaningful: a string at
// HighlyRestrictive also satisfies any policy that asks for
// ModeratelyRestrictive.
const (
	ASCIIOnly Level = iota
	SingleScript
	HighlyRestrictive
	ModeratelyRestrictive
	MinimallyRestrictive
	Unrestricted
)

var levelNames = [...]string{
	"ASCII-Only",
	"Single Script",
	"Highly Restrictive",
	"Moderately Restrictive",
	"Minimally Restrictive",
	"Unrestricted",
}

// String returns the level's name as UTS #39 writes it, for example
// "Highly Restrictive".
func (l Level) String() string {
	if l < 0 || int(l) >= len(levelNames) {
		return "invalid"
	}
	return levelNames[l]
}

// LevelOf returns the restriction level of s, following the eight steps of
// UTS #39 section 5.2.
//
// The identifier profile applied is the General Security Profile of section
// 3.1: a string containing any character outside it is Unrestricted. LevelOf
// does not check identifier well-formedness, which section 5.2 leaves to
// "whatever general syntactic constraints are in force" — in Go that is
// usually a UAX #31 ID_Start/ID_Continue check, and it is the caller's.
//
// The empty string is ASCII-Only.
func LevelOf(s string) Level {
	// 1. If the string contains any characters outside of the Identifier
	// Profile, return Unrestricted.
	if !AllAllowed(s) {
		return Unrestricted
	}
	// 2. If no character in the string is above 0x7F, return ASCII-Only.
	if isASCII(s) {
		return ASCIIOnly
	}
	// 3. Compute the string's SOSS.
	soss := SOSS(s)
	// 4. If the SOSS is empty or the intersection of all entries in the SOSS
	// is nonempty, return Single Script.
	if len(soss) == 0 || !intersectAll(soss).Empty() {
		return SingleScript
	}
	// 5. Remove all the entries from the SOSS that contain Latin.
	rest := soss[:0:0]
	for _, set := range soss {
		if !set.Contains(Latn) {
			rest = append(rest, set)
		}
	}
	// 6. If any of {Kore}, {Hanb}, {Jpan} cover the SOSS, return Highly
	// Restrictive.
	//
	// The spec writes this set as "{Japn}"; Jpan is the ISO 15924 code and
	// the one section 5.1 and Table 1a use.
	for _, ws := range []Script{Kore, Hanb, Jpan} {
		if coversSOSS(NewSet(ws), rest) {
			return HighlyRestrictive
		}
	}
	// 7. If the intersection of all entries in the SOSS contains any single
	// Recommended script except Cyrillic or Greek, return Moderately
	// Restrictive.
	inter := intersectAll(rest).Intersect(recommendedSet)
	inter.w[Cyrl/64] &^= 1 << uint(Cyrl%64)
	inter.w[Grek/64] &^= 1 << uint(Grek%64)
	if !inter.Empty() {
		return ModeratelyRestrictive
	}
	// 8. Otherwise, return Minimally Restrictive.
	return MinimallyRestrictive
}

// coversSOSS reports whether cover shares a script with every entry of soss.
// An empty soss is covered vacuously.
func coversSOSS(cover Set, soss []Set) bool {
	for _, set := range soss {
		if cover.Intersect(set).Empty() {
			return false
		}
	}
	return true
}

// intersectAll returns the intersection of every entry of soss, or ALL when
// soss is empty.
func intersectAll(soss []Set) Set {
	out := allSet
	for _, set := range soss {
		out = out.Intersect(set)
	}
	return out
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7F {
			return false
		}
	}
	return true
}
