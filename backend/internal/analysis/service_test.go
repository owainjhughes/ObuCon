package analysis

import (
	"reflect"
	"testing"

	"obucon/internal/lang/ja"
)

func TestUniqueLemmas(t *testing.T) {
	cases := []struct {
		name string
		in   []ja.Token
		want []string
	}{
		{
			name: "empty input",
			in:   nil,
			want: []string{},
		},
		{
			name: "single token, surface equals lemma",
			in: []ja.Token{
				{Surface: "猫", Lemma: "猫"},
			},
			want: []string{"猫"},
		},
		{
			name: "duplicate lemmas deduplicated, first-seen order preserved",
			in: []ja.Token{
				{Surface: "食べる", Lemma: "食べる"},
				{Surface: "飲む", Lemma: "飲む"},
				{Surface: "食べる", Lemma: "食べる"},
			},
			want: []string{"食べる", "飲む"},
		},
		{
			name: "surface differs from lemma: both added, lemma first",
			in: []ja.Token{
				{Surface: "飲みます", Lemma: "飲む"},
			},
			want: []string{"飲む", "飲みます"},
		},
		{
			name: "empty lemma skipped, surface still added when present",
			in: []ja.Token{
				{Surface: "x", Lemma: ""},
			},
			want: []string{"x"},
		},
		{
			name: "empty surface skipped when equal to empty lemma",
			in: []ja.Token{
				{Surface: "", Lemma: ""},
			},
			want: []string{},
		},
		{
			name: "multiple tokens mixed",
			in: []ja.Token{
				{Surface: "飲みます", Lemma: "飲む"},
				{Surface: "飲む", Lemma: "飲む"},
				{Surface: "コーヒー", Lemma: "コーヒー"},
			},
			want: []string{"飲む", "飲みます", "コーヒー"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := uniqueLemmas(tc.in)
			// Both nil-empty and len-0 non-nil count as "empty" for our purposes.
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("uniqueLemmas:\n got  %v\n want %v", got, tc.want)
			}
		})
	}
}
