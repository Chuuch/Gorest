package logger_test

import (
	"log/slog"
	"testing"

	"github.com/chuuch/gorest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestSetup_JSON(t *testing.T) {
	require.NoError(t, logger.Setup("info", "json", false))
	require.NotNil(t, slog.Default())
}

func TestSetup_TextDebug(t *testing.T) {
	require.NoError(t, logger.Setup("debug", "text", true))
}

func TestSetup_UnknownLevel(t *testing.T) {
	err := logger.Setup("loud", "json", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown log level")
}

func TestSetup_UnknownFormat(t *testing.T) {
	err := logger.Setup("info", "xml", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown log format")
}
