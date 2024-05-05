package utils

import (
	"slices"
	"testing"
)

func getConfigLoadPath(name string) string {
	return getTestFilePath("configload", name)
}

func TestConfigStackBaseOnly(t *testing.T) {
	filepath := getConfigLoadPath("config_only.json")
	check := func(name string, data []byte) error {
		if len(data) != 0 {
			t.Fatalf("Expected empty data, got %s", string(data))
		}
		if name != filepath {
			t.Fatalf("Expected filepath %s, got %s", filepath, name)
		}
		return nil
	}
	read, err := ReadConfigStack(filepath, check, 0)
	if err != nil {
		t.Fatalf("Error on ReadConfigStack: %s", err)
	}
	if len(read) != 1 || read[0] != filepath {
		t.Fatalf("Got incorrect read list: %s", read)
	}
	read, err = ReadConfigStack(filepath, check, 9)
	if err != nil {
		t.Fatalf("Error on ReadConfigStack: %s", err)
	}
	if len(read) != 1 || read[0] != filepath {
		t.Fatalf("Got incorrect read list: %s", read)
	}
}

func TestConfigStackNobase(t *testing.T) {
	filepath := getConfigLoadPath("config_nobase.yaml")
	realfilepath := getConfigLoadPath("config_nobase1.yaml")
	check := func(name string, data []byte) error {
		if len(data) != 0 {
			t.Fatalf("Expected empty data, got %s", string(data))
		}
		if name != realfilepath {
			t.Fatalf("Expected filepath %s, got %s", filepath, name)
		}
		return nil
	}
	read, err := ReadConfigStack(filepath, check, 0)
	if err != nil {
		t.Fatalf("Error on ReadConfigStack: %s", err)
	}
	if len(read) != 0 {
		t.Fatalf("Got incorrect read list (expected empty): %s", read)
	}
	read, err = ReadConfigStack(filepath, check, 9)
	if err != nil {
		t.Fatalf("Error on ReadConfigStack: %s", err)
	}
	if len(read) != 1 || read[0] != realfilepath {
		t.Fatalf("Got incorrect read list (expected yaml file): %s", read)
	}
}

func TestConfigStackSome(t *testing.T) {
	filepath := getConfigLoadPath("config_some.txt")
	data := make(map[string]string)
	data[getConfigLoadPath("config_some.txt")] = "basefile\n"
	data[getConfigLoadPath("config_some0.txt")] = "file0\n"
	data[getConfigLoadPath("config_some1.txt")] = "file1\n"
	data[getConfigLoadPath("config_some2.txt")] = "file2\n"
	check := func(name string, fdata []byte) error {
		fsdata := string(fdata)
		expected, ok := data[name]
		if !ok {
			t.Fatalf("Given unexpected file: %s", name)
		}
		if expected != fsdata {
			t.Fatalf("Unexpected file data: %s vs %s", fsdata, expected)
		}
		return nil
	}
	for i := range 10 {
		read, err := ReadConfigStack(filepath, check, i)
		if err != nil {
			t.Fatalf("Error on ReadConfigStack: %s", err)
		}
		if len(read) != min(i+1, 4) {
			t.Fatalf("Got incorrect read list (expected 1): %s", read)
		}
	}
}

func TestConfigStackNoext(t *testing.T) {
	filepath := getConfigLoadPath("config_noext")
	expected := make([]string, 2)
	expected[0] = getConfigLoadPath("config_noext3")
	expected[1] = getConfigLoadPath("config_noext5")
	// NOTE: don't include the noext9, we want to make sure
	// files outside the range aren't included
	check := func(name string, fdata []byte) error {
		if len(fdata) != 0 {
			t.Fatalf("Expected empty data, got %s", string(fdata))
		}
		if slices.Index(expected, name) < 0 {
			t.Fatalf("Given unexpected file: %s", name)
		}
		return nil
	}
	read, err := ReadConfigStack(filepath, check, 8)
	if err != nil {
		t.Fatalf("Error on ReadConfigStack: %s", err)
	}
	if !slices.Equal(read, expected) {
		t.Fatalf("Unexpected read result: %s vs %s", read, expected)
	}
}
