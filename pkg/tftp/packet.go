package tftp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

//        2 bytes    string   1 byte     string   1 byte
//        -----------------------------------------------
// RRQ/  | 01/02 |  Filename  |   0  |    Mode    |   0  |
// WRQ    -----------------------------------------------
//
//        2 bytes    2 bytes      n bytes
//        ---------------------------------
// DATA  | 03    |   Block #  |    Data    |
//        ---------------------------------
//
//        2 bytes    2 bytes
//        -------------------
// ACK   | 04    |   Block #  |
//        --------------------
//
//        2 bytes  2 bytes        string    1 byte
//        ----------------------------------------
// ERROR | 05    |  ErrorCode |   ErrMsg   |   0  |
//        ----------------------------------------

// Opcodes
const (
	RRQ   Opcode = 1
	WRQ   Opcode = 2
	DATA  Opcode = 3
	ACK   Opcode = 4
	ERROR Opcode = 5
)

// Error codes
const (
	ErrNotDefined       = 0
	ErrFileNotFound     = 1
	ErrAccessViolation  = 2
	ErrDiskFull         = 3
	ErrIllegalOperation = 4
	ErrUnknownTID       = 5
	ErrFileExists       = 6
	ErrNoSuchUser       = 7
)

// Transfer modes
const (
	ModeNetascii = "netascii"
	ModeOctet    = "octet"
	ModeMail     = "mail"
)

// Errors
var (
	ErrInvalidPacket   = errors.New("invalid packet format")
	ErrPacketTooShort  = errors.New("packet too short")
	ErrInvalidOpcode   = errors.New("invalid opcode")
	ErrMissingNullTerm = errors.New("missing null terminator")
)

type Opcode uint16

func (op Opcode) String() string {
	switch op {
	case RRQ:
		return "RRQ"
	case WRQ:
		return "WRQ"
	case DATA:
		return "DATA"
	case ACK:
		return "ACK"
	case ERROR:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", op)
	}
}

type TFTPPacket interface {
	Opcode() Opcode
	Serialize() []byte
}

// ReadRequest (RRQ)
type ReadRequest struct {
	Filename string
	Mode     string
}

func (rq *ReadRequest) Opcode() Opcode { return RRQ }
func (rq *ReadRequest) Serialize() []byte {
	return packRQ(RRQ, rq.Filename, rq.Mode)
}

func ParseRRQ(data []byte) (*ReadRequest, error) {
	if len(data) < 2 || binary.BigEndian.Uint16(data[0:2]) != uint16(RRQ) {
		return nil, ErrInvalidOpcode
	}
	filename, mode, err := unpackRQ(data)
	if err != nil {
		return nil, err
	}
	return &ReadRequest{Filename: filename, Mode: mode}, nil
}

type WriteRequest struct {
	Filename string
	Mode     string
}

func (wr *WriteRequest) Opcode() Opcode { return WRQ }
func (wr *WriteRequest) Serialize() []byte {
	return packRQ(WRQ, wr.Filename, wr.Mode)
}

func ParseWRQ(data []byte) (*WriteRequest, error) {
	if len(data) < 2 || binary.BigEndian.Uint16(data[0:2]) != uint16(WRQ) {
		return nil, ErrInvalidOpcode
	}
	filename, mode, err := unpackRQ(data)
	if err != nil {
		return nil, err
	}
	return &WriteRequest{Filename: filename, Mode: mode}, nil
}

// DATA
type DataPacket struct {
	Block uint16
	Data  []byte
}

func (p *DataPacket) Opcode() Opcode { return DATA }
func (p *DataPacket) Serialize() []byte {
	return PackDATA(p.Block, p.Data)
}

func PackDATA(block uint16, data []byte) []byte {
	packet := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(packet[0:2], uint16(DATA))
	binary.BigEndian.PutUint16(packet[2:4], block)
	copy(packet[4:], data)
	return packet
}

