package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gilf4/golang-tftp/pkg/tftp"
)

const (
	ServerPort  = "69"
	BlockSize   = 512
	ReadTimeout = 5 * time.Second
	MaxRetries  = 3
)

var (
	BaseDir string
)

func main() {
	var err error
	// Преобразуем базовую директорию в абсолютный путь
	BaseDir, err = filepath.Abs("./tftp-root")
	if err != nil {
		log.Fatal("Failed to get absolute path:", err)
	}

	if err := os.MkdirAll(BaseDir, 0755); err != nil {
		log.Fatal("Failed to create root directory:", err)
	}

	addr, err := net.ResolveUDPAddr("udp", ":"+ServerPort)
	if err != nil {
		log.Fatal("ResolveUDPAddr failed:", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("ListenUDP failed:", err)
	}
	defer conn.Close()

	log.Printf("TFTP Server listening on :%s (RRQ only)", ServerPort)
	log.Printf("Serving files from: %s", BaseDir)

	buf := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		go handleRequest(buf[:n], clientAddr, conn)
	}
}

func handleRequest(packet []byte, client *net.UDPAddr, serverConn *net.UDPConn) {
	if len(packet) < 2 {
		return
	}

	opcode := binary.BigEndian.Uint16(packet[0:2])

	switch opcode {
	case uint16(tftp.RRQ):
		rrq, err := tftp.ParseRRQ(packet)
		if err != nil {
			sendError(client, tftp.ErrNotDefined, "Invalid RRQ packet", serverConn)
			return
		}
		log.Printf("RRQ from %s: filename=%s mode=%s", client, rrq.Filename, rrq.Mode)
		go serveReadRequest(rrq, client, serverConn)
	default:
		log.Printf("Unsupported opcode %d from %s", opcode, client)
		sendError(client, tftp.ErrNotDefined, "Only RRQ is supported", serverConn)
	}
}

func serveReadRequest(rrq *tftp.ReadRequest, client *net.UDPAddr, serverConn *net.UDPConn) {
	if rrq.Mode != tftp.ModeOctet && rrq.Mode != tftp.ModeNetascii && rrq.Mode != tftp.ModeMail {
		sendError(client, tftp.ErrNotDefined, "Unsupported transfer mode", serverConn)
		return
	}

	// Безопасное построение пути к файлу
	requestedPath := filepath.Join(BaseDir, filepath.Clean(rrq.Filename))
	filename := filepath.Clean(requestedPath)

	// Проверка что файл находится внутри BaseDir
	relPath, err := filepath.Rel(BaseDir, filename)
	if err != nil {
		log.Printf("Path traversal attempt detected: %s", rrq.Filename)
		sendError(client, tftp.ErrAccessViolation, "Access denied", serverConn)
		return
	}

	// Защита от path traversal
	if strings.HasPrefix(relPath, "..") {
		log.Printf("Path traversal attempt detected: %s -> %s", rrq.Filename, relPath)
		sendError(client, tftp.ErrAccessViolation, "Access denied", serverConn)
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File not found: %s", filename)
			sendError(client, tftp.ErrFileNotFound, "File not found", serverConn)
		} else {
			log.Printf("Cannot open file: %s, error: %v", filename, err)
			sendError(client, tftp.ErrAccessViolation, "Cannot open file", serverConn)
		}
		return
	}
	defer file.Close()

	conn, err := net.DialUDP("udp", nil, client)
	if err != nil {
		log.Printf("Failed to dial client %s: %v", client, err)
		return
	}
	defer conn.Close()

	log.Printf("Starting transfer: %s to %s", filename, client)
	block := uint16(1)
	buf := make([]byte, BlockSize)

	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("Error reading file %s: %v", filename, err)
			return
		}

		dataPacket := tftp.PackDATA(block, buf[:n])
		retries := 0

	sendLoop:
		for retries < MaxRetries {
			_, err = conn.Write(dataPacket)
			if err != nil {
				log.Printf("Failed to send block %d: %v", block, err)
				return
			}

			conn.SetReadDeadline(time.Now().Add(ReadTimeout))
			ackBuf := make([]byte, 4)
			n, _, err := conn.ReadFrom(ackBuf)

			if err != nil {
				if e, ok := err.(net.Error); ok && e.Timeout() {
					retries++
					log.Printf("Timeout waiting for ACK %d, retry %d", block, retries)
					continue sendLoop
				}
				log.Printf("Read error: %v", err)
				return
			}

			if n != 4 {
				continue
			}

			parsedAck, parseErr := tftp.ParseACK(ackBuf)
			if parseErr != nil || parsedAck.Block != block {
				continue
			}

			break sendLoop
		}

		if retries >= MaxRetries {
			log.Printf("Max retries exceeded for block %d", block)
			sendError(client, tftp.ErrNotDefined, "Transfer failed: no ACK", serverConn)
			return
		}

		block++
		if n < BlockSize {
			log.Printf("File transfer completed: %s to %s", rrq.Filename, client)
			return
		}
	}
}

func sendError(client *net.UDPAddr, code uint16, msg string, serverConn *net.UDPConn) {
	packet := tftp.PackERROR(code, msg)
	_, _ = serverConn.WriteTo(packet, client)
}
