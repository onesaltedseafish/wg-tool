// Package errs 自定义错误
package errs

import "errors"

var (
	WgNoConnectPeerError = errors.New("Wg没有连接的节点")
	WgInvalidPortError = errors.New("Wg端口非法")
	WgInvalidAddressError = errors.New("Wg地址非法")
)
