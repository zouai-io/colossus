package clog

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	log_prefixed "github.com/chappjc/logrus-prefix"
	"github.com/knq/jwt/gserviceaccount"
	"github.com/knq/sdhook"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.zouai.io/colossus/colossusconfig"
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
	SubLoggerWithFields(ctx context.Context, fields map[string]interface{}) LoggerInterface
	SubLoggerWithPrefix(ctx context.Context, prefix string) LoggerInterface
	AddToContext(ctx context.Context) context.Context
}

type Logger struct {
	*logrus.Logger
}

func NewRootLogger(ctx context.Context, appName string) (context.Context, *Logger) {
	logger := logrus.New()
	if (terminal.IsTerminal(int(os.Stdout.Fd())) || colossusconfig.DefaultConfig.Colossus.Logging.ForceISaTTY) && !colossusconfig.DefaultConfig.Colossus.Logging.ForceConsoleJSON {
		logger.Formatter = &log_prefixed.TextFormatter{
			ForceColors:     true,
			ForceFormatting: true,
		}
	} else {
		logger.Formatter = &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		}
	}
	instance := &LogInstance{logger: logger.WithField("prefix", appName), prefix: appName}
	l := &Logger{
		Logger: logger,
	}
	subCtx := context.WithValue(ctx, ctxKey, instance)
	if colossusconfig.DefaultConfig.Colossus.Logging.StackDriver {
		l.EnableStackDriverLogging(subCtx)
	}
	if colossusconfig.DefaultConfig.Colossus.Logging.DisableConsole {
		logger.SetOutput(ioutil.Discard)
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
		}
		m.Hooks.Add(h)
		logrus.RegisterExitHandler(h.Wait)
		hasTarget = true

	}
	if colossusconfig.DefaultConfig.Colossus.Logging.StackDriver_.UseGCE {
		instanceId, err := metadata.InstanceName()
		if err != nil {
			m.Logger.Errorf("Error determining instance id: %v", err)
		}
		project, err := metadata.ProjectID()
		if err != nil {
			m.Logger.Errorf("Error determining instance project: %v", err)
			panic(err)
		}
		m.Logger.Infof("Starting StackDriver Logging via GCE on project '%s' with node id `%s`", project, instanceId)
		h, err := sdhook.New(
			sdhook.GoogleComputeCredentials(""),
			sdhook.ProjectID(project),
			sdhook.LogName("colossus"),
			sdhook.Resource("generic_node", map[string]string{
				"project_id": project,
				"node_id":    instanceId,
			}),
		)
		if err != nil {
			m.Logger.Errorf("Error creating stackdriver Logger: %v", err)
			panic(err)
		}
		m.Hooks.Add(h)
		logrus.RegisterExitHandler(h.Wait)
		hasTarget = true

	}
	if colossusconfig.DefaultConfig.Colossus.Logging.StackDriver_.UseApplicationDefaultCredentials || !hasTarget {
		// UseApplicationDefaultCredentials will be the default case
		hostname, err := os.Hostname()
		if err != nil {
			m.Logger.Errorf("Error determining hostname: %v", err)
		}
		data, err := ioutil.ReadFile(colossusconfig.DefaultConfig.Google.Application.Credentials)
		if err != nil {
			panic(err)
		}
		gsa, err := gserviceaccount.FromJSON(data)
		if err != nil {
			panic(err)
		}
		h, err := sdhook.New(
			sdhook.GoogleServiceAccountCredentialsJSON(data),
			sdhook.Resource("generic_node", map[string]string{
				"project_id": gsa.ProjectID,
				"node_id":    hostname,
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
	nextInstance := &LogInstance{logger: m.logger.WithFields(fields), prefix: m.prefix}
	return context.WithValue(ctx, ctxKey, nextInstance)
}

func WithPrefix(ctx context.Context, prefix string) context.Context {
	m := logFromCtx(ctx)
	nextInstance := &LogInstance{logger: m.logger.WithField("prefix", m.prefix+"/"+prefix), prefix: m.prefix + "/" + prefix}
	return context.WithValue(ctx, ctxKey, nextInstance)
}

func SubLoggerWithFields(ctx context.Context, fields map[string]interface{}) LoggerInterface {
	return &subLogger{
		fields: fields,
		prefix: nil,
	}
}

func SubLoggerWithPrefix(ctx context.Context, prefix string) LoggerInterface {
	return &subLogger{
		fields: nil,
		prefix: &prefix,
	}
}

type subLogger struct {
	fields map[string]interface{}
	prefix *string
}

func (s *subLogger) Info(ctx context.Context, msg string) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Info(msg)
}
func (s *subLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Infof(format, args...)
}
func (s *subLogger) Warn(ctx context.Context, msg string) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Warn(msg)
}
func (s *subLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Warnf(format, args...)
}
func (s *subLogger) Err(ctx context.Context, err error, msg string) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.WithError(err).Error(msg)
}
func (s *subLogger) Errf(ctx context.Context, err error, format string, args ...interface{}) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.WithError(err).Errorf(format, args...)
}
func (s *subLogger) Error(ctx context.Context, msg string) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Error(msg)
}
func (s *subLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Errorf(format, args...)
}
func (s *subLogger) Debug(ctx context.Context, msg string) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Debug(msg)
}
func (s *subLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Debugf(format, args...)
}
func (s *subLogger) Trace(ctx context.Context, msg string) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Trace(msg)
}
func (s *subLogger) Tracef(ctx context.Context, format string, args ...interface{}) {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	m.logger.Tracef(format, args...)
}
func (s *subLogger) WithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	nextInstance := &LogInstance{logger: m.logger.WithFields(fields), prefix: m.prefix}
	return context.WithValue(ctx, ctxKey, nextInstance)
}