func ParseDATA(packet []byte) (*DataPacket, error) {
	if len(packet) < 4 {
		return nil, ErrPacketTooShort
	}
	if binary.BigEndian.Uint16(packet[0:2]) != uint16(DATA) {
		return nil, ErrInvalidOpcode
	}
	block := binary.BigEndian.Uint16(packet[2:4])
	return &DataPacket{Block: block, Data: packet[4:]}, nil
}

// ACK
type AckPacket struct {
	Block uint16
}

func (p *AckPacket) Opcode() Opcode { return ACK }
func (p *AckPacket) Serialize() []byte {
	return PackACK(p.Block)
}

func PackACK(block uint16) []byte {
	packet := make([]byte, 4)
	binary.BigEndian.PutUint16(packet[0:2], uint16(ACK))
	binary.BigEndian.PutUint16(packet[2:4], block)
	return packet
}

func ParseACK(packet []byte) (*AckPacket, error) {
	if len(packet) < 4 {
		return nil, ErrPacketTooShort
	}
	if binary.BigEndian.Uint16(packet[0:2]) != uint16(ACK) {
		return nil, ErrInvalidOpcode
	}
	block := binary.BigEndian.Uint16(packet[2:4])
	return &AckPacket{Block: block}, nil
}

// ERROR
type ErrorPacket struct {
	Code    uint16
	Message string
}

func (p *ErrorPacket) Opcode() Opcode { return ERROR }
func (p *ErrorPacket) Serialize() []byte {
	return PackERROR(p.Code, p.Message)
}

func PackERROR(code uint16, msg string) []byte {
	packet := make([]byte, 4+len(msg)+1)
	binary.BigEndian.PutUint16(packet[0:2], uint16(ERROR))
	binary.BigEndian.PutUint16(packet[2:4], code)
	copy(packet[4:], msg)
	packet[len(packet)-1] = 0
	return packet
}

func ParseERROR(packet []byte) (*ErrorPacket, error) {
	if len(packet) < 5 {
		return nil, ErrPacketTooShort
	}
	if binary.BigEndian.Uint16(packet[0:2]) != uint16(ERROR) {
		return nil, ErrInvalidOpcode
	}
	if packet[len(packet)-1] != 0 {
		return nil, ErrMissingNullTerm
	}
	code := binary.BigEndian.Uint16(packet[2:4])
	msg := string(packet[4 : len(packet)-1])
	return &ErrorPacket{Code: code, Message: msg}, nil
}

func isValidMode(mode string) bool {
	switch mode {
	case ModeNetascii, ModeOctet, ModeMail:
		return true
	}
	return false
}

func packRQ(opcode Opcode, filename, mode string) []byte {
	if !isValidMode(mode) {
		panic("unsupported TFTP mode: " + mode)
	}
	packet := make([]byte, 2+len(filename)+1+len(mode)+1)
	binary.BigEndian.PutUint16(packet[0:2], uint16(opcode))
	copy(packet[2:], filename)
	packet[2+len(filename)] = 0
	copy(packet[2+len(filename)+1:], mode)
	packet[2+len(filename)+1+len(mode)] = 0
	return packet
}

func unpackRQ(packet []byte) (filename, mode string, err error) {
	if len(packet) < 4 {
		return "", "", ErrPacketTooShort
	}
	data := packet[2:]

	// Find filename
	filenameEnd := -1
	for i, b := range data {
		if b == 0 {
			filenameEnd = i
			break
		}
	}
	if filenameEnd == -1 {
		return "", "", ErrMissingNullTerm
	}
	filename = string(data[:filenameEnd])

	if filenameEnd+1 >= len(data) {
		return "", "", ErrInvalidPacket
	}

	modeData := data[filenameEnd+1:]
	modeEnd := -1
	for i, b := range modeData {
		if b == 0 {
			modeEnd = i
			break
		}
	}
	if modeEnd == -1 {
		return "", "", ErrMissingNullTerm
	}
	mode = string(modeData[:modeEnd])
	return filename, mode, nil
}
