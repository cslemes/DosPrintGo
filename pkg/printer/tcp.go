package printer

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// SendTCP sends raw byte data (e.g. ZPL, PPLA, ESC/POS) directly to a network printer IP via TCP (port 9100).
func SendTCP(target string, data []byte, timeout time.Duration) error {
	if len(data) == 0 {
		return nil
	}

	// Ensure port is present; default to standard JetDirect port 9100
	if !strings.Contains(target, ":") {
		target = target + ":9100"
	}

	var conn net.Conn
	var err error

	// Retry loop: 3 attempts with brief pause
	for attempt := 1; attempt <= 3; attempt++ {
		conn, err = net.DialTimeout("tcp", target, timeout)
		if err == nil {
			break
		}
		if attempt < 3 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	if err != nil {
		return fmt.Errorf("não foi possível conectar à impressora térmica em '%s': %w", target, err)
	}
	defer conn.Close()

	_ = conn.SetWriteDeadline(time.Now().Add(timeout))

	totalWritten := 0
	for totalWritten < len(data) {
		n, err := conn.Write(data[totalWritten:])
		if err != nil {
			return fmt.Errorf("erro durante envio de dados para '%s' (enviados %d/%d bytes): %w", target, totalWritten, len(data), err)
		}
		totalWritten += n
	}

	return nil
}
