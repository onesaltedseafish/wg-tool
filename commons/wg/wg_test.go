package wg

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

// 只在 linux 环境下进行测试
func TestAddWireguardInterface(t *testing.T) {
	if runtime.GOOS != "linux" {
		return
	}
	var wgInterfaceName = "wgtest01"
	priKey, _, err := GenerateWgKeyPairs()
	assert.Equal(t, nil, err)
	err = AddWireguardInterface(wgInterfaceName, priKey.String(), 51111)
	assert.Equal(t, nil, err)
	// 获取对应的 wg 设备情况
	_, err = netlink.LinkByName(wgInterfaceName)
	assert.Equal(t, nil, err)
	// 删除 wg interface
	err = DeleteWireguardInterface(wgInterfaceName)
	assert.Equal(t, nil, err)
}

// 只在 linux 环境下进行测试
// 主要测试在设置 wireguard 失败的时候
// 时候会讲添加的端口删除
func TestAddWireguardInterfaceFailed(t *testing.T) {
	if runtime.GOOS != "linux" {
		return
	}
	wgName := "wg9999"
	err := AddWireguardInterface(wgName, "1234", 12345)
	assert.NotEqual(t, nil, err)
	// 确定这个端口被删除
	_, err = netlink.LinkByName(wgName)
	assert.NotEqual(t, nil, err)
}
