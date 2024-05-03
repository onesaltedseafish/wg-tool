// Package inet 定义网络库相关的内容
package inet

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/samber/lo"
	"github.com/vishvananda/netlink"
)

var _ sql.Scanner = (*CidrAddress)(nil)
var _ driver.Valuer = (*CidrAddress)(nil)
var _ sql.Scanner = (*SubnetAddresses)(nil)
var _ driver.Valuer = (*SubnetAddresses)(nil)

// IpAddress IP 地址
type IpAddress net.IP

// ParseIpAddressFromString 从字符中初始化 IP
func ParseIpAddressFromString(s string) *IpAddress {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	return (*IpAddress)(&ip)
}

// String to string
func (ip IpAddress) String() string {
	return (net.IP)(ip).String()
}

// Scan 实现 sql.Scanner 接口，Scan 将 value 扫描至
func (addr *IpAddress) Scan(value any) (err error) {
	if value == nil {
		*addr = nil
		return nil
	}
	switch v := value.(type) {
	case string:
		*addr = (IpAddress)(net.ParseIP(v))
	default:
		return fmt.Errorf("无法将值转换为IpAddress类型")
	}

	return nil
}

// Value 实现 driver.Valuer 接口
func (ip IpAddress) Value() (driver.Value, error) {
	if ip == nil {
		return nil, nil
	}

	return ip.String(), nil
}

func (ip IpAddress) GormDataType() string {
	return "string"
}

// CidrAddress CIDR地址
type CidrAddress struct {
	address net.IP    // 主机地址
	network net.IPNet // 子网地址
}

// String 返回CIDR地址表示
func (addr CidrAddress) String() string {
	if addr.address == nil {
		return ""
	}
	cnt, _ := addr.network.Mask.Size()
	return fmt.Sprintf("%s/%d", addr.address.String(), cnt)
}

// ToNetlinkAddr 转换成 netlink 中的地址
func (addr CidrAddress) ToNetlinkAddr() netlink.Addr {
	newIpnet := addr.network
	newIpnet.IP = addr.address
	return netlink.Addr{
		IPNet: &newIpnet,
	}
}

// GetNetwork get CIDR network
func (addr CidrAddress) GetNetwork() net.IPNet {
	return addr.network
}

// Scan 实现 sql.Scanner 接口，Scan 将 value 扫描至
func (addr *CidrAddress) Scan(value any) (err error) {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("Fail to unmarshal cidr address value: %v", value)
	}
	a, n, e := net.ParseCIDR(v)
	if e != nil {
		return e
	}
	addr.address = a
	addr.network = *n
	return
}

// Value 实现 driver.Valuer 接口
func (addr CidrAddress) Value() (driver.Value, error) {
	return addr.String(), nil
}

// NewCidrAddressFromString 从字符串中初始化CIDR地址
func NewCidrAddressFromString(addr string) (CidrAddress, error) {
	a, n, e := net.ParseCIDR(addr)
	if e != nil {
		return CidrAddress{}, e
	}
	return CidrAddress{
		address: a,
		network: *n,
	}, nil
}

// SubnetAddresses 定义子网地址
type SubnetAddresses struct {
	address []CidrAddress
}

// NewSubnetAddressesFromString 从字符串中初始化多个子网地址
func NewSubnetAddressesFromString(addrs string) (SubnetAddresses, error) {
	var err error
	var result SubnetAddresses
	// 将对应的子网地址以,分隔进行存放
	nets := strings.Split(addrs, ",")
	// 去除两边的空格
	nets = lo.Map(nets, func(item string, _ int) string {
		return strings.Trim(item, " ")
	})
	for _, addr := range nets {
		tmpAddr, tmpErr := NewCidrAddressFromString(addr)
		if tmpErr != nil {
			err = errors.Join(err, tmpErr)
		} else {
			result.address = append(result.address, tmpAddr)
		}
	}
	return result, err
}

// GetNetworks get all CIDR network
func (subnet SubnetAddresses) GetNetworks() []net.IPNet {
	return lo.Map(subnet.address, func(item CidrAddress, _ int) net.IPNet {
		return item.GetNetwork()
	})
}

func (subnet SubnetAddresses) String() string {
	return strings.Join(lo.Map(subnet.address, func(item CidrAddress, _ int) string {
		return item.String()
	}), ",")
}

// Scan 实现 sql.Scanner 接口，Scan 将 value 扫描至
func (subnet *SubnetAddresses) Scan(value any) (err error) {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("Fail to unmarshal subnets value: %v", value)
	}
	// 将对应的子网地址以,分隔进行存放
	nets := strings.Split(v, ",")
	// 去除两边的空格
	nets = lo.Map(nets, func(item string, _ int) string {
		return strings.Trim(item, " ")
	})
	for _, addr := range nets {
		tmpAddr, tmpErr := NewCidrAddressFromString(addr)
		if tmpErr != nil {
			err = errors.Join(err, tmpErr)
		} else {
			subnet.address = append(subnet.address, tmpAddr)
		}
	}
	return err
}

// Value 实现 driver.Valuer 接口
func (subnet SubnetAddresses) Value() (driver.Value, error) {
	if len(subnet.address) == 0 {
		return nil, nil
	}
	addrs := lo.Map(subnet.address, func(item CidrAddress, _ int) string {
		return item.String()
	})
	return strings.Join(addrs, ","), nil
}

func (subnet SubnetAddresses) GormDataType() string {
	return "string"
}
