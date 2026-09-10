package scriptset_test

import (
	"testing"
	"unicode"

	"github.com/doxuta/scriptset"
)

// TestTable1a pins every row of Table 1a, Mixed Script Examples, of UTS #39
// section 5.1. The code points are written out because the strings are the
// point: rows 1 to 3 differ only in which characters are Cyrillic.
func TestTable1a(t *testing.T) {
	tests := []struct {
		name     string
		runes    []rune
		resolved string
		single   bool
	}{
		{
			name:     "Circle",
			runes:    []rune{0x0043, 0x0069, 0x0072, 0x0063, 0x006C, 0x0065},
			resolved: "{Latn}",
			single:   true,
		},
		{
			name:     "Circle all Cyrillic",
			runes:    []rune{0x0421, 0x0456, 0x0433, 0x0441, 0x04C0, 0x0435},
			resolved: "{Cyrl}",
			single:   true,
		},
		{
			name:     "Circle mixed Cyrillic and Latin",
			runes:    []rune{0x0421, 0x0069, 0x0072, 0x0441, 0x006C, 0x0435},
			resolved: "{}",
			single:   false,
		},
		{
			name:     "Circ1e with digit one",
			runes:    []rune{0x0043, 0x0069, 0x0072, 0x0063, 0x0031, 0x0065},
			resolved: "{Latn}",
			single:   true,
		},
		{
			name:     "C plus math sans-serif",
			runes:    []rune{0x0043, 0x1D5C2, 0x1D5CB, 0x1D5BC, 0x1D5C5, 0x1D5BE},
			resolved: "{Latn}",
			single:   true,
		},
		{
			name:     "all math sans-serif",
			runes:    []rune{0x1D5A2, 0x1D5C2, 0x1D5CB, 0x1D5BC, 0x1D5C5, 0x1D5BE},
			resolved: "ALL",
			single:   true,
		},
		{
			name:     "shimekiri",
			runes:    []rune{0x3006, 0x5207},
			resolved: "{Hanb, Hani, Jpan, Kore}",
			single:   true,
		},
		{
			name:     "ne ga",
			runes:    []rune{0x306D, 0x30AC},
			resolved: "{Jpan}",
			single:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := string(tt.runes)
			if got := scriptset.Resolve(s).String(); got != tt.resolved {
				t.Errorf("Resolve(%q) = %s, want %s", s, got, tt.resolved)
			}
			if got := scriptset.IsSingleScript(s); got != tt.single {
				t.Errorf("IsSingleScript(%q) = %v, want %v", s, got, tt.single)
			}
			if got := scriptset.IsMixedScript(s); got == tt.single {
				t.Errorf("IsMixedScript(%q) = %v, want %v", s, got, !tt.single)
			}
		})
	}
}

// TestTable1aScriptExtensionsDrift records a divergence between the spec text
// and the data it points at. Table 1a lists U+3006 with Script_Extensions
// {Hani, Hira, Kana}; ScriptExtensions.txt in UCD 17.0.0 gives it {Hani}.
//
// The resolved set of "〆切" is unaffected, because the Hani augmentation rule
// contributes Hanb, Jpan and Kore either way and U+5207 intersects Hira and
// Kana away. This test exists so that a future UCD update which restores the
// wider value is noticed rather than silently absorbed.
func TestTable1aScriptExtensionsDrift(t *testing.T) {
	got := scriptset.Augmented(0x3006)
	if want := "{Hanb, Hani, Jpan, Kore}"; got.String() != want {
		t.Errorf("Augmented(U+3006) = %s, want %s (UCD 17.0.0 Script_Extensions is {Hani})", got, want)
	}
	for _, sc := range []scriptset.Script{scriptset.Hira, scriptset.Kana} {
		if got.Contains(sc) {
			t.Errorf("Augmented(U+3006) contains %s: UCD 17.0.0 dropped it, but Table 1a still shows it", sc)
		}
	}
}

