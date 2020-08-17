package clog_test

import (
	"context"
	"github.com/bmizerany/assert"
	"github.com/chappjc/logrus-prefix"
	"github.com/sirupsen/logrus"
	"gopkg.zouai.io/colossus/clog"
	"gopkg.zouai.io/colossus/colossusconfig"
	"os"
	"reflect"
	"testing"
)

func TestDisableConsoleLogging(t *testing.T) {
	// Zero out the config
	colossusconfig.DefaultConfig = &colossusconfig.Config{}
	// Disable the console
	colossusconfig.DefaultConfig.Colossus.Logging.DisableConsole = true
	_, logger := clog.NewRootLogger(context.Background(), "TestDisableConsoleLogging")
	assert.NotEqual(t, logger.Out, os.Stdout, "Logger is routing to standard out when disable console is true")
	assert.NotEqual(t, logger.Out, os.Stderr, "Logger is routing to standard error when disable console is true")
}

func TestForceConsoleJSON(t *testing.T) {
	// Zero out the config
	colossusconfig.DefaultConfig = &colossusconfig.Config{}
	// Force a TTY and then force JSON
	colossusconfig.DefaultConfig.Colossus.Logging.ForceISaTTY = true
	colossusconfig.DefaultConfig.Colossus.Logging.ForceConsoleJSON = true

	_, logger := clog.NewRootLogger(context.Background(), "TestDisableConsoleLogging")
	switch logger.Formatter.(type) {
	case *logrus.JSONFormatter:
		// This is good
		break
	default:
		t.Fatalf("Expected '*logrus.JSONFormatter' got '%s'", reflect.TypeOf(logger.Formatter).String())
	}
}

func TestForceISaTTY(t *testing.T) {
	// Zero out the config
	colossusconfig.DefaultConfig = &colossusconfig.Config{}
	// Force a TTY
	colossusconfig.DefaultConfig.Colossus.Logging.ForceISaTTY = true
	_, logger := clog.NewRootLogger(context.Background(), "TestForceISaTTY")
	switch logger.Formatter.(type) {
	case *prefixed.TextFormatter:
		// This is good
		break
	default:
		t.Fatalf("Expected '*prefixed.TextFormatter' got '%s'", reflect.TypeOf(logger.Formatter).String())
	}
}