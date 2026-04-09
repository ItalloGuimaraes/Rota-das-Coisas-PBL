package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Variáveis de estado local da UI e caches de rede.
var (
	telemetria       = make(map[string]string)
	atuadores        = make(map[string]bool)
	indicesSensores  []string
	indicesAtuadores []string
	sensorAlvo       = "TODOS"
	clientID         string
	mutex            sync.Mutex
	telaAtual        = "MENU"
	erroMsg          = ""
	conectado        = false
)

// main: Orquestra o ciclo de vida da aplicação cliente.
// Inclui lógica de resolução dinâmica de endereçamento para deploy via Docker Hub.
func main() {
	clientID = os.Getenv("CLIENT_ID")
	if clientID == "" {
		clientID = "GESTOR_DC_01"
	}

	// Resolução de endereço via variáveis de ambiente (Essential for Docker)
	endereco := os.Getenv("BROKER_ADDR")
	if endereco == "" {
		endereco = "127.0.0.1"
	}
	if !strings.Contains(endereco, ":") {
		endereco = endereco + ":9000"
	}

	for {
		conn, err := net.Dial("tcp", endereco)
		if err != nil {
			mutex.Lock()
			conectado = false
			erroMsg = fmt.Sprintf("BROKER [%s] OFFLINE. Tentando reconectar...", endereco)
			mutex.Unlock()
			renderizar()
			time.Sleep(3 * time.Second) // Estratégia de recuo para evitar saturação de CPU
			continue
		}

		mutex.Lock()
		conectado = true
		erroMsg = ""
		atuadores = make(map[string]bool)
		mutex.Unlock()

		// Identificação perante o Broker seguindo o protocolo proprietário
		fmt.Fprintf(conn, "IDENTIFY|CLIENT|%s\n", clientID)
		renderizar()

		connFechada := make(chan bool)
		go escutarBroker(conn, connFechada) // Worker para parsing de mensagens de rede
		go loopEntrada(conn)                // Worker para captura de entrada do usuário

		<-connFechada // Bloqueia até que a conexão seja perdida
		conn.Close()
	}
}

// loopEntrada: Gerencia a interação assíncrona do usuário.
// Permite que a interface responda a comandos enquanto a rede continua processando dados.
func loopEntrada(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	for conectado {
		input, _ := reader.ReadString('\n')
		opcao := strings.TrimSpace(input)

		// Lógica de escape para modo de monitoramento
		if telaAtual == "MONITORANDO" {
			mutex.Lock()
			telaAtual = "ESCOLHER_SENSOR"
			mutex.Unlock()
			renderizar()
			continue
		}

		if opcao != "" {
			handleInput(opcao, conn)
			renderizar()
		}
	}
}

// handleInput: Máquina de Estados da Interface (Menu de Navegação).
// Processa as escolhas lógicas e converte em comandos de protocolo TCP.
func handleInput(opcao string, conn net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()

	if !conectado {
		return
	}
	erroMsg = ""

	switch telaAtual {
	case "MENU":
		if opcao == "1" {
			telaAtual = "ESCOLHER_SENSOR"
		} else if opcao == "2" {
			telaAtual = "ATUADORES"
		} else if opcao == "0" {
			os.Exit(0)
		}
	case "ESCOLHER_SENSOR":
		if opcao == "0" {
			telaAtual = "MENU"
		} else if strings.ToLower(opcao) == "todos" {
			sensorAlvo = "TODOS"
			telaAtual = "MONITORANDO"
		} else {
			escolha, err := strconv.Atoi(opcao)
			if err == nil && escolha > 0 && escolha <= len(indicesSensores) {
				sensorAlvo = indicesSensores[escolha-1]
				telaAtual = "MONITORANDO"
			}
		}
	case "ATUADORES":
		if opcao == "0" {
			telaAtual = "MENU"
		} else {
			escolha, err := strconv.Atoi(opcao)
			if err == nil && escolha > 0 && escolha <= len(indicesAtuadores) {
				idAlvo := indicesAtuadores[escolha-1]
				cmd := "LIGAR"
				if atuadores[idAlvo] {
					cmd = "DESLIGAR"
				}
				fmt.Fprintf(conn, "COMMAND|%s|%s\n", idAlvo, cmd) // Dispatch de comando
			}
		}
	}
}

