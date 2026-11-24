package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kardianos/service"

	"dosprintgo/pkg/config"
	"dosprintgo/pkg/spooler"
	"dosprintgo/pkg/ui"
)

// Program implements the kardianos/service.Interface
type Program struct {
	ConfigPath string
	Manager    *spooler.SpoolManager
	UIServer   *ui.Server
	Cancel     context.CancelFunc
	Logger     service.Logger
	SysLogger  *log.Logger
	OpenGUI    bool
}

// Start is called by service runner when starting
func (p *Program) Start(s service.Service) error {
	p.SysLogger.Printf("Iniciando DosPrintGo com Viper...")

	cfg, err := config.InitViper(p.ConfigPath)
	if err != nil {
		p.SysLogger.Printf("Aviso ao carregar Viper: %v", err)
		cfg = config.DefaultConfig()
	}

	p.SysLogger.Printf("Configuração carregada. Arquivo ativo: %s", config.ConfigFileUsed())

	ctx, cancel := context.WithCancel(context.Background())
	p.Cancel = cancel
	p.Manager = spooler.NewManager(cfg, p.SysLogger)

	// Start SpoolManager
	go func() {
		if err := p.Manager.Start(ctx); err != nil {
			p.SysLogger.Fatalf("Falha crítica ao iniciar SpoolManager: %v", err)
		}
	}()

	// Start local Web GUI server
	p.UIServer = ui.NewServer(cfg.HTTPPort, p.Manager, p.SysLogger)
	p.UIServer.SetServiceController(&serviceControllerImpl{
		configPath: p.ConfigPath,
		manager:    p.Manager,
	})
	go func() {
		if err := p.UIServer.Start(); err != nil {
			p.SysLogger.Printf("Aviso: Web GUI encerrada: %v", err)
		}
	}()

	// Open browser if requested
	if p.OpenGUI {
		go func() {
			time.Sleep(500 * time.Millisecond)
			guiURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.HTTPPort)
			p.SysLogger.Printf("Abrindo painel de controle no navegador: %s", guiURL)
			ui.OpenBrowser(guiURL)
		}()
	}

	return nil
}

// Stop is called by service runner when stopping
func (p *Program) Stop(s service.Service) error {
	p.SysLogger.Printf("Parando serviço DosPrintGo...")
	if p.Cancel != nil {
		p.Cancel()
	}
	if p.Manager != nil {
		p.Manager.Stop()
	}
	if p.UIServer != nil {
		_ = p.UIServer.Stop()
	}
	p.SysLogger.Printf("Serviço DosPrintGo parado com sucesso.")
	return nil
}

func getServiceConfig(configPath string) *service.Config {
	svcConfig := &service.Config{
		Name:        "DosPrintGo",
		DisplayName: "DosPrintGo - Redirecionador LPT para Impressoras IP",
		Description: "Emula portas LPT (LPT1 a LPT9) e redireciona impressões para impressoras térmicas de rede IP (Zebra, Argox, ESC/POS) ou Spooler Windows.",
		Arguments:   []string{},
	}

	if configPath != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--config", configPath)
	}

	return svcConfig
}

type serviceControllerImpl struct {
	configPath string
	manager    *spooler.SpoolManager
}

func (s *serviceControllerImpl) GetStatus() (string, string, error) {
	svc, err := newServiceInstanceWithConfig(s.configPath)
	if err != nil {
		return "not_installed", "⚪ Não Instalado", err
	}

	st, err := svc.Status()
	if err != nil {
		return "not_installed", "⚪ Não Instalado", err
	}

	switch st {
	case service.StatusRunning:
		return "running", "🟢 Em Execução (Serviço Ativo)", nil
	case service.StatusStopped:
		return "stopped", "🔴 Parado (Serviço Instalado)", nil
	default:
		return "unknown", "🟡 Status Desconhecido", nil
	}
}

func (s *serviceControllerImpl) Control(action string) (string, error) {
	if action == "restart" && s.manager != nil {
		cfg, err := config.GetConfig()
		if err == nil {
			_ = s.manager.Reload(cfg)
		}
	}

	svc, err := newServiceInstanceWithConfig(s.configPath)
	if err != nil {
		return "", err
	}

	err = service.Control(svc, action)
	if err != nil {
		return "", err
	}

	switch action {
	case "install":
		return "Serviço 'DosPrintGo' instalado com sucesso no Windows!", nil
	case "start":
		return "Serviço iniciado com sucesso!", nil
	case "stop":
		return "Serviço parado com sucesso.", nil
	case "restart":
		return "Serviço reiniciado com sucesso.", nil
	case "uninstall":
		return "Serviço desinstalado do Windows.", nil
	default:
		return "Comando executado com sucesso.", nil
	}
}
