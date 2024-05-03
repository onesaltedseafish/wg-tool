package models

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/onesaltedseafish/wg-tool/commons/inet"
	"gorm.io/gorm"
)

// DhcpClient models for DHCP Client
type DhcpClient struct {
	gorm.Model
	CIDR         inet.CidrAddress `gorm:"column:cidr"`    // 分配的CIDR地址
	Fqdn         string           `gorm:"column:fqdn"`    // 完整可用的域名,这里可以传递 nodename 进来
	Address      inet.IpAddress   `gorm:"column:address"` // 分配的地址
	HardwareAddr net.HardwareAddr `gorm:"column:mac"`     // MAC 地址
	Enable       bool             `gorm:"column:enable"`  // 是否启用
	EnableTime   time.Time        // 启用的时间
}

// DhcpStorage implements for dhcp.Storage
type DhcpStorage struct {
	db   *gorm.DB
	cidr inet.CidrAddress
}

// NewDHCPStorage New a DB implementation for dhcp.Storage
func NewDHCPStorage(db *gorm.DB, cidr inet.CidrAddress) *DhcpStorage {
	return &DhcpStorage{
		db:   db,
		cidr: cidr,
	}
}

// GetAddressWithMAC if storage has a record of hardwareAddr
// then return the related ip address
// else return nil
func (s *DhcpStorage) GetAddressWithMAC(mac net.HardwareAddr) net.IP {
	record := DhcpClient{HardwareAddr: mac}
	s.db.Model(DhcpClient{}).Where(&record).First(&record)
	if record.ID != 0 {
		return net.IP(record.Address)
	}
	return nil
}

// GetOneUnusedAddress finds the first unused record
func (s *DhcpStorage) GetOneUnusedAddress() net.IP {
	var record DhcpClient
	s.db.Model(DhcpClient{}).Where("enable = ? and cidr = ?", false, s.cidr).First(&record)
	if record.ID != 0 {
		return net.IP(record.Address)
	}
	return nil
}

// GetLastAddress finds the last used ip address
func (s *DhcpStorage) GetLastAddress() net.IP {
	record := DhcpClient{
		CIDR: s.cidr,
	}
	s.db.Model(DhcpClient{}).Where(&record).Last(&record)
	if record.ID != 0 {
		return net.IP(record.Address)
	}
	return s.cidr.GetNetwork().IP
}

// SetAddressWithMAC sets record with ip address and MAC address
func (s *DhcpStorage) SetAddressWithMAC(ip net.IP, mac net.HardwareAddr) {
	record := DhcpClient{
		CIDR:    s.cidr,
		Address: inet.IpAddress(ip),
	}
	// 先查找，如果不存在则新增
	tx := s.db.Model(DhcpClient{}).Where(&record).First(&record)
	err := tx.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新增
			s.db.Create(&DhcpClient{
				CIDR:         s.cidr,
				Enable:       true,
				HardwareAddr: mac,
				Address:      inet.IpAddress(ip),
				EnableTime:   time.Now(),
			})
			return
		} else {
			log.Fatalf("err: %v, record: %v", err, record)
		}
	}
	// 修改
	s.db.Model(&record).Updates(DhcpClient{
		HardwareAddr: mac,
		Enable:       true,
		EnableTime:   time.Now(),
	})
}

// ReleaseAddress release the address
func (s *DhcpStorage) ReleaseAddress(ip net.IP) error {
	record := DhcpClient{
		CIDR:    s.cidr,
		Address: inet.IpAddress(ip),
	}
	if err := s.db.Where(&record).First(&record).Error; err != nil {
		return err
	}
	return s.db.Model(&record).Update("enable", false).Error
}

// IsUsed judge the ip address is used or not
func (s *DhcpStorage) IsUsed(ip net.IP) bool {
	record := DhcpClient{
		CIDR:    s.cidr,
		Address: inet.IpAddress(ip),
	}
	if err := s.db.Where(&record).First(&record).Error; err != nil {
		return false
	}
	return record.Enable
}
