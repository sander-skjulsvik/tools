package ddns

import (
	"testing"

	"gotest.tools/assert"
)

func TestResolveDNS(t *testing.T) {
	testDomain := "local.skjulsvik.com"

	res, err := resolveDNS(testDomain)
	assert.NilError(t, err)
	assert.Equal(t, "127.0.0.1", res.String())

}

func TestGetPublicIP(t *testing.T) {
	ip, err := getPublicFromIPIFY()

	assert.NilError(t, err)
	assert.Equal(t, ip.Is4(), true)
	assert.Equal(t, ip.IsPrivate(), false)

}
