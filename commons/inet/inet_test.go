package inet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

func TestCidrAddress(t *testing.T) {
	testcases := []struct {
		Ori  string
		Want string
	}{
		{"192.168.0.1/24", "192.168.0.1/24"},
		{"192.168.23.23/23", "192.168.23.23/23"},
		{"123.123.123.256/24", ""}, // 非法的CIDR，不返回任何值
		{"123.123.123.123/32", "123.123.123.123/32"},
	}

	for _, testcase := range testcases {
		tmp, err := NewCidrAddressFromString(testcase.Ori)
		if err != nil {
			assert.Equal(t, testcase.Want, "")
		}
		assert.Equal(t, testcase.Want, tmp.String())
	}
}

func TestCidrAddress2NetlinkAddr(t *testing.T) {
	cidrAddr, err := NewCidrAddressFromString("192.168.1.22/24")
	assert.Equal(t, nil, err)
	netlinkAddr, err := netlink.ParseAddr("192.168.1.22/24")
	assert.Equal(t, nil, err)
	assert.Equal(t, netlinkAddr.String(), cidrAddr.ToNetlinkAddr().String())
}