// TestAugmentationRules exercises the five rules of UTS #39 section 5.1 one
// at a time, and the Common/Inherited rule that overrides them.
func TestAugmentationRules(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want string
	}{
		{"Han adds Hanb, Jpan and Kore", 0x4E00, "{Hanb, Hani, Jpan, Kore}"},
		{"Hiragana adds Jpan", 0x3042, "{Hira, Jpan}"},
		{"Katakana adds Jpan", 0x30A2, "{Jpan, Kana}"},
		{"Hangul adds Kore", 0xAC00, "{Hang, Kore}"},
		{"Bopomofo adds Hanb", 0x3105, "{Bopo, Hanb}"},
		{"Latin is untouched", 0x0061, "{Latn}"},
		{"Common becomes ALL", 0x0031, "ALL"},
		{"Inherited becomes ALL", 0x030F, "ALL"},
		// U+0300 is Script=Inherited too, but ScriptExtensions.txt gives it an
		// explicit eight-script set that does not include Zinh, so the ALL
		// rule does not fire. Script and Script_Extensions disagree here.
		{"Inherited with explicit extensions stays narrow", 0x0300,
			"{Cher, Copt, Cyrl, Grek, Latn, Perm, Sunu, Tale}"},
		{"unassigned is Unknown", 0x0378, "{Zzzz}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scriptset.Augmented(tt.r).String(); got != tt.want {
				t.Errorf("Augmented(U+%04X) = %s, want %s", tt.r, got, tt.want)
			}
		})
	}
}

// TestScriptExtensionsNotScript is the reason this package cannot be built on
// unicode.Scripts. U+30FC and U+00B7 both have Script=Common, which a
// Script-property implementation treats as a wildcard, but their
// Script_Extensions are narrow.
func TestScriptExtensionsNotScript(t *testing.T) {
	for _, r := range []rune{0x30FC, 0x00B7} {
		if !unicode.Is(unicode.Common, r) {
			t.Fatalf("U+%04X: precondition failed, expected Script=Common", r)
		}
		if scriptset.Augmented(r).IsAll() {
			t.Errorf("Augmented(U+%04X) = ALL; Script_Extensions is narrower than Script here", r)
		}
	}

	// U+00B7 MIDDLE DOT has sixteen scripts in Script_Extensions and Cyrillic
	// is not among them, so Cyrillic text around a middle dot is mixed-script.
	// An implementation that reads Common as a wildcard calls it single-script.
	const cyrillicWithMiddleDot = "а·б"
	if scriptset.IsSingleScript(cyrillicWithMiddleDot) {
		t.Errorf("IsSingleScript(%q) = true, want false: U+00B7 does not carry Cyrl",
			cyrillicWithMiddleDot)
	}
	// The same shape in Latin is single-script, because Latn is among them.
	if !scriptset.IsSingleScript("a·b") {
		t.Error(`IsSingleScript("a·b") = false, want true: U+00B7 carries Latn`)
	}
}

