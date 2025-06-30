package tftp

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

const (
	RRQ   = 1
	WRQ   = 2
	DATA  = 3
	ACK   = 4
	ERROR = 5
)

//  2 bytes     string    1 byte     string   1 byte
//  ------------------------------------------------
// | Opcode |  Filename  |   0  |    Mode    |   0  |
//  ------------------------------------------------
// 					RRQ/WRQ packet

type pRRQ []byte
type pWRQ []byte

func unpackRQ(packet []byte) (filename, mode string, err error) {

}

func packRQ(opcode uint16, filename, mode string) []byte {

}

//  2 bytes     2 bytes      n bytes
//  ----------------------------------
// | Opcode |   Block #  |   Data     |
//  ----------------------------------
// 	          DATA packet

type pDATA []byte

//  2 bytes     2 bytes
//  ---------------------
// | Opcode |   Block #  |
//  ---------------------
// 		 ACK packet

type pACK []byte

//  2 bytes     2 bytes      string    1 byte
//  -----------------------------------------
// | Opcode |  ErrorCode |   ErrMsg   |   0  |
//  -----------------------------------------
// 				ERROR packet

type pERROR []byte
