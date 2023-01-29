package archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaybeEndpointURL(t *testing.T) {
	endpoint, secure := maybeEndpointURL("s3.example.com")
	assert.Equal(t, endpoint, "s3.example.com")
	assert.True(t, secure)

	endpoint, secure = maybeEndpointURL("http://s3.example.com")
	assert.Equal(t, endpoint, "s3.example.com")
	assert.False(t, secure)

	endpoint, secure = maybeEndpointURL("https://s3.example.com")
	assert.Equal(t, endpoint, "s3.example.com")
	assert.True(t, secure)
}