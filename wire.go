package tftp

import (
    "bytes"
    "encoding/binary"
    "errors"
    "fmt"
    "net"
)

// larger than a typical mtu (1500), and largest DATA packet (516).
// may limit the length of filenames in RRQ/WRQs -- RFC1350 doesn't offer a bound for these.
const MaxPacketSize = 2048

const (
    OpRRQ   uint16 = 1
    OpWRQ          = 2
    OpData         = 3
    OpAck          = 4
    OpError        = 5
)

var Buffers = make(map[string][]byte)
var ServerData = make(map[string][]byte)

// packet is the interface met by all packet structs
type Packet interface {
    // Parse parses a packet from its wire representation
    Parse([]byte) error
    // Serialize serializes a packet to its wire representation
    Serialize() []byte
    // Return uint16 op code of tftp packet
    GetOp() uint16
    // Handle the packet, returning a buffer to send to the receiver
    Handle(*net.UDPAddr) []byte
}


// getBlock retrieves 512 bytes corresponding to the current 
// block number.
func getBlock(buffer string, num uint16) ([]byte) {
    // Final ACK?
    if len(Buffers[buffer]) < (int(num)-1)*512 {
        return nil
    } else if len(Buffers[buffer]) < int(num)*512 {
        return Buffers[buffer][(num-1)*512:]
    } else {
        return Buffers[buffer][(num-1)*512:num*512]
    }
}

// PacketRequest represents a request to read or write a file.
//         2 bytes    string   1 byte     string   1 byte
//        -----------------------------------------------
// RRQ/  | 01/02 |  Filename  |   0  |    Mode    |   0  |
// WRQ    -----------------------------------------------
type PacketRequest struct {
    Op       uint16 // OpRRQ or OpWRQ
    Filename string
    Mode     string
}

func (p *PacketRequest) Handle(addr *net.UDPAddr) ([]byte) {
    // Only octet is implemented per instructions, send error packet if other mode enabled.
    Request.Printf("Client %s OP %d FILE %s MODE %s", addr.String(), p.Op, p.Filename, p.Mode)
    if p.Mode != "octet" {
        Error.Printf("Client %s Unknown Mode %s", addr.String(), p.Mode)
        response := &PacketError{}
        response.Code = 4
        response.Msg = "Unknown transfer mode: " + p.Mode
        return response.Serialize()
    }
    if p.Op == OpRRQ {
        data, ok := ServerData[p.Filename]
        if ok {
            // File exists.  Start sending from block 1.
            Buffers[addr.String()] = data
            response := &PacketData{}
            response.BlockNum = 1
            response.Data = getBlock(addr.String(), response.BlockNum)
            return response.Serialize()
        } else {
            // File doesn't exist.  Error out.
            response := &PacketError{}
            response.Code = 1
            response.Msg = "File not found."
            Error.Printf("Client %s Requested Unknown File: %s", addr.String(), p.Filename)
            return response.Serialize()
        }
    } else {
        Buffers[addr.String()] = append([]byte(p.Filename), make([]byte,1)...)
        // ACK with null block, set for first packet
        response := []byte{0,4,0,0}
        return response
    }
    return nil
}

func (p *PacketRequest) GetOp() (op uint16) {
    return p.Op
}

func (p *PacketRequest) Parse(buf []byte) (err error) {
    if p.Op, buf, err = parseUint16(buf); err != nil {
        return err
    }
    if p.Filename, buf, err = parseString(buf); err != nil {
        return err
    }
    if p.Mode, buf, err = parseString(buf); err != nil {
        return err
    }
    return nil
}

func (p *PacketRequest) Serialize() []byte {
    buf := make([]byte, 2+len(p.Filename)+1+len(p.Mode)+1)
    binary.BigEndian.PutUint16(buf, p.Op)
    copy(buf[2:], p.Filename)
    copy(buf[2+len(p.Filename)+1:], p.Mode)
    return buf
}

// PacketData carries a block of data in a file transmission.
//         2 bytes    2 bytes       n bytes
//        ---------------------------------
// DATA  | 03    |   Block #  |    Data    |
//        ---------------------------------
type PacketData struct {
    BlockNum uint16
    Data     []byte
}

func (p *PacketData) Handle(addr *net.UDPAddr) ([]byte) {
    i := bytes.IndexByte(p.Data, 0)
    Buffers[addr.String()] = append(Buffers[addr.String()], p.Data[:i]...)
    if i < 512 {
        filename, buf, _ := parseString(Buffers[addr.String()])
        ServerData[filename] = buf
    }
    // Acknowledge block transmitted
    response := &PacketAck{}
    response.BlockNum = p.BlockNum
    return response.Serialize()
}

