package main

import (
	"context"
	"gopkg.zouai.io/colossus/clog"
)

func main() {
	ctx, _ := clog.NewRootLogger(context.Background(), "TestApp")

	clog.Info(ctx, "Starting the application!")

	module1Ctx := clog.WithPrefix(ctx, "Module1")

	clog.Warnf(module1Ctx, "Initialized Module1 with '%s'", "Happyness")

	module1RequestCtx := clog.WithFields(module1Ctx, map[string]interface{}{
		"requestID": "Ted",
	})

	clog.Error(module1RequestCtx, "Got some weird request")
}
