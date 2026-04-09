package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Estruturas de dados globais para manutenção do estado do sistema.
// O uso de Mapas permite acesso O(1) para busca de dispositivos.
var (
	sensorData     = make(map[string]string)    // Armazena a última leitura de cada sensor
	sensorLastSeen = make(map[string]time.Time) // Timestamp para detecção de inatividade (Watchdog)
	actuators      = make(map[string]net.Conn)  // Pool de sockets TCP ativos para atuadores
	actuatorStates = make(map[string]bool)      // Estado lógico (LIGADO/DESLIGADO) dos atuadores
	clients        = make(map[net.Conn]string)  // Pool de sockets TCP para aplicações clientes
	totalRequests  uint64                       // Contador global
	mutex          sync.Mutex                   // Primitiva de sincronização para evitar Condições de Corrida (Race Conditions)
)

// renderDashboard: Função de Interface Humano-Máquina (IHM) via terminal.
// Utiliza sequências de escape ANSI para limpar o buffer de saída e redesenhar a telemetria.
func renderDashboard() {
	fmt.Print("\033[H\033[2J") // Limpa o terminal e move cursor para o topo
	fmt.Println("====================================================")
	fmt.Println("         BROKER DE INTEGRAÇÃO - DATA CENTER         ")
	fmt.Println("====================================================")

	mutex.Lock()         // Bloqueia acesso concorrente aos mapas durante a leitura
	defer mutex.Unlock() // Garante a liberação do lock após a execução da função

	fmt.Println("\n[TELEMETRIA - SENSORES (UDP)]")
	fmt.Printf("%-25s | %-20s\n", "ID DO SENSOR", "VALOR ATUAL")
	fmt.Println(strings.Repeat("-", 50))

	// Ordenação determinística das chaves para evitar "pulo" visual no dashboard
	sensorKeys := make([]string, 0, len(sensorData))
	for k := range sensorData {
		sensorKeys = append(sensorKeys, k)
	}
	sort.Strings(sensorKeys)

	for _, id := range sensorKeys {
		fmt.Printf("%-25s | %-20s\n", id, sensorData[id])
	}

	fmt.Println("\n[ATUADORES DE INFRAESTRUTURA (TCP)]")
	fmt.Printf("%-25s | %-20s\n", "ID DO ATUADOR", "ESTADO")
	fmt.Println(strings.Repeat("-", 50))

	actuatorKeys := make([]string, 0, len(actuatorStates))
	for k := range actuatorStates {
		actuatorKeys = append(actuatorKeys, k)
	}
	sort.Strings(actuatorKeys)

	for _, id := range actuatorKeys {
		estado := "DESLIGADO"
		if actuatorStates[id] {
			estado = "LIGADO"
		}
		fmt.Printf("%-25s | %-20s\n", id, estado)
	}

	fmt.Printf("\nClientes Conectados: %d | Atuadores Online: %d\n", len(clients), len(actuators))
	fmt.Println("====================================================")
	fmt.Printf("Última Sincronização: %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("Total de Requisições Processadas: %d\n", totalRequests)
	fmt.Println("COMANDO: [0] Encerrar Servidor")
	fmt.Println("====================================================")
}

// checkSensorTimeouts: Monitor de integridade de sensores (Keep-alive/Timeout).
// Executa em uma goroutine separada verificando se o sensor parou de transmitir dados UDP.
func checkSensorTimeouts() {
	for {
		time.Sleep(2 * time.Second)
		mutex.Lock()
		mudou := false
		for id, lastSeen := range sensorLastSeen {
			// Se o tempo desde a última mensagem for > 5s, o sensor é considerado offline
			if time.Since(lastSeen) > 5*time.Second {
				delete(sensorData, id)
				delete(sensorLastSeen, id)
				mudou = true

				// Notifica todos os clientes que o sensor caiu
				msgOffline := fmt.Sprintf("STATUS|%s|OFFLINE\n", id)
				for conn := range clients {
					fmt.Fprint(conn, msgOffline)
				}
			}
		}
		mutex.Unlock()

		if mudou {
			renderDashboard() // Atualiza UI se houver alteração na lista de dispositivos
		}
	}
}

// processTelemetry: Parser de pacotes UDP.
// Realiza a decomposição da string de telemetria e atualiza o estado volátil do Broker.
func processTelemetry(msg string) {
	parts := strings.Split(msg, ":")
	if len(parts) >= 2 {
		id := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		mutex.Lock()
		sensorData[id] = val
		sensorLastSeen[id] = time.Now() // Atualiza timestamp de atividade
		mutex.Unlock()
		renderDashboard()
		broadcastToClients(msg) // Encaminha dados para todos os clientes TCP ativos
	}
}

// broadcastToClients: Implementação do padrão Observer/Pub-Sub.
// Replica as mensagens de telemetria para todas as conexões de clientes em tempo real.
func broadcastToClients(message string) {
	mutex.Lock()
	defer mutex.Unlock()
	for conn := range clients {
		fmt.Fprint(conn, message+"\n") // Escrita síncrona no buffer de rede TCP
	}
}

// listenUDP: Listener para protocolo de transporte não-orientado a conexão.
// Otimizado para baixa latência, ideal para o tráfego massivo de sensores.
func listenUDP() {
	addr, _ := net.ResolveUDPAddr("udp", ":9001")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Erro Fatal UDP:", err)
		return
	}
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf) // Operação bloqueante aguardando datagramas
		if err == nil {
			processTelemetry(string(buf[:n]))
		}
	}
}

