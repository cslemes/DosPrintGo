package spooler

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"dosprintgo/pkg/config"
)

func TestSpoolManagerLifecycleAndDispatch(t *testing.T) {
	// 1. Start mock TCP printer
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao iniciar mock TCP: %v", err)
	}
	defer listener.Close()

	mockPrinterAddr := listener.Addr().String()

	var receivedData bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_, _ = io.Copy(&receivedData, conn)
	}()

	// 2. Setup temporary spool directory
	tempSpoolDir, err := os.MkdirTemp("", "dosprintgo_spool_*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tempSpoolDir)

	cfg := &config.Config{
		SpoolDir: tempSpoolDir,
		Mappings: []config.MappingConfig{
			{
				Port:         "LPT1",
				TargetType:   config.TargetTCP,
				Target:       mockPrinterAddr,
				Description:  "Mock Zebra Printer",
				TimeoutMs:    2000,
				InactivityMs: 150, // Short quiet period for quick test execution
			},
		},
	}

	// 3. Start SpoolManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(cfg, nil)
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("SpoolManager.Start falhou: %v", err)
	}
	defer manager.Stop()

	spoolFilePath := filepath.Join(tempSpoolDir, "SPOOL1.SPL")

	// Verify that SPOOL1.SPL was initialized
	if _, err := os.Stat(spoolFilePath); os.IsNotExist(err) {
		t.Fatalf("esperava que SPOOL1.SPL existisse, mas não foi encontrado")
	}

	// 4. Simulate Consinco ERP writing label data to SPOOL1.SPL
	expectedZPL := []byte("^XA^FO50,50^FDTESTE_SPOOLER_CONSINCO^FS^XZ")
	err = os.WriteFile(spoolFilePath, expectedZPL, 0666)
	if err != nil {
		t.Fatalf("falha ao escrever dados de teste no spool: %v", err)
	}

	// 5. Wait for the TCP mock server to receive the job
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		// Received successfully!
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout: mock TCP printer não recebeu os dados do spooler a tempo")
	}

	if !bytes.Equal(receivedData.Bytes(), expectedZPL) {
		t.Errorf("dados recebidos divergem dos enviados.\nEsperado: %s\nRecebido: %s",
			string(expectedZPL), receivedData.String())
	}

	// 6. Verify that SPOOL1.SPL has been recreated empty and ready for next prints
	time.Sleep(100 * time.Millisecond)
	info, err := os.Stat(spoolFilePath)
	if err != nil {
		t.Fatalf("SPOOL1.SPL não existe após processamento do job: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("esperava SPOOL1.SPL vazio após envio, mas tamanho é %d bytes", info.Size())
	}
}
