package export

import "testing"

func TestStableJSONSortsAndSanitizes(t *testing.T) {
	value := map[string]any{
		"b": "one\t two   three \r\n",
		"a": map[string]any{
			"10": "ten",
			"2":  "two",
			"x":  "ex",
		},
	}
	got, err := StableJSON(value, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":{"2":"two","10":"ten","x":"ex"},"b":"one  two three\n"}`
	if string(got) != want {
		t.Fatalf("StableJSON mismatch\nwant: %s\n got: %s", want, got)
	}
}