// monitorKeyboard: Listener local para controle administrativo.
// Permite o encerramento gracioso do serviço através da entrada padrão (stdin).
func monitorKeyboard() {
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) == "0" {
			fmt.Printf("\n[!] Encerrando Broker...\n")
			os.Exit(0)
		}
	}
}

// handleConnection: Gerenciador de ciclo de vida de conexões TCP.
// Implementa identificação de dispositivos e mantém a persistência da conexão (Keep-alive).
func handleConnection(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	var deviceType, deviceID string

	// Handshake Inicial: O dispositivo deve se identificar no primeiro pacote
	if scanner.Scan() {
		parts := strings.Split(scanner.Text(), "|")
		if len(parts) < 3 || parts[0] != "IDENTIFY" {
			conn.Close()
			return
		}

		deviceType, deviceID = parts[1], parts[2]

		mutex.Lock()
		switch deviceType {
		case "ACTUATOR":
			actuators[deviceID] = conn
			actuatorStates[deviceID] = false
			// Notifica clientes sobre novo atuador online
			msg := fmt.Sprintf("STATUS|%s|false\n", deviceID)
			for c := range clients {
				fmt.Fprint(c, msg)
			}
		case "CLIENT":
			clients[conn] = deviceID
			// Sincroniza estado atual dos atuadores para o novo cliente
			for id, state := range actuatorStates {
				fmt.Fprintf(conn, "STATUS|%s|%t\n", id, state)
			}
		}
		mutex.Unlock()
		renderDashboard()

		// Loop de escuta ativa para comandos vindos de clientes
		for scanner.Scan() {
			msg := scanner.Text()
			if deviceType == "CLIENT" && strings.HasPrefix(msg, "COMMAND|") {
				routeCommand(msg)
			}
		}
	}

	// Cleanup: Remoção de dispositivos desconectados da memória
	mutex.Lock()
	if deviceType == "CLIENT" {
		delete(clients, conn)
	} else if deviceType == "ACTUATOR" {
		delete(actuators, deviceID)
		delete(actuatorStates, deviceID)
		// Notifica clientes sobre perda de conexão do atuador
		msg := fmt.Sprintf("STATUS|%s|OFFLINE\n", deviceID)
		for c := range clients {
			fmt.Fprint(c, msg)
		}
	}
	mutex.Unlock()
	conn.Close()
	renderDashboard()
}

// routeCommand: Mecanismo de Roteamento de Comandos (Client -> Broker -> Actuator).
// Traduz requisições lógicas em ações físicas sobre os atuadores TCP.
func routeCommand(msg string) {
	parts := strings.Split(msg, "|")
	if len(parts) < 3 {
		return
	}
	targetID, comando := parts[1], parts[2]

	mutex.Lock()
	totalRequests++ // Incrementa toda vez que um comando é lido e roteado
	defer mutex.Unlock()

	// Localiza o socket do atuador alvo e encaminha o comando de ação
	if targetConn, ok := actuators[targetID]; ok {
		fmt.Fprintf(targetConn, "ACTION|%s\n", comando)

		// Atualiza estado interno e reflete alteração para todos os clientes
		novoEstado := (comando == "LIGAR" || comando == "ATIVAR")
		actuatorStates[targetID] = novoEstado
		statusMsg := fmt.Sprintf("STATUS|%s|%t\n", targetID, novoEstado)
		for clientConn := range clients {
			fmt.Fprint(clientConn, statusMsg)
		}
	}
}

// main: Ponto de entrada do sistema.
// Orquestra as goroutines de listeners UDP/TCP e o monitoramento de timeouts.
func main() {
	go listenUDP()
	go monitorKeyboard()
	go checkSensorTimeouts()

	// Listener TCP para conexões confiáveis (Porta 9000)
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Erro Fatal TCP:", err)
		return
	}
	defer ln.Close()
	renderDashboard()

	for {
		conn, err := ln.Accept() // Aceita novas conexões de Clientes e Atuadores
		if err == nil {
			go handleConnection(conn) // Multiplexação: Cada conexão ganha uma goroutine
		}
	}
}
