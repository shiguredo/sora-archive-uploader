package archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaybeEndpointURL(t *testing.T) {
	endpoint, secure := maybeEndpointURL("s3.example.com")
	assert.Equal(t, "s3.example.com", endpoint)
	assert.True(t, secure)

	endpoint, secure = maybeEndpointURL("http://s3.example.com")
	assert.Equal(t, "s3.example.com", endpoint)
	assert.False(t, secure)

	endpoint, secure = maybeEndpointURL("https://s3.example.com")
	assert.Equal(t, "s3.example.com", endpoint)
	assert.True(t, secure)

	endpoint, secure = maybeEndpointURL("ldap://s3.example.com")
	assert.Equal(t, "ldap://s3.example.com", endpoint)
	assert.False(t, secure)
}
