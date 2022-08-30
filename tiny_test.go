package tinyq

import (
	"github.com/lucagez/tinyq/testutil"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// db pool factory is initialized by importing it

	code := m.Run()

	if err := testutil.PG.Teardown(); err != nil {
		log.Fatalln("could not purge db pool:", err)
	}

	os.Exit(code)
}
