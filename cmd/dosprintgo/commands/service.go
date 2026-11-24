package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Gerencia o DosPrintGo como Serviço do Windows",
	Long: `Gerencia o ciclo de vida do DosPrintGo no Service Control Manager (SCM) do Windows.
Permite instalar, desinstalar, iniciar, parar e verificar o status do serviço.`,
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Instala o DosPrintGo como Serviço do Windows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return controlService("install")
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Desinstala o serviço do Windows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return controlService("uninstall")
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Inicia o serviço do Windows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return controlService("start")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Para o serviço do Windows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return controlService("stop")
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Reinicia o serviço do Windows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return controlService("restart")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Exibe o status atual do serviço",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newServiceInstance()
		if err != nil {
			return err
		}

		status, err := s.Status()
		if err != nil {
			return fmt.Errorf("não foi possível obter o status do serviço: %w", err)
		}

		var desc string
		switch status {
		case service.StatusRunning:
			desc = "🟢 EM EXECUÇÃO (Running)"
		case service.StatusStopped:
			desc = "🔴 PARADO (Stopped)"
		default:
			desc = "🟡 DESCONHECIDO / NÃO INSTALADO"
		}

		fmt.Printf("Serviço DosPrintGo: %s\n", desc)
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(installCmd)
	serviceCmd.AddCommand(uninstallCmd)
	serviceCmd.AddCommand(startCmd)
	serviceCmd.AddCommand(stopCmd)
	serviceCmd.AddCommand(restartCmd)
	serviceCmd.AddCommand(statusCmd)

	rootCmd.AddCommand(serviceCmd)
}

func newServiceInstance() (service.Service, error) {
	return newServiceInstanceWithConfig(configPathFlag)
}

func newServiceInstanceWithConfig(configPath string) (service.Service, error) {
	stdLog := log.New(os.Stdout, "[DosPrintGo] ", log.LstdFlags)
	prg := &Program{
		ConfigPath: configPath,
		SysLogger:  stdLog,
	}

	svcConfig := getServiceConfig(configPath)
	return service.New(prg, svcConfig)
}

func controlService(action string) error {
	s, err := newServiceInstance()
	if err != nil {
		return fmt.Errorf("erro ao inicializar serviço: %w", err)
	}

	err = service.Control(s, action)
	if err != nil {
		return fmt.Errorf("falha ao executar '%s' no serviço: %w", action, err)
	}

	switch action {
	case "install":
		fmt.Println("✓ Serviço 'DosPrintGo' instalado com sucesso no Windows!")
		fmt.Println("  • Tipo de Inicialização: Automático (inicia no boot antes do login)")
		fmt.Println("  • Para iniciar o serviço agora, execute:")
		fmt.Println("      DosPrintGo.exe service start")
		fmt.Println("  • Painel Web de Configuração:")
		fmt.Println("      http://127.0.0.1:8989")
	case "start":
		fmt.Println("✓ Serviço 'DosPrintGo' iniciado com sucesso!")
		fmt.Println("  • As portas LPT configuradas já estão ativas e interceptando impressões.")
		fmt.Println("  • Painel Web: http://127.0.0.1:8989")
	case "stop":
		fmt.Println("✓ Serviço 'DosPrintGo' parado. Portas LPT restauradas.")
	case "restart":
		fmt.Println("✓ Serviço 'DosPrintGo' reiniciado com sucesso.")
	case "uninstall":
		fmt.Println("✓ Serviço 'DosPrintGo' desinstalado do Windows.")
	default:
		fmt.Printf("Comando '%s' executado com sucesso no serviço DosPrintGo.\n", action)
	}

	return nil
}
