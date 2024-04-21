// Package models 定义数据库中存储的表内容
package models

import (
	"fmt"
	"net"
	"time"

	"github.com/onesaltedseafish/wg-tool/commons/errs"
	"github.com/onesaltedseafish/wg-tool/commons/inet"
	"github.com/onesaltedseafish/wg-tool/commons/wg"
	pb "github.com/onesaltedseafish/wg-tool/protocols"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

type Peer struct {
	gorm.Model
	InterfaceName     string               `gorm:"column:interface_name"`      // 接口名
	PublicIp          string               `gorm:"column:public_ip"`           // 公网 IP
	PeerName          string               `gorm:"column:peer_name"`           // Peer 名称
	PeerAddress       inet.CidrAddress     `gorm:"column:address"`             // Peer Ip 地址
	PeerSubnetAddress inet.SubnetAddresses `gorm:"column:subnet_addresses"`    // 子网地址
	PeerType          uint                 `gorm:"column:type"`                // 节点类型
	IsServer          bool                 `gorm:"column:is_server"`           // 是否作为服务器
	ConnectTo         uint                 `gorm:"column:connect_to"`          // 连接到哪个 peer
	ListenPort        uint16               `gorm:"column:port"`                // 监听端口
	PrivateKey        string               `gorm:"column:private_key"`         // 私钥
	PublicKey         string               `gorm:"column:public_key"`          // 公钥
	KeepAliveInterval int                  `gorm:"column:keep_alive_interval"` // 保持心跳的时间间隔，单位秒
	Remark            string               `gorm:"column:remark"`              // 备注
}

// ToWgServerConfig 将数据库中的record转换为 wg server peer初始化需要的记录
func (p Peer) ToWgServerConfig() wg.WgServerConfig {
	return wg.WgServerConfig{
		InterfaceName: p.InterfaceName,
		ListenPort:    int(p.ListenPort),
		PrivateKey:    p.PrivateKey,
		Address:       p.PeerAddress.ToNetlinkAddr(),
	}
}

// ToWgPeerConfig 将数据库中的 record 转换为 wg peer 初始化需要的记录
func (p Peer) ToWgPeerConfig(db *gorm.DB) (wg.WgPeerConfig, error) {
	var config wg.WgPeerConfig
	var pubKey wgtypes.Key
	var err error
	var endpoint *net.UDPAddr
	var allowIps []net.IPNet // 允许的子网
	if p.ConnectTo == 0 {
		return config, errs.WgNoConnectPeerError
	}
	connectPeer := Peer{}
	connectPeer.ID = p.ConnectTo
	if err = db.First(&connectPeer).Error; err != nil {
		return config, err
	}
	if pubKey, err = wgtypes.ParseKey(connectPeer.PublicKey); err != nil {
		return config, err
	}
	if endpoint, err = connectPeer.GetEndpoint(); err != nil {
		return config, err
	}
	allowIps = append(allowIps, connectPeer.PeerAddress.GetNetwork())
	if p.PeerType == uint(pb.PeerType_SubNet) {
		allowIps = append(allowIps, p.PeerSubnetAddress.GetNetworks()...)
	}
	return wg.WgPeerConfig{
		InterfaceName: p.InterfaceName,
		PeerConfig: wgtypes.PeerConfig{
			PublicKey:                   pubKey,
			Endpoint:                    endpoint,
			PersistentKeepaliveInterval: p.GetKeepAliveInterval(),
			AllowedIPs:                  allowIps,
		},
	}, nil
}

// GetEndpoint 获取连接端点
func (p Peer) GetEndpoint() (addr *net.UDPAddr, err error) {
	if p.ListenPort == 0 {
		err = errs.WgInvalidPortError
		return
	}
	ip := net.ParseIP(p.PublicIp)
	if ip == nil {
		err = fmt.Errorf("%w: %s", errs.WgInvalidAddressError, p.PublicIp)
		return
	}

	addr = &net.UDPAddr{
		IP:   ip,
		Port: int(p.ListenPort),
	}
	return
}

// GetKeepAliveInterval 获取保活时长
func (p Peer) GetKeepAliveInterval() *time.Duration {
	var dur time.Duration
	if p.KeepAliveInterval <= 0 {
		return nil
	}
	dur = time.Second * time.Duration(p.KeepAliveInterval)
	return &dur
}
