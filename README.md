# DosPrintGo

Substituto moderno, leve e resiliente em **Go** para o antigo *DOSPrint*. 

Permite que sistemas legados (como o ERP **Consinco / C5**, Clipper, FoxPro, etc.) emitam etiquetas e impressões para portas paralelas **`LPT1` a `LPT9`** (ou `PRN`), interceptando os dados no nível do subsistema Win32 e despachando-os diretamente via **TCP (porta 9100)** para impressoras térmicas de rede (**Zebra ZPL**, **Argox PPLA/PPLB**, **ESC/POS**) ou filas de Spooler do Windows.

---

## 🚀 Principais Vantagens

1. **Roda nativamente como Serviço do Windows:**
   - Inicia com o Windows antes de qualquer login de usuário.
   - Cria o link `DefineDosDevice` globalmente (visível em todas as sessões RDP / Terminal Services e locais).
2. **Gerenciamento de Configuração com Viper:**
   - Leitura, escrita e recarga dinâmica do arquivo `config.json` sem perda de integridade.
3. **Painel de Controle Visual (Web GUI Integrado):**
   - Acesse pelo navegador em `http://127.0.0.1:8989`.
   - Adicione, edite ou exclua mapeamentos visualmente com um clique.
   - **Botão "⚡ Testar"**: Permite enviar etiquetas de teste (**Zebra ZPL**, **Argox PPLA**, **Cupom ESC/POS**) direto para a impressora para validar a comunicação sem precisar abrir o Consinco!
4. **Envio RAW Direto via TCP (Socket 9100):**
   - **Sem drivers:** Não precisa instalar driver no Windows nem criar portas TCP/IP no Painel de Controle.
   - **Sem filas travadas:** Elimina o problema crônico de "Documento pendente" do Spooler do Windows.
   - **Zero corrupção de comandos ZPL / PPLA:** Os caracteres de controle chegam 100% íntegros à cabeça térmica.

---

## 🖥️ Como Abrir o Painel Gráfico (GUI)

- **Com duplo clique no Windows:** Ao dar duplo clique no `DosPrintGo.exe`, ele abre o console e dispara automaticamente o seu navegador no painel de controle (`http://127.0.0.1:8989`).
- **Pela linha de comando:**
  ```cmd
  DosPrintGo.exe -gui
  ```
- **Quando estiver rodando como Serviço do Windows:**
  O serviço mantém o painel ativo em segundo plano! Basta abrir no navegador da máquina:
  ```
  http://127.0.0.1:8989
  ```

---

## ⚙️ Configuração (`config.toml`)

O arquivo `config.toml` é lido prioritariamente da **pasta atual de execução** do binário:

```toml
# DosPrintGo - Configuração de Redirecionamento de Portas LPT
# Gerenciado via Viper (suporte a recarga dinâmica e Web GUI)

spool_dir = "C:\\ProgramData\\DosPrintGo\\spool"
http_port = 8989

[[mappings]]
port = "LPT1"
target_type = "tcp"
target = "192.168.1.150:9100"
description = "Zebra ZPL - Etiquetas Gôndola Consinco"
timeout_ms = 5000
inactivity_ms = 300

[[mappings]]
port = "LPT2"
target_type = "tcp"
target = "192.168.1.151:9100"
description = "Argox PPLA/PPLB - Etiquetas Recebimento/Expedição"
timeout_ms = 5000
inactivity_ms = 300

[[mappings]]
port = "LPT3"
target_type = "tcp"
target = "192.168.1.152:9100"
description = "Térmica Não Fiscal - Cupom/Romaneio"
timeout_ms = 5000
inactivity_ms = 300
```

### Parâmetros:
- **`port`**: Porta DOS a ser emulada (`LPT1`, `LPT2`, ..., `LPT9`).
- **`target_type`**: `"tcp"` (envio direto por IP) ou `"windows_printer"` (via fila do Windows).
- **`target`**: `"IP:9100"` no modo TCP, ou o nome exato da impressora no Windows no modo spooler.
- **`inactivity_ms`**: Tempo em milissegundos de silêncio após a última gravação para considerar o lote de etiquetas encerrado e disparar o envio (padrão: `300ms`).

---

## 📦 Comandos da CLI (Cobra)

Abra o **Prompt de Comando (CMD)** ou **PowerShell** como **Administrador**:

### 1. Gerenciamento do Serviço do Windows (`service`)
```cmd
# Instalar o serviço no Windows
DosPrintGo.exe service install

# Iniciar o serviço
DosPrintGo.exe service start

# Verificar status do serviço
DosPrintGo.exe service status

# Reiniciar o serviço
DosPrintGo.exe service restart

# Parar o serviço
DosPrintGo.exe service stop

# Desinstalar o serviço
DosPrintGo.exe service uninstall
```

### 2. Painel Gráfico e Modo Console
```cmd
# Abrir o painel de controle Web diretamente no navegador padrão:
DosPrintGo.exe gui

# Executar em modo interativo no console (logs no terminal):
DosPrintGo.exe run

# Executar no console sem abrir o navegador:
DosPrintGo.exe run --no-gui
```

### 3. Flags Globais
- `-c`, `--config`: Especifica o caminho de um arquivo de configuração customizado.
  ```cmd
  DosPrintGo.exe service install --config "C:\MinhaPasta\config.json"
  ```

---

## 🧪 Teste Rápido no Windows

Com o `DosPrintGo` rodando, faça um teste enviando um comando direto pelo CMD:

### Teste Zebra (ZPL na LPT1):
```cmd
echo ^XA^FO50,50^A0N,50,50^FDTESTE ZEBRA CONSINCO^FS^XZ > LPT1:
```

### Teste Argox (PPLA na LPT2):
```cmd
echo ^BL^MD11^M100^M0000^L0000TESTE ARGOX^E > LPT2:
```
