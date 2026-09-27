// Package testsupport contains shared Cucumber plumbing, never product behavior.
package testsupport

import (
	"encoding/json"
	"fmt"
	"github.com/cucumber/godog"
	"os"
	"reflect"
	"testing"
)

func Run(t *testing.T, initialize func(*godog.ScenarioContext)) {
	t.Helper()
	suite := godog.TestSuite{ScenarioInitializer: initialize, Options: &godog.Options{
		Format: "pretty", Paths: []string{"../features"}, Tags: "~@todo", Strict: true, TestingT: t,
		ShowStepDefinitions: os.Getenv("GODOG_STEPS") == "1",
	}}
	if suite.Run() != 0 {
		t.Fatal("Cucumber scenarios failed")
	}
}
func Log(format string, args ...any) { fmt.Printf(format+"\n", args...) }
func Equal(actual, expected any) error {
	a, err := json.Marshal(actual)
	if err != nil {
		return err
	}
	b, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	var av, bv any
	if err = json.Unmarshal(a, &av); err != nil {
		return err
	}
	if err = json.Unmarshal(b, &bv); err != nil {
		return err
	}
	Log("actual=%s expected=%s", a, b)
	if !reflect.DeepEqual(av, bv) {
		return fmt.Errorf("got %s; want %s", a, b)
	}
	return nil
}
func JSON(actual any, doc *godog.DocString) error {
	var expected any
	if err := json.Unmarshal([]byte(doc.Content), &expected); err != nil {
		return err
	}
	return Equal(actual, expected)
}
