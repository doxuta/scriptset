package scriptset_test

import (
	"fmt"

	"github.com/doxuta/scriptset"
)

func ExampleLevelOf() {
	for _, s := range []string{"paypal", "ラーメンramen", "abcабв", "hello world"} {
		fmt.Printf("%-14s %s\n", s, scriptset.LevelOf(s))
	}
	// Output:
	// paypal         ASCII-Only
	// ラーメンramen      Highly Restrictive
	// abcабв         Minimally Restrictive
	// hello world    Unrestricted
}

// The levels are ordered from most to least restrictive, so a policy is one
// comparison.
func ExampleLevel() {
	allow := func(name string) string {
		if scriptset.LevelOf(name) > scriptset.ModeratelyRestrictive {
			return "rejected"
		}
		return "accepted"
	}
	fmt.Println("김치kimchi", allow("김치kimchi"))
	fmt.Println("abcабв   ", allow("abcабв"))
	// Output:
	// 김치kimchi accepted
	// abcабв    rejected
}

func ExampleIsMixedScript() {
	// Three of these six letters are Cyrillic.
	fmt.Println(scriptset.IsMixedScript("Сirсlе"))
	// Hangul and Han, which is ordinary Korean, not an attack.
	fmt.Println(scriptset.IsMixedScript("한글漢字"))
	// Output:
	// true
	// false
}

func ExampleResolve() {
	fmt.Println(scriptset.Resolve("ねガ"))
	fmt.Println(scriptset.Resolve("日本語"))
	fmt.Println(scriptset.Resolve("Сirсlе"))
	// Output:
	// {Jpan}
	// {Hanb, Hani, Jpan, Kore}
	// {}
}

func ExampleAugmented() {
	// Han carries the three writing systems as well as Hani.
	fmt.Println(scriptset.Augmented('日'))
	// U+30FC has Script=Common but Script_Extensions={Hira, Kana}.
	fmt.Println(scriptset.Augmented('ー'))
	// Output:
	// {Hanb, Hani, Jpan, Kore}
	// {Hira, Jpan, Kana}
}

func ExampleCovers() {
	kore := scriptset.NewSet(scriptset.Kore)
	fmt.Println(scriptset.Covers(kore, "한글漢字"))
	fmt.Println(scriptset.Covers(kore, "ラーメン"))
	// Output:
	// true
	// false
}

func ExampleAllowed() {
	// Bopomofo became a Limited Use script in Unicode 17, which took its
	// letters out of the identifier profile.
	fmt.Println(scriptset.Allowed('ㄅ'))
	fmt.Println(scriptset.Allowed('한'))
	// Output:
	// false
	// true
}
