package catalog

import "testing"

func TestListGroupsGermanUmlautsAndNonLetterNames(t *testing.T) {
	catalog := &Catalog{all: []*Species{
		{NameTrivial: "Zwiebelpilz"},
		{NameTrivial: "32-sporiger Pilz"},
		{NameTrivial: "!Sonderpilz"},
		{NameTrivial: "Ästiger Pilz"},
		{NameTrivial: "Apfelpilz"},
		{NameTrivial: "Zitterpilz"},
		{NameTrivial: "Ölbaumpilz"},
		{NameTrivial: "Überhäuteter Pilz"},
	}}

	tests := []struct {
		letter string
		want   []string
	}{
		{letter: "A", want: []string{"Apfelpilz", "Ästiger Pilz"}},
		{letter: "O", want: []string{"Ölbaumpilz"}},
		{letter: "U", want: []string{"Überhäuteter Pilz"}},
		{letter: "Z", want: []string{"Zitterpilz", "32-sporiger Pilz", "Zwiebelpilz"}},
		{letter: "#", want: []string{"!Sonderpilz"}},
	}
	for _, test := range tests {
		results, pages := catalog.List(test.letter, 1, 50)
		if pages != 1 || len(results) != len(test.want) {
			t.Fatalf("List(%q) returned %d results and %d pages; want %d results and 1 page", test.letter, len(results), pages, len(test.want))
		}
		for index, want := range test.want {
			if results[index].NameTrivial != want {
				t.Errorf("List(%q)[%d] = %q; want %q", test.letter, index, results[index].NameTrivial, want)
			}
		}
	}

	filtered, _, _ := catalog.Filter("", nil, 1, 25, "", 1, 50)
	for index := 1; index < len(filtered); index++ {
		if speciesNameLess(filtered[index], filtered[index-1]) {
			t.Errorf("Filter results are not alphabetic: %q appears before %q", filtered[index-1].NameTrivial, filtered[index].NameTrivial)
		}
	}
	if got := germanNumberWords(32); got != "zweiunddreißig" {
		t.Errorf("germanNumberWords(32) = %q, want %q", got, "zweiunddreißig")
	}

	letters := catalog.Letters()
	if len(letters) != 27 || letters[len(letters)-1] != "#" {
		t.Fatalf("Letters() ends with %q and has %d entries; want 26 letters followed by #", letters[len(letters)-1], len(letters))
	}
}

func TestSearchRanksTitleMatchesByRelevance(t *testing.T) {
	catalog := &Catalog{all: []*Species{
		{Slug: "synonym", NameTrivial: "Anderer Pilz", Synonyme: []string{"Steinpilz"}},
		{Slug: "contains-k", NameTrivial: "Kiefern-Steinpilz"},
		{Slug: "scientific", NameTrivial: "Waldpilz", NameWissenschaftlich: "Steinpilzus example"},
		{Slug: "prefix", NameTrivial: "Steinpilze"},
		{Slug: "contains-b", NameTrivial: "Birkensteinpilz"},
		{Slug: "exact", NameTrivial: "Steinpilz"},
	}}
	want := []string{"exact", "prefix", "contains-b", "contains-k", "scientific", "synonym"}

	results := catalog.Search("steinpilz", 0)
	if len(results) != len(want) {
		t.Fatalf("Search returned %d results, want %d", len(results), len(want))
	}
	for index, slug := range want {
		if results[index].Slug != slug {
			t.Errorf("Search result %d = %q, want %q", index, results[index].Slug, slug)
		}
	}
	if limited := catalog.Search("steinpilz", 2); len(limited) != 2 || limited[0].Slug != "exact" || limited[1].Slug != "prefix" {
		t.Errorf("limited Search results = %#v, want exact and prefix matches", limited)
	}

	filtered, total, _ := catalog.Filter("steinpilz", nil, 1, 25, "", 1, 50)
	if total != len(want) || len(filtered) != len(want) {
		t.Fatalf("Filter returned %d results and total %d, want %d", len(filtered), total, len(want))
	}
	for index, slug := range want {
		if filtered[index].Slug != slug {
			t.Errorf("Filter result %d = %q, want %q", index, filtered[index].Slug, slug)
		}
	}
}
