// Package wg 定义了一些 wireguard 能力
package wg

import (
	"errors"

	"github.com/samber/lo"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// GenerateWgKeyPairs 生成 wireguard 的公钥和私钥
// WireGuard使用椭圆曲线加密算法（Elliptic Curve Cryptography，ECC）来生成公钥和私钥
// 公钥和私钥都通过 base64 进行编码
func GenerateWgKeyPairs() (hexPrivateKey wgtypes.Key, hexPublicKey wgtypes.Key, err error) {
	hexPrivateKey, err = wgtypes.GeneratePrivateKey()
	if err != nil {
		return
	}
	hexPublicKey = hexPrivateKey.PublicKey()
	return
}

// AddWireguardInterface 添加一个 wg 接口
func AddWireguardInterface(
	InterfaceName string,
	privateKey string,
	listenPort int,
) (err error) {
	var shutdownFuncs []func() error
	defer func() { // 执行一些后续的清理工作
		if err == nil {
			return
		}
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn())
		}
	}()
	wgAttrs := netlink.NewLinkAttrs()
	wgAttrs.Name = InterfaceName

	wgLink := &netlink.GenericLink{
		LinkAttrs: wgAttrs,
		LinkType:  "wireguard",
	}

	if err = netlink.LinkAdd(wgLink); err != nil {
		return err
	}

	// 如果已经添加成功了网络接口
	// 后续如果设置 wireguard 失败，则删除这个接口
	shutdownFuncs = append(shutdownFuncs, func() error {
		return netlink.LinkDel(wgLink)
	})

	// 配置 wireguard 端口
	config := wgtypes.Config{}
	priKey, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return err
	}
	config.PrivateKey = &priKey
	// 配置端口
	if listenPort > 0 && listenPort <= 65535 {
		config.ListenPort = &listenPort
	}

	// 应用配置
	wgClient, err := wgctrl.New()
	if err != nil {
		return err
	}
	defer wgClient.Close()
	return wgClient.ConfigureDevice(InterfaceName, config)
}

// DeleteWireguardInterface 删除 wireguard 接口
func DeleteWireguardInterface(name string) (err error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	return netlink.LinkDel(&netlink.GenericLink{
		LinkAttrs: attrs,
	})
}

// AddWgPeer 添加节点
func AddWgPeer(wgInterfaceName string, peerconfig wgtypes.PeerConfig) (err error) {
	client, err := wgctrl.New()
	defer client.Close()

	device, err := client.Device(wgInterfaceName)
	if err != nil {
		return
	}
	originPeerConfigs := lo.Map[wgtypes.Peer, wgtypes.PeerConfig](device.Peers, func(item wgtypes.Peer, _ int) wgtypes.PeerConfig {
		return Peer2PeerConfig(item)
	})
	configs := append(originPeerConfigs, peerconfig)

	return client.ConfigureDevice(wgInterfaceName, wgtypes.Config{
		Peers: configs,
	})
}

// Peer2PeerConfig 从获取到的设备的 peer 转为需要设置的 peerConfigs
func Peer2PeerConfig(p wgtypes.Peer) wgtypes.PeerConfig {
	peerConfig := wgtypes.PeerConfig{}
	peerConfig.PublicKey = p.PublicKey
	if p.PresharedKey.String() != "" {
		peerConfig.PresharedKey = &p.PresharedKey
	}
	peerConfig.Endpoint = p.Endpoint
	if p.PersistentKeepaliveInterval != 0 {
		peerConfig.PersistentKeepaliveInterval = &p.PersistentKeepaliveInterval
	}
	peerConfig.AllowedIPs = p.AllowedIPs
	peerConfig.ReplaceAllowedIPs = true
	return peerConfig
}
