package printer

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestSendTCPSuccess(t *testing.T) {
	// Start a mock TCP printer server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao iniciar mock TCP printer: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	// Sample ZPL label data as sent by Consinco
	sampleZPL := []byte("^XA^FO50,50^A0N,40,40^FDTESTE CONSINCO GO^FS^XZ")

	var receivedData bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)

	// Handle the incoming print job
	go func() {
		defer wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_, _ = io.Copy(&receivedData, conn)
	}()

	// Send data using our SendTCP function
	err = SendTCP(serverAddr, sampleZPL, 2*time.Second)
	if err != nil {
		t.Fatalf("SendTCP retornou erro inesperado: %v", err)
	}

	wg.Wait()

	if !bytes.Equal(receivedData.Bytes(), sampleZPL) {
		t.Errorf("dados recebidos divergem dos enviados.\nEsperado: %s\nRecebido: %s",
			string(sampleZPL), receivedData.String())
	}
}

func TestSendTCPOfflinePrinter(t *testing.T) {
	// Pick an address that is not listening
	offlineAddr := "127.0.0.1:54321"

	err := SendTCP(offlineAddr, []byte("test"), 200*time.Millisecond)
	if err == nil {
		t.Fatalf("SendTCP deveria ter retornado erro para impressora offline, mas retornou nil")
	}
}

func TestSendTCPEmptyData(t *testing.T) {
	// Empty data should succeed immediately without making a connection
	err := SendTCP("127.0.0.1:9100", []byte{}, 1*time.Second)
	if err != nil {
		t.Errorf("SendTCP com dados vazios retornou erro: %v", err)
	}
}
