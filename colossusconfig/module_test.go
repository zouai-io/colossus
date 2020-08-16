package colossusconfig

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	c := &Config{}
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "blah")
	os.Setenv("COLOSSUS_LOGGING_STACKDRIVER", "true")
	os.Setenv("COLOSSUS_LOGGING_STACKDRIVER_USELOGGINGAGENT", "true")
	os.Setenv("COLOSSUS_LOGGING_STACKDRIVER_USEGCE", "true")
	err := envconfig.Process("", c)
	assert.NoError(t, err, "error processing config")
	assert.Equal(t, "blah", c.Google.Application.Credentials)
	assert.True(t, c.Colossus.Logging.StackDriver)
	assert.True(t, c.Colossus.Logging.StackDriver_.UseLoggingAgent)
	assert.True(t, c.Colossus.Logging.StackDriver_.UseGCE)
}