// escutarBroker: Listener assíncrono para mensagens de rede.
// Realiza o parsing de STATUS (TCP) e Telemetria (replicada do UDP).
func escutarBroker(conn net.Conn, done chan bool) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()
		mutex.Lock()
		// Parsing de telemetria baseada em caractere delimitador ":"
		if strings.Contains(msg, ":") {
			parts := strings.Split(msg, ":")
			if len(parts) >= 2 {
				telemetria[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(msg, "STATUS|") {
			// Atualização de estado de dispositivos baseada em protocolo de mensagens
			parts := strings.Split(msg, "|")
			if len(parts) >= 3 {
				idDispositivo := parts[1]
				status := parts[2]

				if status == "OFFLINE" {
					delete(atuadores, idDispositivo)
					delete(telemetria, idDispositivo)
				} else {
					atuadores[idDispositivo] = (status == "true")
				}
			}
		}
		podeRedesenhar := (telaAtual == "MONITORANDO" || telaAtual == "ATUADORES")
		mutex.Unlock()
		if podeRedesenhar {
			renderizar()
		}
	}
	mutex.Lock()
	conectado = false
	mutex.Unlock()
	done <- true
}

// renderizar: Motor gráfico do terminal.
// Centraliza toda a lógica de desenho para garantir consistência visual.
func renderizar() {
	mutex.Lock()
	defer mutex.Unlock()

	fmt.Print("\033[H\033[2J")
	fmt.Println("====================================================")
	statusStr := "ONLINE"
	if !conectado {
		statusStr = "OFFLINE"
	}
	fmt.Printf("   SISTEMA ROTA DAS COISAS [%s] - %s\n", statusStr, clientID)
	fmt.Println("====================================================")

	if erroMsg != "" {
		fmt.Printf("\n[AVISO] %s\n----------------------------------------------------\n", erroMsg)
	}
	if !conectado {
		return
	}

	switch telaAtual {
	case "MENU":
		fmt.Println("\n[1] Ver Sensores\n[2] Ver Atuadores\n[0] Sair\n\nSua escolha: ")
	case "ESCOLHER_SENSOR":
		fmt.Println("\nSENSORES DISPONÍVEIS:")
		indicesSensores = []string{}
		keys := []string{}
		for k := range telemetria {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, id := range keys {
			indicesSensores = append(indicesSensores, id)
			fmt.Printf(" [%d] %s\n", i+1, id)
		}
		fmt.Println("\n[TODOS] Acompanhar tudo | [0] Voltar")
	case "MONITORANDO":
		fmt.Printf("\n>>> MONITORANDO: %s <<<\n", sensorAlvo)
		keys := []string{}
		for k := range telemetria {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, id := range keys {
			if sensorAlvo == "TODOS" || id == sensorAlvo {
				fmt.Printf("%-15s: %s\n", id, telemetria[id])
			}
		}
		fmt.Println("\n[ ENTER para voltar ]")
	case "ATUADORES":
		fmt.Println("\nESTADO DOS ATUADORES:")
		indicesAtuadores = []string{}
		keys := []string{}
		for k := range atuadores {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, id := range keys {
			indicesAtuadores = append(indicesAtuadores, id)
			statusStr := "DESLIGADO"
			if atuadores[id] {
				statusStr = "LIGADO"
			}
			fmt.Printf(" [%d] %-15s | STATUS: %s\n", i+1, id, statusStr)
		}
		fmt.Println("\n[0] Voltar")
	}
}
