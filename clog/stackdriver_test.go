package clog_test

import (
	"context"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gopkg.zouai.io/colossus/clog"
	"gopkg.zouai.io/colossus/colossusconfig"
	"testing"
)

func TestStackDriver(t *testing.T) {
	ctx := context.Background()
	colossusconfig.DefaultConfig.Colossus.Logging.StackDriver = true
	colossusconfig.DefaultConfig.Colossus.Logging.StackDriver_.UseApplicationDefaultCredentials = true
	ctx, logger := clog.NewRootLogger(ctx, "TestApp")
	assert.NotNil(t, logger)
	uuid := uuid.NewV4()
	subCtx := clog.WithFields(ctx, map[string]interface{}{
		"code": uuid.String(),
	})
	clog.Infof(subCtx, "InfoTest %s", uuid.String())
	logrus.Exit(0)
}
