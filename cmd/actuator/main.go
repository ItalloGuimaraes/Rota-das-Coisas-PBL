package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// main: Orquestra a conexão e o gerenciamento de estados do atuador.
func main() {
	actuatorID := os.Getenv("ACTUATOR_ID")
	actuatorType := os.Getenv("ACTUATOR_TYPE")

	if actuatorID == "" {
		actuatorID = "ACT_RESILIENTE_01"
	}
	if actuatorType == "" {
		actuatorType = "CONTROLE"
	}

	// Inicia a escuta do teclado em background para controle local (Item 7)
	go monitorKeyboard(actuatorID)

	// Endereçamento dinâmico via variável de ambiente para suporte a Docker e LARSID
	enderecoBroker := os.Getenv("BROKER_ADDR")
	if enderecoBroker == "" {
		enderecoBroker = "127.0.0.1"
	}
	if !strings.Contains(enderecoBroker, ":") {
		enderecoBroker = enderecoBroker + ":9000"
	}

	fmt.Printf(">>> [%s] Iniciando Atuador de %s\n", actuatorID, actuatorType)
	fmt.Println(strings.Repeat("-", 40))

	// Tempo de espera para Docker a sincronizar
	time.Sleep(100 * time.Millisecond)

	// Loop de Resiliência: Tentativa de recuperação de conexão TCP em caso de falha do Broker
	for {
		fmt.Printf("[%s] Tentando conectar ao Broker em %s...\n", time.Now().Format("15:04:05"), enderecoBroker)

		conn, err := net.Dial("tcp", enderecoBroker) // Abertura de canal confiável
		if err != nil {
			fmt.Printf("!!! Falha na conexão. Nova tentativa em 5s...\n")
			time.Sleep(5 * time.Second)
			continue
		}

		handleBrokerConnection(conn, actuatorID, actuatorType)

		fmt.Println("!!! Conexão perdida. Reiniciando ciclo de recuperação...")
		time.Sleep(2 * time.Second)
	}
}

// monitorKeyboard: Listener para comando local de interrupção forçada.
func monitorKeyboard(id string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) == "0" {
			fmt.Printf("\n[!] Encerrando Atuador %s... Limpando recursos.\n", id)
			os.Exit(0)
		}
	}
}

// handleBrokerConnection: Implementa o loop de processamento de comandos remotos.
func handleBrokerConnection(conn net.Conn, id, tipo string) {
	defer conn.Close()
	// Identificação perante o Broker conforme protocolo estabelecido
	fmt.Fprintf(conn, "IDENTIFY|ACTUATOR|%s\n", id)

	ligado := false
	fmt.Printf(">>> Atuador [%s] ONLINE e registrado!\n", id)
	renderStatus(id, tipo, "DESLIGADO")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()
		parts := strings.Split(msg, "|")
		// Filtro de mensagens: Apenas pacotes do tipo ACTION são processados
		if len(parts) < 2 || parts[0] != "ACTION" {
			continue
		}

		acao := parts[1]
		fmt.Printf("[%s] Comando recebido: %s\n", time.Now().Format("15:04:05"), acao)

		// Lógica de inversão de estado baseada no payload textual
		switch acao {
		case "LIGAR", "ATIVAR", "ABRIR":
			ligado = true
		case "DESLIGAR", "DESATIVAR", "FECHAR":
			ligado = false
		}

		statusVisual := "DESLIGADO"
		if ligado {
			statusVisual = "LIGADO"
		}
		renderStatus(id, tipo, statusVisual)
	}
}

// renderStatus: Exibe o feedback visual do estado do hardware simulado.
func renderStatus(id, tipo, status string) {
	fmt.Println("\n-------------------------------------------")
	fmt.Printf(" DISPOSITIVO: %s (%s)\n", id, tipo)
	fmt.Printf(" ESTADO ATUAL: %s\n", status)
	fmt.Println("-------------------------------------------")
	fmt.Println("COMANDO: [0] Encerrar Atuador") // RESTAURADO: Comando de encerramento local
}
