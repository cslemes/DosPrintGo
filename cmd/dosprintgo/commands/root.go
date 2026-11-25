package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var (
	configPathFlag string
	rootCmd        = &cobra.Command{
		Use:   "dosprintgo",
		Short: "DosPrintGo - Emulador de Portas LPT para Impressoras Térmicas IP",
		Long: `DosPrintGo emula portas LPT (LPT1 a LPT9) e redireciona os dados de impressão
para impressoras de rede por IP (Zebra ZPL, Argox PPLA/PPLB, ESC/POS) ou Spooler do Windows.

Projetado especificamente para sistemas ERP legados, Clipper e FoxPro.`,
		RunE: runRoot,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPathFlag, "config", "c", "", "Caminho do arquivo config.json")
}

// Execute runs the root Cobra command
func Execute() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) error {
	stdLog := log.New(os.Stdout, "[DosPrintGo] ", log.LstdFlags|log.Lmsgprefix)

	isInteractive := service.Interactive()
	openGUI := isInteractive

	prg := &Program{
		ConfigPath: configPathFlag,
		SysLogger:  stdLog,
		OpenGUI:    openGUI,
	}

	svcConfig := getServiceConfig(configPathFlag)
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("erro ao configurar serviço: %w", err)
	}

	if isInteractive {
		fmt.Println("=================================================================")
		fmt.Println("             DosPrintGo - Executando no Console                 ")
		fmt.Println("  Painel Web: http://127.0.0.1:8989                             ")
		fmt.Println("  Pressione Ctrl+C para encerrar e restaurar as portas LPT       ")
		fmt.Println("=================================================================")
	}

	return s.Run()
}
