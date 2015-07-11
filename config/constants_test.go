package config

import (
	"config"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVariablesOverrideConfig(t *testing.T) {
	os.Unsetenv("PORT")
	config.SetupConstants()
	port := config.Constants.Port

	os.Setenv("PORT", "123456as")
	config.SetupConstants()
	port2 := config.Constants.Port

	assert.NotEqual(t, port, port2)
}