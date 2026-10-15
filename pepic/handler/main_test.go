package handler

import (
	"fmt"
	"os"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
)

func TestMain(m *testing.M) {
	if err := config.Load("../../etc/pepic/config.yml"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
