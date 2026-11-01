package ddns

import (
	"testing"

	"gotest.tools/assert"
)

func TestGetPublicIP(t *testing.T) {
	ip, err := getPublicFromIPIFY()

	assert.NilError(t, err)
	assert.Equal(t, ip.Is4(), true)
	assert.Equal(t, ip.IsPrivate(), false)

}