func (s *subLogger) WithPrefix(ctx context.Context, prefix string) context.Context {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	nextInstance := &LogInstance{logger: m.logger.WithField("prefix", m.prefix+"/"+prefix), prefix: m.prefix + "/" + prefix}
	return context.WithValue(ctx, ctxKey, nextInstance)
}
func (s *subLogger) SubLoggerWithFields(ctx context.Context, fields map[string]interface{}) LoggerInterface {
	fieldCopy := map[string]interface{}{}
	if s.fields != nil {
		for oldKey, old := range s.fields {
			fieldCopy[oldKey] = old
		}
	}
	for newKey, new := range fields {
		fieldCopy[newKey] = new
	}
	return &subLogger{
		fields: fieldCopy,
		prefix: s.prefix,
	}
}

func (s *subLogger) SubLoggerWithPrefix(ctx context.Context, prefix string) LoggerInterface {
	p := prefix
	if s.prefix != nil {
		p = *s.prefix + "/" + prefix
	}
	return &subLogger{
		fields: nil,
		prefix: &p,
	}
}

// AddToContext injects the values from this logginginstance into the context, allowing log calls further down the stack to have the prefix of this in its path
func (s *subLogger) AddToContext(ctx context.Context) context.Context {
	if s.fields != nil {
		ctx = WithFields(ctx, s.fields)
	}
	if s.prefix != nil {
		ctx = WithPrefix(ctx, *s.prefix)
	}
	m := logFromCtx(ctx)
	nextInstance := &LogInstance{logger: m.logger.WithField("prefix", m.prefix), prefix: m.prefix}
	return context.WithValue(ctx, ctxKey, nextInstance)
}

func Info(ctx context.Context, msg string) {
	m := logFromCtx(ctx)
	m.logger.Info(msg)
}
func Infof(ctx context.Context, format string, args ...interface{}) {
	m := logFromCtx(ctx)
	m.logger.Infof(format, args...)
}
func Warn(ctx context.Context, msg string) {
	m := logFromCtx(ctx)
	m.logger.Warn(msg)
}
func Warnf(ctx context.Context, format string, args ...interface{}) {
	m := logFromCtx(ctx)
	m.logger.Warnf(format, args...)
}
func Err(ctx context.Context, err error, msg string) {
	m := logFromCtx(ctx)
	m.logger.WithError(err).Error(msg)
}
func Errf(ctx context.Context, err error, format string, args ...interface{}) {
	m := logFromCtx(ctx)
	m.logger.WithError(err).Errorf(format, args...)
}
func Error(ctx context.Context, msg string) {
	m := logFromCtx(ctx)
	m.logger.Error(msg)
}
func Errorf(ctx context.Context, format string, args ...interface{}) {
	m := logFromCtx(ctx)
	m.logger.Errorf(format, args...)
}
func Debug(ctx context.Context, msg string) {
	m := logFromCtx(ctx)
	m.logger.Debug(msg)
}
func Debugf(ctx context.Context, format string, args ...interface{}) {
	m := logFromCtx(ctx)
	m.logger.Debugf(format, args...)
}
func Trace(ctx context.Context, msg string) {
	m := logFromCtx(ctx)
	m.logger.Trace(msg)
}
func Tracef(ctx context.Context, format string, args ...interface{}) {
	m := logFromCtx(ctx)
	m.logger.Tracef(format, args...)
}
