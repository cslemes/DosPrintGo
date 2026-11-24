package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var (
	noGUIFlag bool
	runCmd    = &cobra.Command{
		Use:     "run",
		Aliases: []string{"console"},
		Short:   "Executa o DosPrintGo interativamente no console",
		Long:    `Inicia o emulador de portas LPT e o despachador TCP em primeiro plano exibindo logs no terminal.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			stdLog := log.New(os.Stdout, "[DosPrintGo] ", log.LstdFlags|log.Lmsgprefix)

			prg := &Program{
				ConfigPath: configPathFlag,
				SysLogger:  stdLog,
				OpenGUI:    !noGUIFlag,
			}

			svcConfig := getServiceConfig(configPathFlag)
			s, err := service.New(prg, svcConfig)
			if err != nil {
				return fmt.Errorf("erro ao configurar serviço: %w", err)
			}

			fmt.Println("=================================================================")
			fmt.Println("             DosPrintGo - Modo Console Interativo                ")
			fmt.Println("  Painel Web: http://127.0.0.1:8989                             ")
			fmt.Println("  Pressione Ctrl+C para encerrar e restaurar as portas LPT       ")
			fmt.Println("=================================================================")

			return s.Run()
		},
	}
)

func init() {
	runCmd.Flags().BoolVar(&noGUIFlag, "no-gui", false, "Não abre o navegador automaticamente ao iniciar")
	rootCmd.AddCommand(runCmd)
}
