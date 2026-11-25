package ui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"dosprintgo/pkg/config"
	"dosprintgo/pkg/printer"
	"dosprintgo/pkg/spooler"
)

//go:embed frontend/*
var frontendFS embed.FS

// ServiceController defines operations to query and control the Windows Service
type ServiceController interface {
	GetStatus() (status string, display string, err error)
	Control(action string) (message string, err error)
}

// Server handles the local Web GUI for configuring ports and printers
type Server struct {
	httpServer    *http.Server
	manager       *spooler.SpoolManager
	svcController ServiceController
	logger        *log.Logger
	port          int
}

// NewServer creates a new GUI Web Server
func NewServer(port int, manager *spooler.SpoolManager, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	if port <= 0 {
		port = 8989
	}

	s := &Server{
		manager: manager,
		logger:  logger,
		port:    port,
	}

	mux := http.NewServeMux()

	// Static files
	subFS, err := fs.Sub(frontendFS, "frontend")
	if err == nil {
		mux.Handle("/", http.FileServer(http.FS(subFS)))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Erro ao carregar arquivos da interface", 500)
		})
	}

	// APIs
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/test-print", s.handleTestPrint)
	mux.HandleFunc("/api/service/restart", s.handleRestart)
	mux.HandleFunc("/api/service/status", s.handleServiceStatus)
	mux.HandleFunc("/api/service/control", s.handleServiceControl)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// Start begins listening on localhost
func (s *Server) Start() error {
	s.logger.Printf("Painel de Controle Web GUI iniciado em: http://127.0.0.1:%d", s.port)
	return s.httpServer.ListenAndServe()
}

// Stop shuts down the web server
func (s *Server) Stop() error {
	return s.httpServer.Close()
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		cfg, err := config.GetConfig()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		type response struct {
			*config.Config
			ConfigFile string `json:"config_file"`
		}

		resp := response{
			Config:     cfg,
			ConfigFile: config.ConfigFileUsed(),
		}

		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if r.Method == http.MethodPost {
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, fmt.Sprintf(`{"success":false,"error":"JSON inválido: %v"}`, err), http.StatusBadRequest)
			return
		}

		if err := config.SaveConfig(&newCfg); err != nil {
			http.Error(w, fmt.Sprintf(`{"success":false,"error":"Erro ao salvar via Viper: %v"}`, err), http.StatusInternalServerError)
			return
		}

		s.logger.Printf("Configurações atualizadas via Web GUI e salvas no disco.")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		return
	}

	http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
}

type testPrintRequest struct {
	Port    string `json:"port"`
	RawData string `json:"raw_data"`
}

var funnyTestPhrases = []string{
	"A IMPRESSAO FALHOU!",
	"CRIS E DEMAIS!",
	"AINDA NAO IMPRIMIU...",
	"BANCO DE DADOS APAGADO COM SUCESSO!",
	"ENQUANTO VOCE LE ISSO, O SISTEMA TRAVOU",
	"REINICIE O MODEM 3 VEZES ANTES DE LER",
	"FAVOR NAO ALIMENTAR O ESTAGIARIO",
	"SE ESTA ETIQUETA SAIU, O CAFE ACABOU!",
	"TESTANDO... 1, 2, 3... CAMBIO!",
	"FORMATANDO C:\\ EM 5 SEGUNDOS...",
	"FOI CULPA DO FARINHA",
	"DE LONGE VOCE PARECE FEIO, DE PERTO PARECE QUE ESTA LONGE",
	"TINTA ACABANDO, FAVOR TROCAR O TONER",
	"PARABENS, SUA IMPRESSORA ESTA VIVA!",
}

func wrapText(text string, maxLen int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) <= maxLen {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	lines = append(lines, current)
	return lines
}

