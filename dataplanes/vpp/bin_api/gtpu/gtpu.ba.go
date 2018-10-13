// Code generated by GoVPP binapi-generator. DO NOT EDIT.
// source: api/gtpu.api.json

/*
Package gtpu is a generated VPP binary API of the 'gtpu' VPP module.

It is generated from this file:
	gtpu.api.json

It contains these VPP binary API objects:
	6 messages
	3 services
*/
package gtpu

import "git.fd.io/govpp.git/api"
import "github.com/lunixbochs/struc"
import "bytes"

// Reference imports to suppress errors if they are not otherwise used.
var _ = api.RegisterMessage
var _ = struc.Pack
var _ = bytes.NewBuffer

/* Messages */

// GtpuAddDelTunnel represents the VPP binary API message 'gtpu_add_del_tunnel'.
// Generated from 'gtpu.api.json', line 4:
//
//            "gtpu_add_del_tunnel",
//            [
//                "u16",
//                "_vl_msg_id"
//            ],
//            [
//                "u32",
//                "client_index"
//            ],
//            [
//                "u32",
//                "context"
//            ],
//            [
//                "u8",
//                "is_add"
//            ],
//            [
//                "u8",
//                "is_ipv6"
//            ],
//            [
//                "u8",
//                "src_address",
//                16
//            ],
//            [
//                "u8",
//                "dst_address",
//                16
//            ],
//            [
//                "u32",
//                "mcast_sw_if_index"
//            ],
//            [
//                "u32",
//                "encap_vrf_id"
//            ],
//            [
//                "u32",
//                "decap_next_index"
//            ],
//            [
//                "u32",
//                "teid"
//            ],
//            {
//                "crc": "0x7ce9952e"
//            }
//
type GtpuAddDelTunnel struct {
	IsAdd          uint8
	IsIPv6         uint8
	SrcAddress     []byte `struc:"[16]byte"`
	DstAddress     []byte `struc:"[16]byte"`
	McastSwIfIndex uint32
	EncapVrfID     uint32
	DecapNextIndex uint32
	Teid           uint32
}

func (*GtpuAddDelTunnel) GetMessageName() string {
	return "gtpu_add_del_tunnel"
}
func (*GtpuAddDelTunnel) GetCrcString() string {
	return "7ce9952e"
}
func (*GtpuAddDelTunnel) GetMessageType() api.MessageType {
	return api.RequestMessage
}
func NewGtpuAddDelTunnel() api.Message {
	return &GtpuAddDelTunnel{}
}

// GtpuAddDelTunnelReply represents the VPP binary API message 'gtpu_add_del_tunnel_reply'.
// Generated from 'gtpu.api.json', line 56:
//
//            "gtpu_add_del_tunnel_reply",
//            [
//                "u16",
//                "_vl_msg_id"
//            ],
//            [
//                "u32",
//                "context"
//            ],
//            [
//                "i32",
//                "retval"
//            ],
//            [
//                "u32",
//                "sw_if_index"
//            ],
//            {
//                "crc": "0xfda5941f"
//            }
//
type GtpuAddDelTunnelReply struct {
	Retval    int32
	SwIfIndex uint32
}

func (*GtpuAddDelTunnelReply) GetMessageName() string {
	return "gtpu_add_del_tunnel_reply"
}
func (*GtpuAddDelTunnelReply) GetCrcString() string {
	return "fda5941f"
}
func (*GtpuAddDelTunnelReply) GetMessageType() api.MessageType {
	return api.ReplyMessage
}
func NewGtpuAddDelTunnelReply() api.Message {
	return &GtpuAddDelTunnelReply{}
}

// GtpuTunnelDump represents the VPP binary API message 'gtpu_tunnel_dump'.
// Generated from 'gtpu.api.json', line 78:
//
//            "gtpu_tunnel_dump",
//            [
//                "u16",
//                "_vl_msg_id"
//            ],
//            [
//                "u32",
//                "client_index"
//            ],
//            [
//                "u32",
//                "context"
//            ],
//            [
//                "u32",
//                "sw_if_index"
//            ],
//            {
//                "crc": "0x529cb13f"
//            }
//
type GtpuTunnelDump struct {
	SwIfIndex uint32
}

func (*GtpuTunnelDump) GetMessageName() string {
	return "gtpu_tunnel_dump"
}
func (*GtpuTunnelDump) GetCrcString() string {
	return "529cb13f"
}
func (*GtpuTunnelDump) GetMessageType() api.MessageType {
	return api.RequestMessage
}
func NewGtpuTunnelDump() api.Message {
	return &GtpuTunnelDump{}
}

