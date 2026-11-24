package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Abre o painel de controle Web no navegador padrão",
	Long:  `Inicia o DosPrintGo e abre automaticamente a interface gráfica no navegador em http://127.0.0.1:8989`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stdLog := log.New(os.Stdout, "[DosPrintGo] ", log.LstdFlags|log.Lmsgprefix)

		prg := &Program{
			ConfigPath: configPathFlag,
			SysLogger:  stdLog,
			OpenGUI:    true,
		}

		svcConfig := getServiceConfig(configPathFlag)
		s, err := service.New(prg, svcConfig)
		if err != nil {
			return fmt.Errorf("erro ao configurar serviço: %w", err)
		}

		fmt.Println("=================================================================")
		fmt.Println("             DosPrintGo - Painel Web GUI                         ")
		fmt.Println("  Painel: http://127.0.0.1:8989                                 ")
		fmt.Println("  Pressione Ctrl+C para encerrar                                ")
		fmt.Println("=================================================================")

		return s.Run()
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