func (s *Server) handleTestPrint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req testPrintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Requisição inválida: %v"}`, err), http.StatusBadRequest)
		return
	}

	cfg, err := config.GetConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	var targetMapping *config.MappingConfig
	reqPortUpper := strings.ToUpper(strings.TrimSpace(req.Port))
	for _, m := range cfg.Mappings {
		if strings.ToUpper(strings.TrimSpace(m.Port)) == reqPortUpper {
			targetMapping = &m
			break
		}
	}

	if targetMapping == nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Porta %s não encontrada na configuração atual."}`, req.Port), http.StatusNotFound)
		return
	}

	var phraseUsed string
	if strings.TrimSpace(req.RawData) == "" {
		phraseUsed = funnyTestPhrases[rand.Intn(len(funnyTestPhrases))]
		descLower := strings.ToLower(targetMapping.Description)
		nowStr := time.Now().Format("15:04:05")
		wrapped := wrapText(phraseUsed, 26)

		if strings.Contains(descLower, "argox") || strings.Contains(descLower, "ppla") {
			var pplaLines string
			baseY := 450
			for i, line := range wrapped {
				y := baseY - (i * 50)
				if y < 100 {
					y = 100
				}
				pplaLines += fmt.Sprintf("19110000%04d0030%s\n", y, line)
			}
			req.RawData = fmt.Sprintf("\x02L\nD11\n%s1911000000500020DOSPRINTGO - SPOOLER LPT [%s]\n1E000000003000207891234567890\nE\n", pplaLines, nowStr)
		} else if strings.Contains(descLower, "cupom") || strings.Contains(descLower, "recibo") || strings.Contains(descLower, "escpos") {
			bodyLines := strings.Join(wrapped, "\n  ")
			req.RawData = fmt.Sprintf("\x1b\x40--------------------------------\n    DOSPRINTGO - SPOOLER LPT\n--------------------------------\n  %s\n--------------------------------\nHora: %s\nStatus: Porta LPT Operacional!\n\n\x1d\x56\x42\x00", bodyLines, nowStr)
		} else {
			zplLines := strings.Join(wrapped, `\&`)
			req.RawData = fmt.Sprintf("^XA\n^FO30,25^FB500,3,4,L,0^A0N,28,28^FD%s^FS\n^FO30,115^A0N,20,20^FDDOSPRINTGO - SPOOLER LPT [%s]^FS\n^FO30,145^BY2,2.5,45^BCN,45,Y,N,N^FD7891234567890^FS\n^XZ", zplLines, nowStr)
		}
	}

	data := []byte(req.RawData)
	s.logger.Printf("[%s] Enviando impressão teste de %d bytes para %s (%s)...",
		targetMapping.Port, len(data), targetMapping.Target, targetMapping.Description)

	var sendErr error
	if targetMapping.TargetType == config.TargetTCP {
		sendErr = printer.SendTCP(targetMapping.Target, data, targetMapping.GetTimeoutDuration())
	} else {
		sendErr = printer.SendWindowsSpooler(targetMapping.Target, "DosPrintGo Teste", data)
	}

	if sendErr != nil {
		s.logger.Printf("[%s] Falha no teste de impressão: %v", targetMapping.Port, sendErr)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   sendErr.Error(),
		})
		return
	}

	s.logger.Printf("[%s] Impressão teste enviada com sucesso!", targetMapping.Port)
	respMsg := "Etiqueta teste enviada com sucesso!"
	if phraseUsed != "" {
		respMsg = fmt.Sprintf("Impresso: %s", phraseUsed)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"phrase":  phraseUsed,
		"message": respMsg,
	})
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	if s.manager != nil {
		go func() {
			s.logger.Printf("Recarregando redirecionamento de portas...")
			cfg, err := config.GetConfig()
			if err == nil {
				_ = s.manager.Reload(cfg)
			}
		}()
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Serviço recarregado com as novas portas.",
	})
}

// SetServiceController configures the service controller instance
func (s *Server) SetServiceController(ctrl ServiceController) {
	s.svcController = ctrl
}

func (s *Server) handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.svcController == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "unsupported",
			"display": "Ambiente sem suporte a serviço",
		})
		return
	}

	status, display, err := s.svcController.GetStatus()
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "not_installed",
			"display": "⚪ Não Instalado",
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"display": display,
	})
}

type serviceControlRequest struct {
	Action string `json:"action"`
}

func (s *Server) handleServiceControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	if s.svcController == nil {
		http.Error(w, `{"success":false,"error":"Controlador de serviço não disponível"}`, http.StatusBadRequest)
		return
	}

	var req serviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"JSON inválido: %v"}`, err), http.StatusBadRequest)
		return
	}

	msg, err := s.svcController.Control(req.Action)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": msg,
	})
}

// OpenBrowser opens the specified URL in the default system browser
func OpenBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	_ = exec.Command(cmd, args...).Start()
}
