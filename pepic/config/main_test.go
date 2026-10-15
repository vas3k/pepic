package config

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := Load("../../etc/pepic/config.yml"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