// GtpuTunnelDetails represents the VPP binary API message 'gtpu_tunnel_details'.
// Generated from 'gtpu.api.json', line 100:
//
//            "gtpu_tunnel_details",
//            [
//                "u16",
//                "_vl_msg_id"
//            ],
//            [
//                "u32",
//                "context"
//            ],
//            [
//                "u32",
//                "sw_if_index"
//            ],
//            [
//                "u8",
//                "is_ipv6"
//            ],
//            [
//                "u8",
//                "src_address",
//                16
//            ],
//            [
//                "u8",
//                "dst_address",
//                16
//            ],
//            [
//                "u32",
//                "mcast_sw_if_index"
//            ],
//            [
//                "u32",
//                "encap_vrf_id"
//            ],
//            [
//                "u32",
//                "decap_next_index"
//            ],
//            [
//                "u32",
//                "teid"
//            ],
//            {
//                "crc": "0x68853c3d"
//            }
//
type GtpuTunnelDetails struct {
	SwIfIndex      uint32
	IsIPv6         uint8
	SrcAddress     []byte `struc:"[16]byte"`
	DstAddress     []byte `struc:"[16]byte"`
	McastSwIfIndex uint32
	EncapVrfID     uint32
	DecapNextIndex uint32
	Teid           uint32
}

func (*GtpuTunnelDetails) GetMessageName() string {
	return "gtpu_tunnel_details"
}
func (*GtpuTunnelDetails) GetCrcString() string {
	return "68853c3d"
}
func (*GtpuTunnelDetails) GetMessageType() api.MessageType {
	return api.ReplyMessage
}
func NewGtpuTunnelDetails() api.Message {
	return &GtpuTunnelDetails{}
}

// SwInterfaceSetGtpuBypass represents the VPP binary API message 'sw_interface_set_gtpu_bypass'.
// Generated from 'gtpu.api.json', line 148:
//
//            "sw_interface_set_gtpu_bypass",
//            [
//                "u16",
//                "_vl_msg_id"
//            ],
//            [
//                "u32",
//                "client_index"
//            ],
//            [
//                "u32",
//                "context"
//            ],
//            [
//                "u32",
//                "sw_if_index"
//            ],
//            [
//                "u8",
//                "is_ipv6"
//            ],
//            [
//                "u8",
//                "enable"
//            ],
//            {
//                "crc": "0xe74ca095"
//            }
//
type SwInterfaceSetGtpuBypass struct {
	SwIfIndex uint32
	IsIPv6    uint8
	Enable    uint8
}

func (*SwInterfaceSetGtpuBypass) GetMessageName() string {
	return "sw_interface_set_gtpu_bypass"
}
func (*SwInterfaceSetGtpuBypass) GetCrcString() string {
	return "e74ca095"
}
func (*SwInterfaceSetGtpuBypass) GetMessageType() api.MessageType {
	return api.RequestMessage
}
func NewSwInterfaceSetGtpuBypass() api.Message {
	return &SwInterfaceSetGtpuBypass{}
}

// SwInterfaceSetGtpuBypassReply represents the VPP binary API message 'sw_interface_set_gtpu_bypass_reply'.
// Generated from 'gtpu.api.json', line 178:
//
//            "sw_interface_set_gtpu_bypass_reply",
//            [
//                "u16",
//                "_vl_msg_id"
//            ],
//            [
//                "u32",
//                "context"
//            ],
//            [
//                "i32",
//                "retval"
//            ],
//            {
//                "crc": "0xe8d4e804"
//            }
//
type SwInterfaceSetGtpuBypassReply struct {
	Retval int32
}

func (*SwInterfaceSetGtpuBypassReply) GetMessageName() string {
	return "sw_interface_set_gtpu_bypass_reply"
}
func (*SwInterfaceSetGtpuBypassReply) GetCrcString() string {
	return "e8d4e804"
}
func (*SwInterfaceSetGtpuBypassReply) GetMessageType() api.MessageType {
	return api.ReplyMessage
}
func NewSwInterfaceSetGtpuBypassReply() api.Message {
	return &SwInterfaceSetGtpuBypassReply{}
}

/* Services */

type Services interface {
	DumpGtpuTunnel(*GtpuTunnelDump) (*GtpuTunnelDetails, error)
	GtpuAddDelTunnel(*GtpuAddDelTunnel) (*GtpuAddDelTunnelReply, error)
	SwInterfaceSetGtpuBypass(*SwInterfaceSetGtpuBypass) (*SwInterfaceSetGtpuBypassReply, error)
}

func init() {
	api.RegisterMessage((*GtpuAddDelTunnel)(nil), "gtpu.GtpuAddDelTunnel")
	api.RegisterMessage((*GtpuAddDelTunnelReply)(nil), "gtpu.GtpuAddDelTunnelReply")
	api.RegisterMessage((*GtpuTunnelDump)(nil), "gtpu.GtpuTunnelDump")
	api.RegisterMessage((*GtpuTunnelDetails)(nil), "gtpu.GtpuTunnelDetails")
	api.RegisterMessage((*SwInterfaceSetGtpuBypass)(nil), "gtpu.SwInterfaceSetGtpuBypass")
	api.RegisterMessage((*SwInterfaceSetGtpuBypassReply)(nil), "gtpu.SwInterfaceSetGtpuBypassReply")
}