func (p *PacketData) GetOp() (op uint16) {
    return OpData
}

func (p *PacketData) Parse(buf []byte) (err error) {
    buf = buf[2:] // skip over op
    if p.BlockNum, buf, err = parseUint16(buf); err != nil {
        return err
    }
    p.Data = buf
    return nil
}

func (p *PacketData) Serialize() []byte {
    buf := make([]byte, 4+len(p.Data))
    binary.BigEndian.PutUint16(buf, OpData)
    binary.BigEndian.PutUint16(buf[2:], p.BlockNum)
    copy(buf[4:], p.Data)
    return buf
}

// PacketAck acknowledges receipt of a data packet
//        2 bytes    2 bytes
//        --------------------
// ACK   | 04    |   Block #  |
//        --------------------

type PacketAck struct {
    BlockNum uint16
}

func (p *PacketAck) Handle(addr *net.UDPAddr) ([]byte) {
    // Ack received, send the next block.
    response := &PacketData{}
    var sendingBlock = p.BlockNum+1
    response.BlockNum = sendingBlock
    response.Data = getBlock(addr.String(), sendingBlock)
    return response.Serialize()
}

func (p *PacketAck) GetOp() (op uint16) {
    return OpAck
}

func (p *PacketAck) Parse(buf []byte) (err error) {
    buf = buf[2:] // skip over op
    if p.BlockNum, buf, err = parseUint16(buf); err != nil {
        return err
    }
    return nil
}

func (p *PacketAck) Serialize() []byte {
    buf := make([]byte, 4)
    binary.BigEndian.PutUint16(buf, OpAck)
    binary.BigEndian.PutUint16(buf[2:], p.BlockNum)
    return buf
}

// PacketError is sent by a peer who has encountered an error condition
//        2 bytes  2 bytes        string    1 byte
//        ----------------------------------------
// ERROR | 05    |  ErrorCode |   ErrMsg   |   0  |
//        ----------------------------------------
type PacketError struct {
    Code uint16
    Msg  string
}

func (p *PacketError) Handle(addr *net.UDPAddr) ([]byte) {
    Error.Printf("Fatal. Client: %s MSG %s", addr.String(), p.Msg)
    return nil
}

func (p *PacketError) GetOp() (op uint16) {
    return OpError
}

func (p *PacketError) Parse(buf []byte) (err error) {
    buf = buf[2:] // skip over op
    if p.Code, buf, err = parseUint16(buf); err != nil {
        return err
    }
    if p.Msg, buf, err = parseString(buf); err != nil {
        return err
    }
    return nil
}

func (p *PacketError) Serialize() []byte {
    buf := make([]byte, 4+len(p.Msg)+1)
    binary.BigEndian.PutUint16(buf, OpError)
    binary.BigEndian.PutUint16(buf[2:], p.Code)
    copy(buf[4:], p.Msg)
    return buf
}

// parseUint16 reads a big-endian uint16 from the beginning of buf,
// returning it along with a slice pointing at the next position in the buffer.
func parseUint16(buf []byte) (uint16, []byte, error) {
    if len(buf) < 2 {
        Error.Printf("Packet Truncated.  Packet: %s", string(buf))
        return 0, nil, errors.New("packet truncated")
    }
    return binary.BigEndian.Uint16(buf), buf[2:], nil
}

// parseString reads a null-terminated ASCII string from buf,
// returning it along with a slice pointing at the next position in the buffer.
func parseString(buf []byte) (string, []byte, error) {
    i := bytes.IndexByte(buf, 0)
    if i < 0 {
        Error.Printf("Packet Truncated.  Packet: %s", string(buf))
        return "", nil, errors.New("packet truncated")
    }
    return string(buf[:i]), buf[i+1:], nil
}

// ParsePacket parses a packet from its wire representation.
func ParsePacket(buf []byte) (p Packet, err error) {
    var opcode uint16
    if opcode, _, err = parseUint16(buf); err != nil {
        return
    }
    switch opcode {
        case OpRRQ, OpWRQ:
            p = &PacketRequest{}
        case OpData:
            p = &PacketData{}
        case OpAck:
            p = &PacketAck{}
        case OpError:
            p = &PacketError{}
        default:
            err = fmt.Errorf("unexpected opcode %d", opcode)
            return
    }
    err = p.Parse(buf)
    return
}
