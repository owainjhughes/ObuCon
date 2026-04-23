package config

import (
	"reflect"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("OBU_TEST_KEY", "hello")
		if got := getEnv("OBU_TEST_KEY", "fallback"); got != "hello" {
			t.Errorf("getEnv set: got %q, want %q", got, "hello")
		}
	})

	t.Run("returns fallback when unset", func(t *testing.T) {
		if got := getEnv("OBU_TEST_MISSING_KEY_XYZ", "fallback"); got != "fallback" {
			t.Errorf("getEnv unset: got %q, want %q", got, "fallback")
		}
	})

	t.Run("returns empty string when explicitly set to empty", func(t *testing.T) {
		t.Setenv("OBU_TEST_EMPTY", "")
		if got := getEnv("OBU_TEST_EMPTY", "fallback"); got != "" {
			t.Errorf("getEnv empty-set: got %q, want %q", got, "")
		}
	})
}

func TestGetEnvBool(t *testing.T) {
	cases := []struct {
		name   string
		set    bool
		value  string
		fallbk bool
		want   bool
	}{
		{"unset returns fallback true", false, "", true, true},
		{"unset returns fallback false", false, "", false, false},
		{"true literal", true, "true", false, true},
		{"false literal", true, "false", true, false},
		{"1 parses true", true, "1", false, true},
		{"0 parses false", true, "0", true, false},
		{"garbage returns fallback true", true, "not-a-bool", true, true},
		{"garbage returns fallback false", true, "maybe", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := "OBU_TEST_BOOL"
			if tc.set {
				t.Setenv(key, tc.value)
			} else {
				// ensure unset
				t.Setenv(key, "sentinel")
				// t.Setenv doesn't support unset, so use a distinct unused key instead
				key = "OBU_TEST_BOOL_UNSET_XYZ"
			}
			if got := getEnvBool(key, tc.fallbk); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	defaults := []string{"a", "b"}

	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"simple csv", "x,y,z", []string{"x", "y", "z"}},
		{"trims whitespace", "  foo , bar,baz  ", []string{"foo", "bar", "baz"}},
		{"drops empty segments", "x,,y", []string{"x", "y"}},
		{"empty string returns defaults", "", defaults},
		{"only commas returns defaults", ",,,", defaults},
		{"only whitespace returns defaults", "  ,  , ", defaults},
		{"single value", "only", []string{"only"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCSV(tc.in, defaults)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitCSV(%q): got %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
