# scriptset

Mixed-script detection and restriction levels for Go, from section 5 of
[UTS #39, Unicode Security Mechanisms](https://www.unicode.org/reports/tr39/).

It answers two questions about a string:

- **Is it mixed-script?** `Сirсlе` with three Cyrillic letters is. `한글漢字` is not,
  even though it uses two scripts, and neither is `ラーメン`.
- **What restriction level is it?** One of the six the standard defines, from
  ASCII-Only through Highly Restrictive to Unrestricted.

Pure standard library, no dependencies, tables generated from Unicode 17.0.0.

## Why

Mixed-script detection is the part of a homograph defence that is easy to get
subtly wrong, and the CJK rules are where it goes wrong.

The naive version compares the **Script** property of each character and flags
anything with more than one. That version reports `한글漢字` (Hangul + Han) and
`漢字かな` (Han + Hiragana) as attacks, because they genuinely use two scripts.
Ordinary Korean and Japanese words are false positives, so the check either gets
turned off or gets a hand-written CJK exception that is wrong in a different way.

UTS #39 solves this with **augmented script sets**. A Han character carries not
only `Hani` but also the writing systems `Hanb`, `Jpan` and `Kore`; Hiragana and
Katakana carry `Jpan`; Hangul carries `Kore`. The intersection over `한글漢字` is
`{Kore}` — non-empty, so single-script, so not an attack. That is the whole
trick, and it is why the spec needs three ISO 15924 codes that are not Script
property values at all.

The second half is that the spec requires **Script_Extensions**, not Script.
They differ, and the standard library only has the latter:

| Code point | `Script` (`unicode.Scripts`) | `Script_Extensions` |
|---|---|---|
| U+30FC KATAKANA-HIRAGANA PROLONGED SOUND MARK | `Common` | `{Hira, Kana}` |
| U+00B7 MIDDLE DOT | `Common` | 16 scripts, **not** including Cyrillic |

An implementation that reads `Common` as a wildcard calls the Cyrillic string
`а·б` single-script. It is not: `U+00B7` does not carry Cyrillic, so the
resolved set is empty.

## Install

```
go get github.com/doxuta/scriptset
```

## Try it

One command, no arguments, no setup:

```
go run github.com/doxuta/scriptset/cmd/scriptset@latest
```

```
scriptset — UTS #39 section 5, Unicode 17.0.0

LEVEL                   RESOLVED SET              SINGLE  STRING
ASCII-Only              {Latn}                    yes     "paypal"  — plain ASCII
Minimally Restrictive   {}                        no      "pаypal"  — the 'a' is U+0430 CYRILLIC SMALL LETTER A
Single Script           {Hanb, Hani, Jpan, Kore}  yes     "日本語"  — Han only: one script, four values in the resolved set
Single Script           {Kore}                    yes     "한글漢字"  — Hangul and Han: single-script via Kore
Highly Restrictive      {}                        no      "ラーメンramen"  — Latin + Japanese is an approved combination
Highly Restrictive      {}                        no      "김치kimchi"  — Latin + Korean is an approved combination
Minimally Restrictive   {}                        no      "abcабв"  — Latin + Cyrillic is not
Moderately Restrictive  {}                        no      "abcไทย"  — Latin + one other Recommended script
Minimally Restrictive   {}                        no      "а·б"  — U+00B7 MIDDLE DOT carries 16 scripts, none of them Cyrillic
Unrestricted            {Latn}                    yes     "hello world"  — the space is not in the identifier profile
```

Pass your own strings as arguments, and `-v` to see the augmented script set of
every character.

## Use

```go
import "github.com/doxuta/scriptset"

// Reject an identifier that is looser than a policy allows. The levels are
// ordered, so a policy is one comparison.
if scriptset.LevelOf(username) > scriptset.ModeratelyRestrictive {
    return errors.New("username mixes scripts in a way we do not allow")
}

scriptset.IsMixedScript("Сirсlе")   // true  — three Cyrillic letters
scriptset.IsMixedScript("한글漢字")   // false — resolved set is {Kore}
scriptset.Resolve("ねガ").String()   // "{Jpan}"
scriptset.Augmented('日').String()   // "{Hanb, Hani, Jpan, Kore}"
scriptset.Covers(scriptset.NewSet(scriptset.Kore), "한글漢字") // true
scriptset.Allowed('ㄅ')              // false — Restricted since Unicode 17
```

`LevelOf` follows the eight steps of section 5.2 exactly, including step 1: a
string containing any character outside the identifier profile is
`Unrestricted`, whatever its scripts. That is why `"hello world"` is
`Unrestricted` — the space is not an identifier character. These are identifier
rules, not prose rules.

## Two things worth knowing about Unicode 17

**Bopomofo cannot reach Highly Restrictive any more.** Section 5.2 lists
`Latin + Han + Bopomofo` as one of the three approved combinations. But UAX #31
Table 5 moved Bopomofo to Limited Use in Unicode 17, so Bopomofo letters went
from `Identifier_Type=Recommended` / `Identifier_Status=Allowed` in Unicode 16
to `Limited_Use` / `Restricted` in Unicode 17. Any string containing one now
fails step 1 and comes back `Unrestricted`. A scan of the whole code space finds
**0** code points that carry `Hanb`, are not Han, and are still Allowed, so the
`{Hanb}` branch can only be satisfied by characters that `{Jpan}` and `{Kore}`
also satisfy. `TestBopomofoIsUnreachableInUnicode17` asserts this.

**Table 1a has drifted from the data.** The table lists U+3006 IDEOGRAPHIC
CLOSING MARK with `Script_Extensions` of `{Hani, Hira, Kana}`;
`ScriptExtensions.txt` in UCD 17.0.0 gives it `{Hani}`. The resolved set of the
table's `〆切` example is unchanged, because the Han augmentation rule supplies
`Hanb`, `Jpan` and `Kore` either way and the second character intersects `Hira`
and `Kana` away. All eight rows of Table 1a still reproduce; only the
intermediate column differs. `TestTable1aScriptExtensionsDrift` pins it so a
future data change is noticed.

## Limitations

- **No confusable detection.** Skeletons and the confusables mapping of section
  4 are a separate problem and are already solved in Go — see Prior art. This
  package is the other half, and the two compose: skeleton first, then
  `LevelOf`.
- **No minimal cover sets.** Section 5.1 notes that an API returning "the
  scripts in a string" typically returns a minimal cover set. Computing one is
  set cover, and the restriction-level algorithm does not need it, so it is not
  here. `Resolve` gives the resolved script set and `SOSS` gives the raw set of
  script sets, which is what the algorithm actually uses.
- **No identifier well-formedness check.** Section 5.2 requires the string to be
  "well-formed according to whatever general syntactic constraints are in
  force". In Go that is normally a UAX #31 `ID_Start`/`ID_Continue` test, and it
  is the caller's job. `LevelOf` applies the identifier *profile* (section 3.1),
  not the identifier *syntax*.
- **No mixed-number detection.** Section 5.3 is not implemented.
- **One profile.** The General Security Profile of section 3.1 is built in.
  There is no way to supply your own allowed-character set.
- **Whole-string only.** Real spoofing checks usually run per word or per label.
  Split the input yourself; this package does not tokenise.
- **ALL includes the three writing systems.** The spec defines ALL as "the set
  of all script values", and `Hanb`, `Jpan` and `Kore` are not Script values.
  Including them cannot change a restriction level — an ALL entry always
  contains `Latn` and so is always removed at step 5 — but it does mean
  `Resolve` on an all-Common string reports them.
- **Tables are pinned to Unicode 17.0.0.** Regenerate with `go run gen.go`,
  which downloads from unicode.org. The security data comes from
  `.../Public/security/latest/` because the versioned directory for this release
  is not published yet; the generator refuses to run if that file's
  `# Version:` header is not the pinned release.

## Prior art

- [`mtibben/confusables`](https://github.com/mtibben/confusables) implements the
  UTS #39 section 4 skeleton algorithm for Go. Its source carries
  `// TODO: implement xidmodifications.txt restricted characters`, which is the
  identifier profile this package builds on. `scriptset` deliberately does not
  duplicate its skeleton work.
- [`eskriett/confusables`](https://github.com/eskriett/confusables) is another
  Go confusables mapping, with `ToASCII` and `ToNumber` on top.
- [ICU](https://unicode-org.github.io/icu/userguide/transforms/general/) has had
  `SpoofChecker` with restriction levels for years, in C++ and Java. This is a
  much smaller thing in Go, not a port of it.
- The data is the Unicode Character Database and the UTS #39 security data,
  © Unicode, Inc., used under the
  [Unicode terms of use](https://www.unicode.org/terms_of_use.html).

## Development

```
go run gen.go      # regenerate tables.go from unicode.org (needs network)
go test ./...
go test -race -cover ./...
```

CI runs gofmt, vet, race tests, a 30-second fuzz of `LevelOf`, and govulncheck.

## Disclosure

Built with an AI-agent workflow: the spec sections were read and the algorithm,
tables and tests were written in one session with Claude Code, then checked
against the standard's own worked examples. The two Unicode 17 observations
above were found by running the code against the data, not by reading about
them. Errors are mine.
