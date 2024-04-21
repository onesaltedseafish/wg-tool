package wg

import (
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/onesaltedseafish/wg-tool/commons/inet"
	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// 全局的用于测试的配置
var (
	serverPrivKey, serverPubKey, _ = GenerateWgKeyPairs()
	peerPrivKey, peerPubKey, _     = GenerateWgKeyPairs()
	interfaceName                  = "wg-test-9999"
	keepAliveInterval              = time.Duration(10) * time.Second
	endpoint                       = net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 51820,
	}
	serverAddress, _ = netlink.ParseAddr("192.168.233.11/24")
	subnet, _        = inet.NewSubnetAddressesFromString("192.168.222.1/24, 10.176.22.1/23")

	serverConfig = WgServerConfig{
		InterfaceName: interfaceName,
		PrivateKey:    serverPrivKey.String(),
		ListenPort:    51111,
		Address:       *serverAddress,
	}

	peerConfig = WgPeerConfig{
		InterfaceName: interfaceName,
		PeerConfig: wgtypes.PeerConfig{
			PublicKey:                   peerPubKey,
			Endpoint:                    &endpoint,
			PersistentKeepaliveInterval: &keepAliveInterval,
			AllowedIPs:                  subnet.GetNetworks(),
		},
	}
)

// 只在 linux 环境下进行测试
func TestAddWireguardInterface(t *testing.T) {
	if runtime.GOOS != "linux" {
		return
	}
	err := AddWireguardInterface(serverConfig)
	assert.Equal(t, nil, err)
	// 获取对应的 wg 设备情况
	_, err = netlink.LinkByName(interfaceName)
	assert.Equal(t, nil, err)
	// 删除 wg interface
	err = DeleteWireguardInterface(interfaceName)
	assert.Equal(t, nil, err)
}

// 只在 linux 环境下进行测试
// 主要测试在设置 wireguard 失败的时候
// 时候会将添加的端口删除
func TestAddWireguardInterfaceFailed(t *testing.T) {
	if runtime.GOOS != "linux" {
		return
	}
	errConfig := WgServerConfig{
		InterfaceName: interfaceName,
		PrivateKey:    serverPrivKey.String(),
		ListenPort:    522222,
		Address:       *serverAddress,
	}
	err := AddWireguardInterface(errConfig)
	assert.NotEqual(t, nil, err)
	// 确定这个端口被删除
	_, err = netlink.LinkByName(errConfig.InterfaceName)
	assert.NotEqual(t, nil, err)
}

// 只在 linux 下进行测试
func TestAddWireguardPeers(t *testing.T) {
	if runtime.GOOS != "linux" {
		return
	}
	// 添加 server peer
	err := AddWireguardInterface(serverConfig)
	assert.Equal(t, nil, err)
	// 添加 peer 节点
	err = AddWgPeer(peerConfig)
	assert.Equal(t, nil, err)
	// 检查添加的 wireguard peer 是否正常
	client, err := wgctrl.New()
	assert.Equal(t, nil, err)
	device, err := client.Device(interfaceName)
	assert.Equal(t, nil, err)
	// 开始校验 server peer
	assert.Equal(t, interfaceName, device.Name)
	assert.Equal(t, serverConfig.ListenPort, device.ListenPort)
	assert.Equal(t, serverConfig.PrivateKey, device.PrivateKey.String())
	assert.Equal(t, serverPubKey.String(), device.PublicKey.String())
	assert.Equal(t, 1, len(device.Peers))
	// 开始校验 peer
	p := device.Peers[0]
	assert.Equal(t, peerConfig.PeerConfig.PublicKey.String(), p.PublicKey.String())
	assert.Equal(t, peerConfig.PeerConfig.Endpoint.String(), p.Endpoint.String())
	assert.Equal(t, peerConfig.PeerConfig.PersistentKeepaliveInterval.Seconds(), p.PersistentKeepaliveInterval.Seconds())
	assert.Equal(t, fmt.Sprintf("%s", peerConfig.PeerConfig.AllowedIPs), 
		fmt.Sprintf("%s", p.AllowedIPs))
	// 删除wg接口
	err = DeleteWireguardInterface(interfaceName)
	assert.Equal(t, nil, err)
}
