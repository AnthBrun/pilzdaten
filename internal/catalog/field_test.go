package catalog

import (
	"reflect"
	"testing"
)

func monthsOf(text string) []int {
	months := seasonMonths(text)
	result := make([]int, 0)
	for month := 1; month <= 12; month++ {
		if months[month] {
			result = append(result, month)
		}
	}
	return result
}

func TestSeasonMonthsParsesRangesAndSingleSeasons(t *testing.T) {
	tests := []struct {
		text string
		want []int
	}{
		{"Mischwald, Symbiosepilz, Frühsommer bis Spätherbst.", []int{6, 7, 8, 9, 10, 11}},
		{"Auf Holz, Herbst.", []int{9, 10, 11}},
		{"Spätherbst bis Frühjahr", []int{1, 2, 3, 4, 5, 11, 12}},
		{"Sommer-Herbst", []int{6, 7, 8, 9, 10, 11}},
		{"An totem Laubholz, ganzjährig, Hauptzeit: Sommer bis Herbst.", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{"Keine Angabe zur Zeit", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{"Frühling, Herbst", []int{3, 4, 5, 9, 10, 11}},
	}
	for _, test := range tests {
		if got := monthsOf(test.text); !reflect.DeepEqual(got, test.want) {
			t.Errorf("seasonMonths(%q) = %v, want %v", test.text, got, test.want)
		}
	}
}

func TestForestTagsRecognizeTreesWithoutFalsePositives(t *testing.T) {
	tests := []struct {
		text string
		want []string
	}{
		{"Mischwald, gern Fichten, seltener bei Eichen", []string{"wald_laub", "wald_nadel", "baum_eiche", "baum_fichte"}},
		{"Unter Stieleichen und Rotbuchen", []string{"wald_laub", "baum_buche", "baum_eiche"}},
		{"Auf weichen Böden in feuchten Bereichen, Viehweiden", []string{}},
		{"Bei Hainbuchen", []string{}},
		{"Kiefernwälder auf Sand", []string{"wald_nadel", "baum_kiefer"}},
	}
	for _, test := range tests {
		if got := forestTagsFor(test.text); !reflect.DeepEqual(got, test.want) {
			t.Errorf("forestTagsFor(%q) = %v, want %v", test.text, got, test.want)
		}
	}
}

func TestCommonnessReadsFrequencyHints(t *testing.T) {
	tests := []struct {
		text string
		tags []string
		want int
	}{
		{"Laubwald, sehr häufig.", nil, 3},
		{"Mischwald, häufig, weit verbreitet.", nil, 3},
		{"Sandige Kiefernwälder, sehr selten, RL3 Deutschland", nil, -4},
		{"Mischwald, seltener bei Eichen", []string{"nicht_selten"}, 1},
	}
	for _, test := range tests {
		if got := commonness(map[string]any{"vorkommen": test.text}, test.tags); got != test.want {
			t.Errorf("commonness(%q) = %d, want %d", test.text, got, test.want)
		}
	}
}

func TestRelevanceRanksImportantAndCommonSpeciesFirst(t *testing.T) {
	species := []*Species{
		{Slug: "rare", NameTrivial: "Aaa", Priority: 3, PhotoCount: 50},
		{Slug: "known-few", NameTrivial: "Ccc", Priority: 1, PhotoCount: 2},
		{Slug: "known-many", NameTrivial: "Ddd", Priority: 1, PhotoCount: 80},
		{Slug: "known-huge", NameTrivial: "Zzz", Priority: 1, PhotoCount: 600},
		{Slug: "mid-common", NameTrivial: "Eee", Priority: 2, Commonness: 2},
		{Slug: "mid", NameTrivial: "Bbb", Priority: 2, PhotoCount: 90},
	}
	// Ab 40 Fotos zählt nur noch der Name: 600 Bilder schlagen 80 Bilder nicht.
	want := []string{"known-many", "known-huge", "known-few", "mid-common", "mid", "rare"}
	for i := range species {
		for j := i + 1; j < len(species); j++ {
			if speciesRelevanceLess(species[j], species[i]) {
				species[i], species[j] = species[j], species[i]
			}
		}
	}
	for index, slug := range want {
		if species[index].Slug != slug {
			t.Errorf("rank %d = %q, want %q", index, species[index].Slug, slug)
		}
	}
}