func TestLevelOf(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want scriptset.Level
	}{
		{"empty", "", scriptset.ASCIIOnly},
		{"ascii", "abc", scriptset.ASCIIOnly},
		{"ascii with digit", "Circ1e", scriptset.ASCIIOnly},
		{"cyrillic alone", "СігсӀе", scriptset.SingleScript},
		{"japanese", "日本語", scriptset.SingleScript},
		{"korean", "한국어", scriptset.SingleScript},
		{"hangul and han", "한글漢字", scriptset.SingleScript},
		{"latin and japanese", "ラーメンramen", scriptset.HighlyRestrictive},
		{"latin and korean", "김치kimchi", scriptset.HighlyRestrictive},
		{"latin and han", "Tokyo日本", scriptset.HighlyRestrictive},
		{"latin and thai", "abcไทย", scriptset.ModeratelyRestrictive},
		{"latin and arabic", "abcعربي", scriptset.ModeratelyRestrictive},
		{"latin and greek is excluded", "abcΑΒΓ", scriptset.MinimallyRestrictive},
		{"latin and cyrillic is excluded", "abcабв", scriptset.MinimallyRestrictive},
		{"cyrillic spoof of Circle", "Сircбe", scriptset.MinimallyRestrictive},
		{"space is not in the profile", "hello world", scriptset.Unrestricted},
		{"at sign is not in the profile", "user@host", scriptset.Unrestricted},
		{"math sans-serif is not in the profile", "\U0001D5A2", scriptset.Unrestricted},
		{"invalid utf-8", "\xff\xfe", scriptset.Unrestricted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scriptset.LevelOf(tt.s); got != tt.want {
				t.Errorf("LevelOf(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// TestSpecMinimallyRestrictiveExamples pins the four strings UTS #39 section
// 5.2 names as examples of "arbitrary mixtures of scripts" at the Minimally
// Restrictive level. They are worth pinning because all four also have to
// pass step 1: if any of their characters were outside the identifier
// profile, the spec's own illustration would come back Unrestricted.
func TestSpecMinimallyRestrictiveExamples(t *testing.T) {
	for _, s := range []string{"Ωmega", "Teχ", "HλLF-LIFE", "Toys-Я-Us"} {
		if !scriptset.AllAllowed(s) {
			t.Errorf("AllAllowed(%q) = false; every character should be in the identifier profile", s)
		}
		if got := scriptset.LevelOf(s); got != scriptset.MinimallyRestrictive {
			t.Errorf("LevelOf(%q) = %v, want %v", s, got, scriptset.MinimallyRestrictive)
		}
	}
}

// TestProfileIsCheckedFirst covers step 1 of section 5.2: a restricted
// character forces Unrestricted even when the string is single-script and
// would otherwise be ASCII-Only or Highly Restrictive.
func TestProfileIsCheckedFirst(t *testing.T) {
	tests := []struct {
		name string
		s    string
	}{
		{"single-script but restricted", "\U0001D5A2\U0001D5C2\U0001D5CB"},
		{"ascii but restricted", "a b"},
		{"latin and japanese but restricted", "ramenラーメン!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if scriptset.AllAllowed(tt.s) {
				t.Fatalf("precondition: %q should contain a restricted character", tt.s)
			}
			if got := scriptset.LevelOf(tt.s); got != scriptset.Unrestricted {
				t.Errorf("LevelOf(%q) = %v, want %v", tt.s, got, scriptset.Unrestricted)
			}
		})
	}
	// ...and the same strings without the restricted character are not
	// Unrestricted, so the test above is not passing for some other reason.
	if got := scriptset.LevelOf("ramenラーメン"); got != scriptset.HighlyRestrictive {
		t.Errorf("control: LevelOf without the restricted character = %v, want %v",
			got, scriptset.HighlyRestrictive)
	}
}

// TestBopomofoIsUnreachableInUnicode17 records a consequence of Unicode 17
// that the text of UTS #39 does not mention. Section 5.2 lists "Latin + Han +
// Bopomofo; or equivalently: Latn + Hanb" as one of the three combinations
// that reach Highly Restrictive. But UAX #31 Table 5 moved Bopomofo to
// Limited Use in Unicode 17, so Bopomofo letters have Identifier_Type
// Limited_Use and Identifier_Status Restricted, and any string containing one
// fails step 1.
//
// The scan below asserts the general form: no code point that carries Hanb
// and is not Han survives the identifier profile, so the {Hanb} branch can
// only ever be satisfied by characters the {Jpan} and {Kore} branches also
// satisfy.
func TestBopomofoIsUnreachableInUnicode17(t *testing.T) {
	if scriptset.Allowed(0x3105) {
		t.Error("Allowed(U+3105 BOPOMOFO LETTER B) = true; Unicode 17 makes it Restricted")
	}
	if got := scriptset.LevelOf("abcㄅㄆ"); got != scriptset.Unrestricted {
		t.Errorf("LevelOf(Latin + Bopomofo) = %v, want %v", got, scriptset.Unrestricted)
	}

	var reachable []rune
	for r := rune(0); r <= 0x10FFFF; r++ {
		set := scriptset.Augmented(r)
		if set.IsAll() || !set.Contains(scriptset.Hanb) || set.Contains(scriptset.Hani) {
			continue
		}
		if scriptset.Allowed(r) {
			reachable = append(reachable, r)
		}
	}
	if len(reachable) != 0 {
		t.Errorf("found %d allowed non-Han code points carrying Hanb, e.g. U+%04X; "+
			"the Latn+Hanb branch is reachable after all", len(reachable), reachable[0])
	}
}

func TestCovers(t *testing.T) {
	jpan := scriptset.NewSet(scriptset.Jpan)
	kore := scriptset.NewSet(scriptset.Kore)
	tests := []struct {
		s          string
		jpan, kore bool
	}{
		{"ラーメン", true, false}, // ramen in katakana
		{"한글漢字", false, true}, // hangul and han
		{"日本語", true, true},   // han only: both writing systems
		{"abc", false, false}, // latin
	}
	for _, tt := range tests {
		if got := scriptset.Covers(jpan, tt.s); got != tt.jpan {
			t.Errorf("Covers({Jpan}, %q) = %v, want %v", tt.s, got, tt.jpan)
		}
		if got := scriptset.Covers(kore, tt.s); got != tt.kore {
			t.Errorf("Covers({Kore}, %q) = %v, want %v", tt.s, got, tt.kore)
		}
	}
}

func TestSOSS(t *testing.T) {
	// "Сirсlе" has two distinct augmented sets even though it has six
	// characters, and they appear in first-appearance order.
	got := scriptset.SOSS("Сircбe")
	if len(got) != 2 {
		t.Fatalf("SOSS = %v, want 2 entries", got)
	}
	if got[0].String() != "{Cyrl}" || got[1].String() != "{Latn}" {
		t.Errorf("SOSS = [%s %s], want [{Cyrl} {Latn}]", got[0], got[1])
	}
	if n := len(scriptset.SOSS("")); n != 0 {
		t.Errorf("SOSS(\"\") has %d entries, want 0", n)
	}
}

func TestSetAlgebra(t *testing.T) {
	jp := scriptset.NewSet(scriptset.Hira, scriptset.Kana, scriptset.Jpan)
	kr := scriptset.NewSet(scriptset.Hang, scriptset.Kore)

	if got := jp.Len(); got != 3 {
		t.Errorf("Len = %d, want 3", got)
	}
	if !jp.Contains(scriptset.Hira) || jp.Contains(scriptset.Hang) {
		t.Error("Contains is wrong")
	}
	if !jp.Intersect(kr).Empty() {
		t.Error("Intersect of disjoint sets is not empty")
	}
	if got := jp.Union(kr).Len(); got != 5 {
		t.Errorf("Union Len = %d, want 5", got)
	}
	if got := jp.String(); got != "{Hira, Jpan, Kana}" {
		t.Errorf("String = %s, want {Hira, Jpan, Kana}", got)
	}
	if got := (scriptset.Set{}).String(); got != "{}" {
		t.Errorf("empty String = %s, want {}", got)
	}
	if !scriptset.All().IsAll() || scriptset.All().String() != "ALL" {
		t.Error("All is not ALL")
	}
	if scriptset.All().Len() == 0 {
		t.Error("All is empty")
	}
	// Sets are comparable, which SOSS deduplication relies on.
	if scriptset.NewSet(scriptset.Latn) != scriptset.NewSet(scriptset.Latn) {
		t.Error("equal sets do not compare equal")
	}
}

func TestScriptNames(t *testing.T) {
	for _, tt := range []struct {
		sc   scriptset.Script
		code string
	}{
		{scriptset.Latn, "Latn"},
		{scriptset.Hani, "Hani"},
		{scriptset.Jpan, "Jpan"},
		{scriptset.Kore, "Kore"},
		{scriptset.Hanb, "Hanb"},
		{scriptset.Zyyy, "Zyyy"},
	} {
		if got := tt.sc.String(); got != tt.code {
			t.Errorf("Script(%d).String() = %q, want %q", tt.sc, got, tt.code)
		}
		got, ok := scriptset.ParseScript(tt.code)
		if !ok || got != tt.sc {
			t.Errorf("ParseScript(%q) = %v, %v, want %v, true", tt.code, got, ok, tt.sc)
		}
	}
	if _, ok := scriptset.ParseScript("Nope"); ok {
		t.Error(`ParseScript("Nope") reported ok`)
	}
	if got := scriptset.Script(9999).String(); got != "" {
		t.Errorf("out-of-range Script.String() = %q, want empty", got)
	}
}

func TestLevelString(t *testing.T) {
	want := []string{
		"ASCII-Only", "Single Script", "Highly Restrictive",
		"Moderately Restrictive", "Minimally Restrictive", "Unrestricted",
	}
	for i, w := range want {
		if got := scriptset.Level(i).String(); got != w {
			t.Errorf("Level(%d) = %q, want %q", i, got, w)
		}
	}
	if got := scriptset.Level(99).String(); got != "invalid" {
		t.Errorf("Level(99) = %q, want %q", got, "invalid")
	}
	// The levels are ordered from most to least restrictive, so a policy can
	// be expressed as a single comparison.
	if !(scriptset.ASCIIOnly < scriptset.HighlyRestrictive &&
		scriptset.HighlyRestrictive < scriptset.Unrestricted) {
		t.Error("levels are not ordered")
	}
}

func TestAllowed(t *testing.T) {
	for _, r := range []rune{'a', 'Z', '0', '_', '-', 0x4E00, 0xAC00, 0x3042} {
		if !scriptset.Allowed(r) {
			t.Errorf("Allowed(U+%04X) = false, want true", r)
		}
	}
	for _, r := range []rune{' ', '@', '!', 0x3105, 0x1D5A2, 0x0378} {
		if scriptset.Allowed(r) {
			t.Errorf("Allowed(U+%04X) = true, want false", r)
		}
	}
	if scriptset.AllAllowed("\xff") {
		t.Error("AllAllowed on invalid UTF-8 = true, want false")
	}
}

// TestResolveIsIntersection checks Resolve against the definition rather than
// against a table: the resolved script set is the intersection of the
// augmented script sets of every character.
func TestResolveIsIntersection(t *testing.T) {
	for _, s := range []string{
		"abc", "日本語", "한글漢字",
		"ラーメン", "Сircбe", "a1b2",
	} {
		want := scriptset.All()
		for _, r := range s {
			want = want.Intersect(scriptset.Augmented(r))
		}
		if got := scriptset.Resolve(s); got != want {
			t.Errorf("Resolve(%q) = %s, want %s", s, got, want)
		}
	}
}

func FuzzLevelOf(f *testing.F) {
	for _, s := range []string{
		"", "abc", "日本語", "한국어", "Сircбe",
		"\xff\xfe", "a·b", "\U0001D5A2",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		lvl := scriptset.LevelOf(s)
		if lvl < scriptset.ASCIIOnly || lvl > scriptset.Unrestricted {
			t.Fatalf("LevelOf(%q) = %d, out of range", s, lvl)
		}
		if lvl.String() == "invalid" {
			t.Fatalf("LevelOf(%q) produced an invalid level", s)
		}
		// A string is single-script exactly when it is not mixed-script, and
		// Resolve must agree with both.
		single := scriptset.IsSingleScript(s)
		if single == scriptset.IsMixedScript(s) {
			t.Fatalf("IsSingleScript and IsMixedScript agree on %q", s)
		}
		if single != !scriptset.Resolve(s).Empty() {
			t.Fatalf("IsSingleScript disagrees with Resolve on %q", s)
		}
		// Anything that reaches a level below Unrestricted is in the profile.
		if lvl < scriptset.Unrestricted && !scriptset.AllAllowed(s) {
			t.Fatalf("LevelOf(%q) = %v but the string is not in the identifier profile", s, lvl)
		}
	})
}

func BenchmarkLevelOf(b *testing.B) {
	const s = "ラーメンramen"
	b.ReportAllocs()
	for b.Loop() {
		scriptset.LevelOf(s)
	}
}

func BenchmarkAugmented(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		scriptset.Augmented(0x65E5)
	}
}
