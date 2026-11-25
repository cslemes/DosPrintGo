package spooler

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dosprintgo/pkg/config"
	"dosprintgo/pkg/dosdevice"
	"dosprintgo/pkg/printer"
)

// PortWatcher manages monitoring and printing for a specific LPT port
type PortWatcher struct {
	mapping   config.MappingConfig
	spoolFile string
	logger    *log.Logger
	stopCh    chan struct{}
}

// SpoolManager coordinates all configured port watchers
type SpoolManager struct {
	cfg      *config.Config
	logger   *log.Logger
	watchers []*PortWatcher
	wg       sync.WaitGroup
	mu       sync.Mutex
	ctx      context.Context
}

// NewManager creates a new SpoolManager
func NewManager(cfg *config.Config, logger *log.Logger) *SpoolManager {
	if logger == nil {
		logger = log.New(os.Stdout, "[DosPrintGo] ", log.LstdFlags)
	}
	return &SpoolManager{
		cfg:    cfg,
		logger: logger,
	}
}

// Start initializes the spool directory, redirects all LPT ports, and starts monitoring routines
func (m *SpoolManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx = ctx

	return m.startWatchersLocked()
}

func (m *SpoolManager) startWatchersLocked() error {
	if err := os.MkdirAll(m.cfg.SpoolDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar pasta de spool '%s': %w", m.cfg.SpoolDir, err)
	}

	for _, mapping := range m.cfg.Mappings {
		portUpper := strings.ToUpper(strings.TrimSpace(mapping.Port))
		portNum := strings.TrimPrefix(portUpper, "LPT")
		if portNum == portUpper {
			portNum = "1"
		}
		spoolFileName := fmt.Sprintf("SPOOL%s.SPL", portNum)
		spoolFilePath, err := filepath.Abs(filepath.Join(m.cfg.SpoolDir, spoolFileName))
		if err != nil {
			spoolFilePath = filepath.Join(m.cfg.SpoolDir, spoolFileName)
		}

		// Ensure the file exists
		if _, err := os.Stat(spoolFilePath); os.IsNotExist(err) {
			_ = os.WriteFile(spoolFilePath, []byte{}, 0666)
		}

		// Redirect DOS device
		if err := dosdevice.RedirectLPT(portUpper, spoolFilePath); err != nil {
			m.logger.Printf("ERRO: Falha ao redirecionar porta %s para '%s': %v", portUpper, spoolFilePath, err)
			return err
		}

		m.logger.Printf("Porta %s redirecionada com sucesso para '%s' -> %s (%s)",
			portUpper, spoolFilePath, mapping.Target, mapping.Description)

		pw := &PortWatcher{
			mapping:   mapping,
			spoolFile: spoolFilePath,
			logger:    m.logger,
			stopCh:    make(chan struct{}),
		}
		m.watchers = append(m.watchers, pw)

		m.wg.Add(1)
		go func(watcher *PortWatcher) {
			defer m.wg.Done()
			watcher.run(m.ctx)
		}(pw)
	}

	return nil
}

// Reload stops current watchers and starts fresh ones with updated config
func (m *SpoolManager) Reload(newCfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Printf("Recarregando configuração do SpoolManager...")
	m.stopWatchersLocked()

	m.cfg = newCfg
	return m.startWatchersLocked()
}

// Stop removes all LPT redirections and stops the watchers
func (m *SpoolManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopWatchersLocked()
}

func (m *SpoolManager) stopWatchersLocked() {
	m.logger.Printf("Encerrando monitoramento e restaurando portas LPT...")

	for _, w := range m.watchers {
		close(w.stopCh)
		portUpper := strings.ToUpper(strings.TrimSpace(w.mapping.Port))
		if err := dosdevice.RemoveLPT(portUpper, w.spoolFile); err != nil {
			m.logger.Printf("Aviso ao remover %s: %v", portUpper, err)
		} else {
			m.logger.Printf("Porta %s restaurada.", portUpper)
		}
	}

	m.watchers = nil
	m.wg.Wait()
	m.logger.Printf("Monitores de spool finalizados.")
}

func (pw *PortWatcher) run(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastSize int64 = 0
	var lastChange time.Time
	quietDuration := pw.mapping.GetInactivityDuration()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pw.stopCh:
			return
		case now := <-ticker.C:
			info, err := os.Stat(pw.spoolFile)
			if err != nil {
				continue
			}

			curSize := info.Size()
			if curSize == 0 {
				lastSize = 0
				continue
			}

			if curSize != lastSize {
				// File is currently growing or changed
				lastSize = curSize
				lastChange = now
				continue
			}

			// Size has not changed. Check if quiet period has elapsed
			if now.Sub(lastChange) >= quietDuration {
				// Time to dispatch!
				pw.processJob(curSize)
				lastSize = 0
			}
		}
	}
}

func (pw *PortWatcher) processJob(expectedSize int64) {
	port := strings.ToUpper(pw.mapping.Port)
	jobName := fmt.Sprintf("job_%d_%s.bin", time.Now().UnixNano(), port)
	jobPath := filepath.Join(filepath.Dir(pw.spoolFile), jobName)

	// Try atomic rename. If the application (ERP/DOS) still has the file open with write lock,
	// rename will fail on Windows and we will retry on the next tick.
	err := os.Rename(pw.spoolFile, jobPath)
	if err != nil {
		// File likely still locked by sender
		return
	}

	// Immediately recreate empty spool file so the LPT device is ready for new prints
	_ = os.WriteFile(pw.spoolFile, []byte{}, 0666)

	// Read captured print job data
	data, err := os.ReadFile(jobPath)
	if err != nil {
		pw.logger.Printf("[%s] Erro ao ler arquivo de impressão temporário '%s': %v", port, jobPath, err)
		_ = os.Remove(jobPath)
		return
	}
	defer os.Remove(jobPath)

	if len(data) == 0 {
		return
	}

	pw.logger.Printf("[%s] Capturado trabalho de impressão com %d bytes. Despachando para %s...",
		port, len(data), pw.mapping.Target)

	// Dispatch print job in background goroutine to avoid blocking the watcher
	go func(targetMapping config.MappingConfig, jobData []byte) {
		start := time.Now()
		var sendErr error

		switch targetMapping.TargetType {
		case config.TargetTCP:
			sendErr = printer.SendTCP(targetMapping.Target, jobData, targetMapping.GetTimeoutDuration())
		case config.TargetSpooler:
			docName := fmt.Sprintf("DosPrintGo - %s", port)
			sendErr = printer.SendWindowsSpooler(targetMapping.Target, docName, jobData)
		default:
			sendErr = fmt.Errorf("tipo de destino não suportado: '%s'", targetMapping.TargetType)
		}

		if sendErr != nil {
			pw.logger.Printf("[%s] ERRO AO IMPRIMIR em '%s' (%s): %v",
				port, targetMapping.Target, targetMapping.Description, sendErr)
		} else {
			pw.logger.Printf("[%s] SUCESSO: %d bytes enviados com êxito para '%s' (%s) em %v.",
				port, len(jobData), targetMapping.Target, targetMapping.Description, time.Since(start))
		}
	}(pw.mapping, data)
}
