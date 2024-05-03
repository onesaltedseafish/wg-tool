package models_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/onesaltedseafish/go-utils/log"
	gormlog "github.com/onesaltedseafish/go-utils/log/gorm"
	"github.com/onesaltedseafish/go-utils/simulate/dhcp"
	"github.com/onesaltedseafish/wg-tool/commons/inet"
	"github.com/onesaltedseafish/wg-tool/commons/wg"
	"github.com/onesaltedseafish/wg-tool/models"
	pb "github.com/onesaltedseafish/wg-tool/protocols"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	testSqlitePath = "test.sqlite"
)

var (
	testDb     *gorm.DB
	err        error
	logOpt     = log.CommonLogOpt.WithDirectory("logs").WithLogLevel(zapcore.DebugLevel).WithTraceIDEnable(false).WithConsoleLog(false)
	logger     = log.GetLogger("models-test", &logOpt)
	gormLogger = gormlog.NewLogger("gorm", &logOpt)
	ctx        = context.Background()
)

func init() {
	testDb, err = gorm.Open(
		sqlite.Open(testSqlitePath), &gorm.Config{
			Logger: gormLogger,
		},
	)
	if err != nil {
		logger.Fatal(ctx, "start sqlite error", zap.Error(err))
	}
	// migrate
	if err = testDb.AutoMigrate(
		models.Peer{},
		models.DhcpClient{},
	); err != nil {
		logger.Fatal(ctx, "migrate models error", zap.Error(err))
	}
}

func TestPeerRecords(t *testing.T) {
	relayAddress, _ := inet.NewCidrAddressFromString("192.168.222.1/24")
	relayPrivKey, relayPubKey, _ := wg.GenerateWgKeyPairs()
	recordRelayNode := models.Peer{
		InterfaceName: "wg0",
		PeerName:      "relay_node1",
		PeerAddress:   relayAddress,
		PeerType:      uint(pb.PeerType_P2P),
		IsServer:      true,
		ListenPort:    51820,
		PrivateKey:    relayPrivKey.String(),
		PublicKey:     relayPubKey.String(),
		Remark:        "Relay 节点",
		PublicIp:      "1.2.3.4",
	}
	// 创建主节点
	r1 := testDb.Create(&recordRelayNode)
	assert.Equal(t, nil, r1.Error)

	peerAddress, _ := inet.NewCidrAddressFromString("192.168.222.10/32")
	peerSubnetAddrs, _ := inet.NewSubnetAddressesFromString("10.192.10.1/23, 10.176.22.1/23")
	peerPrivKey, peerPubKey, _ := wg.GenerateWgKeyPairs()
	recordPeerNode := models.Peer{
		InterfaceName:     "wg0",
		PeerName:          "nat_node1",
		PeerAddress:       peerAddress,
		PeerType:          uint(pb.PeerType_SubNet),
		PeerSubnetAddress: peerSubnetAddrs,
		IsServer:          false,
		ConnectTo:         recordRelayNode.ID,
		PrivateKey:        peerPrivKey.String(),
		PublicKey:         peerPubKey.String(),
		KeepAliveInterval: 5,
		Remark:            "NAT 节点",
	}
	r2 := testDb.Create(&recordPeerNode)
	assert.Equal(t, nil, r2.Error)

	// 开始校验查询
	testcases := []models.Peer{recordRelayNode, recordPeerNode}
	for _, v := range testcases {
		// 执行查询操作
		tmpRecord := models.Peer{}
		tmpRecord.ID = v.ID
		r := testDb.First(&tmpRecord)
		assert.Equal(t, nil, r.Error)
		assert.Equal(t, v.PrivateKey, tmpRecord.PrivateKey)
		assert.Equal(t, v.PublicKey, tmpRecord.PublicKey)
		assert.Equal(t, v.CreatedAt.UTC().Unix(), tmpRecord.CreatedAt.UTC().Unix())
		assert.Equal(t, v.PeerAddress, tmpRecord.PeerAddress)
		assert.Equal(t, v.PeerSubnetAddress, tmpRecord.PeerSubnetAddress)
		assert.Equal(t, v.PeerType, tmpRecord.PeerType)
	}

	// 删除
	for _, v := range testcases {
		assert.Equal(t, nil, testDb.Delete(&v).Error)
	}
}

func TestDhcpClient(t *testing.T) {
	cidr, err := inet.NewCidrAddressFromString("192.168.0.1/24")
	address := inet.ParseIpAddressFromString("192.168.0.2")
	assert.Equal(t, nil, err)
	macAddr, err := net.ParseMAC("00:16:3e:03:57:05")
	assert.Equal(t, nil, err)
	record := models.DhcpClient{
		CIDR:         cidr,
		Fqdn:         "test.node",
		Address:      *address,
		Enable:       true,
		EnableTime:   time.Now(),
		HardwareAddr: macAddr,
	}
	// 创建
	assert.Equal(t, nil, testDb.Create(&record).Error)

	// 开始校验查询
	testcases := []models.DhcpClient{record}
	for _, v := range testcases {
		var data models.DhcpClient
		data.ID = v.ID
		assert.Equal(t, nil, testDb.First(&data).Error)
		// 校验具体的数据
		assert.Equal(t, v.Address, data.Address)
		assert.Equal(t, v.CIDR, data.CIDR)
		assert.Equal(t, v.Enable, data.Enable)
		assert.Equal(t, v.EnableTime.Unix(), data.EnableTime.Unix())
		assert.Equal(t, v.HardwareAddr, data.HardwareAddr)
	}
}

func TestDHCPDBImpl(t *testing.T) {
	var ip net.IP
	var (
		cidr, err = inet.NewCidrAddressFromString("127.0.0.0/30")
		mac1, _   = net.ParseMAC("00:16:3e:03:57:45")
		mac2, _   = net.ParseMAC("02:42:be:7f:b3:58")
		mac3, _   = net.ParseMAC("02:42:fe:21:ad:e3")
		mac4, _   = net.ParseMAC("9a:7e:a6:2f:f0:d0")
		mac5, _   = net.ParseMAC("00:16:3e:03:57:46")
	)

	assert.Equal(t, nil, err)

	dhcpClient := dhcp.New(cidr.GetNetwork(), models.NewDHCPStorage(testDb, cidr))

	// 删除表中所有存在的记录
	err = testDb.Where("1 = 1").Delete(&models.DhcpClient{}).Error
	assert.Equal(t, nil, err)

	// allocate from start
	ip, err = dhcpClient.AllocateAddress(mac1)
	assert.Equal(t, nil, err)
	assert.Equal(t, "127.0.0.1", ip.String())

	// release and allocate the same
	err = dhcpClient.ReleaseAddress(net.ParseIP("127.0.0.1"))
	assert.Equal(t, nil, err)
	ip, err = dhcpClient.AllocateAddress(mac1)
	assert.Equal(t, nil, err)
	assert.Equal(t, "127.0.0.1", ip.String())
	// allocate more 2 address
	_, err = dhcpClient.AllocateAddress(mac2)
	assert.Equal(t, nil, err)
	_, err = dhcpClient.AllocateAddress(mac3)
	assert.Equal(t, nil, err)
	// release one and alloate to mac4
	err = dhcpClient.ReleaseAddress(net.ParseIP("127.0.0.2"))
	assert.Equal(t, nil, err)
	ip, err = dhcpClient.AllocateAddress(mac4)
	assert.Equal(t, nil, err)
	assert.Equal(t, "127.0.0.2", ip.String())
	// can't allocate address for mac5
	_, err = dhcpClient.AllocateAddress(mac5)
	assert.NotEqual(t, nil, err)
}
