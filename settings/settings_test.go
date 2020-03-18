package settings

import (
	"fmt"
	"testing"
)

type testSettingsData struct {
	Foo int
	Bar string
}

var testData testSettingsData
var testSet *Settings

func initTests(t *testing.T) error {
	testData = testSettingsData{}

	var err error
	testSet, err = New("settings_test.json", &testData, false)

	if err != nil {
		if t != nil {
			t.Error("Error init settings:", err)
		}
		return err
	}

	return nil
}

func TestSave(t *testing.T) {
	fmt.Println("Save TEST")

	err := initTests(t)
	if err != nil {
		return
	}

	err = testSet.SaveSettings()
	if err != nil {
		t.Error("Error saving settings:", err)
	}
}

func TestLoad(t *testing.T) {
	fmt.Println("Load TEST")

	err := initTests(t)
	if err != nil {
		return
	}

	err = testSet.LoadSettings()
	if err != nil {
		t.Error("Error loading settings:", err)
	}
}

func TestLoadSave(t *testing.T) {
	if t != nil {
		fmt.Println("Save and reload TEST")
	}

	err := initTests(t)
	if err != nil {
		return
	}

	testData = testSettingsData{
		Foo: -1,
		Bar: "mh",
	}

	err = testSet.SaveSettings()
	if err != nil {
		if t != nil {
			t.Error("Error saving settings:", err)
		}
		return
	}

	testData2 := testData
	testData = testSettingsData{}

	err = testSet.LoadSettings()
	if err != nil {
		if t != nil {
			t.Error("Error loading settings:", err)
		}
		return
	}

	if testData != testData2 {
		if t != nil {
			t.Error("Loaded data mismatch: ", testData2, testData)
		}
		return
	}
}

func BenchmarkLoadSave(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestLoadSave(nil)
	}
}
