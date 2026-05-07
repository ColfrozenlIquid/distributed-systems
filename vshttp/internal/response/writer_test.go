package response

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteStatusLineWritesHTTPStatusCodeAsDecimal(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteStatusLine(OK)

	require.NoError(t, err)
	assert.Equal(t, "HTTP/1.1 200 OK\r\n", buf.String())
}
