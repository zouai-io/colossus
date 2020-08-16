package clog

import (
	"context"
	log_prefixed "github.com/chappjc/logrus-prefix"
	"github.com/knq/sdhook"
	"github.com/sirupsen/logrus"
	"gopkg.zouai.io/colossus/colossusconfig"
	"os"
)



type LoggerInterface interface {
	Info(ctx context.Context, msg string)
	Infof(ctx context.Context, format string, args ...interface{})
	Warn(ctx context.Context, msg string)
	Warnf(ctx context.Context, format string, args ...interface{})
	Err(ctx context.Context, err error, msg string)
	Errf(ctx context.Context, err error, format string, args ...interface{})
	Error(ctx context.Context, msg string)
	Errorf(ctx context.Context, format string, args ...interface{})
	Debug(ctx context.Context, msg string)
	Debugf(ctx context.Context, format string, args ...interface{})
	Trace(ctx context.Context, msg string)
	Tracef(ctx context.Context, format string, args ...interface{})

	WithFields(ctx context.Context, fields map[string]interface{}) context.Context
	WithPrefix(ctx context.Context, prefix string) context.Context
}

type Logger struct {
	*logrus.Logger
}

func NewRootLogger(ctx context.Context, appName string) (context.Context, *Logger) {
	logger := logrus.New()
	logger.Formatter = &log_prefixed.TextFormatter{}
	instance := &LogInstance{logger:logger.WithField("prefix", appName), prefix:appName}
	l := &Logger{
		Logger: logger,
	}
	subCtx := context.WithValue(ctx, ctxKey, instance)
	if colossusconfig.DefaultConfig.Colossus.Logging.StackDriver {
		l.EnableStackDriverLogging(subCtx)
	}
	return subCtx, l
}

func (m *Logger) EnableStackDriverLogging(ctx context.Context) *Logger {
	hasTarget := false
	if colossusconfig.DefaultConfig.Colossus.Logging.StackDriver_.UseLoggingAgent {
		h, err := sdhook.New(
			sdhook.GoogleLoggingAgent(),
			sdhook.LogName("colossus"),
		)
		if err != nil {
			m.Logger.Errorf("Error creating stackdriver Logger: %v", err)
			panic(err)
		} else {
			m.Hooks.Add(h)
			logrus.RegisterExitHandler(h.Wait)
			hasTarget = true
		}
	}
	 if colossusconfig.DefaultConfig.Colossus.Logging.StackDriver_.UseApplicationDefaultCredentials || !hasTarget {
		// UseApplicationDefaultCredentials will be the default case
		hostname, err := os.Hostname()
		if err != nil {
			m.Logger.Errorf("Error determining hostname: %v", err)
		}
		h, err := sdhook.New(
			sdhook.GoogleServiceAccountCredentialsFile(colossusconfig.DefaultConfig.Google.Application.Credentials),
			sdhook.Resource("generic_node", map[string]string{
				"project_id": "colossus-test",
				"node_id": hostname,
			}),
			sdhook.LogName("colossus"),
		)
		if err != nil {
			m.Logger.Errorf("Error creating stackdriver Logger: %v", err)
			panic(err)
		}
		m.Hooks.Add(h)
		logrus.RegisterExitHandler(h.Wait)
	}
	return m
}

type keyT string

var ctxKey = keyT("clog")

type LogInstance struct {
	logger logrus.Ext1FieldLogger
	prefix string
}

func logFromCtx(ctx context.Context) *LogInstance {
	val := ctx.Value(ctxKey)
	if val == nil {
		return &LogInstance{
			logger: logrus.WithField("prefix", "ORPHAN CONTEXT"),
		}
	}
	tx, ok := val.(*LogInstance)
	if !ok {
		return &LogInstance{
			logger: logrus.WithField("prefix", "ORPHAN CONTEXT"),
		}
	}
	return tx
}

func WithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	m := logFromCtx(ctx)
	nextInstance := &LogInstance{logger:m.logger.WithFields(fields), prefix:m.prefix}
	return context.WithValue(ctx, ctxKey, nextInstance)
}

func WithPrefix(ctx context.Context, prefix string) context.Context {
	m := logFromCtx(ctx)
	nextInstance := &LogInstance{logger:m.logger.WithField("prefix", m.prefix+"/"+prefix), prefix:m.prefix+"/"+prefix}
	return context.WithValue(ctx, ctxKey, nextInstance)
}

func Info(ctx context.Context, msg string) {
	m := logFromCtx(ctx)
	m.logger.Info(msg)
}
func Infof(ctx context.Context, format string, args ...interface{}){
	m := logFromCtx(ctx)
	m.logger.Infof(format, args...)
}
func Warn(ctx context.Context, msg string){
	m := logFromCtx(ctx)
	m.logger.Warn(msg)
}
func Warnf(ctx context.Context, format string, args ...interface{}){
	m := logFromCtx(ctx)
	m.logger.Warnf(format, args...)
}
func Err(ctx context.Context, err error, msg string){
	m := logFromCtx(ctx)
	m.logger.WithError(err).Error(msg)
}
func Errf(ctx context.Context, err error, format string, args ...interface{}){
	m := logFromCtx(ctx)
	m.logger.WithError(err).Errorf(format, args...)
}
func Error(ctx context.Context, msg string){
	m := logFromCtx(ctx)
	m.logger.Error(msg)
}
func Errorf(ctx context.Context, format string, args ...interface{}){
	m := logFromCtx(ctx)
	m.logger.Errorf(format, args...)
}
func Debug(ctx context.Context, msg string){
	m := logFromCtx(ctx)
	m.logger.Debug(msg)
}
func Debugf(ctx context.Context, format string, args ...interface{}){
	m := logFromCtx(ctx)
	m.logger.Debugf(format, args...)
}
func Trace(ctx context.Context, msg string){
	m := logFromCtx(ctx)
	m.logger.Trace(msg)
}
func Tracef(ctx context.Context, format string, args ...interface{}){
	m := logFromCtx(ctx)
	m.logger.Tracef(format, args...)
}