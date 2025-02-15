package ddns

import (
	"testing"

	"gotest.tools/assert"
)

func TestResolveDNS(t *testing.T) {
	testDomain := "local.skjulsvik.com"

	res, _ := resolveDNS(testDomain)
	if len(res) < 1 {
		t.Errorf("No res")
		t.FailNow()
	}
	assert.Equal(t, "127.0.0.1", res[0])

}

func TestGetPublicIP(t *testing.T) {
	ip, err := getPublicIPIPIFY()

	assert.NilError(t, err)

	assert.Equal(t, ip.Is4(), true)
	assert.Equal(t, ip.IsPrivate(), false)

}
