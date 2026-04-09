package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

// main: Ponto de entrada do simulador de sensores.
// Gerencia a geração de dados pseudo-aleatórios e transmissão via datagramas UDP.
func main() {
	// Injeção de dependências via variáveis de ambiente para deploy distribuído
	sensorID := os.Getenv("SENSOR_ID")
	sensorType := os.Getenv("SENSOR_TYPE")
	unit := os.Getenv("SENSOR_UNIT")

	if sensorID == "" {
		sensorID = "S_RACK_01"
	}
	if sensorType == "" {
		sensorType = "TEMPERATURA"
	}

	// Resolução dinâmica do IP do Broker
	enderecoBroker := os.Getenv("BROKER_ADDR")
	if enderecoBroker == "" {
		enderecoBroker = "127.0.0.1"
	}
	if !strings.Contains(enderecoBroker, ":") {
		enderecoBroker = enderecoBroker + ":9001"
	}

	// Resolve o endereço UDP e prepara o socket para transmissão
	udpAddr, err := net.ResolveUDPAddr("udp", enderecoBroker)
	if err != nil {
		fmt.Printf("Erro de DNS/Endereço: %v\n", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Printf("Falha na conexão UDP: %v\n", err)
		return
	}
	defer conn.Close()

	// PADRÃO RESTAURADO: Inicia a escuta do teclado em background
	go monitorKeyboard(sensorID)

	// Configuração de parâmetros físicos por tipo de grandeza simulada
	var valorAtual, min, max, variacaoMax float64
	switch sensorType {
	case "TEMPERATURA":
		valorAtual, min, max, variacaoMax = 22.0, 18.0, 32.0, 0.5
	case "UMIDADE":
		valorAtual, min, max, variacaoMax = 50.0, 30.0, 70.0, 1.2
	case "ENERGIA":
		valorAtual, min, max, variacaoMax = 10.0, 8.0, 14.0, 0.3
	default:
		valorAtual, min, max, variacaoMax = 50.0, 0.0, 100.0, 5.0
	}

	fmt.Printf(">>> [%s] Sensor de %s conectado em %s\n", sensorID, sensorType, enderecoBroker)
	fmt.Println("COMANDO: [0] Encerrar Sensor")
	fmt.Println(strings.Repeat("-", 40))

	// Tempo de espera para Docker a sincronizar
	time.Sleep(100 * time.Millisecond)

	// Loop Infinito de Telemetria (Duty Cycle de 500ms)
	for {
		delta := (rand.Float64()*2 - 1) * variacaoMax
		valorAtual += delta

		if valorAtual < min {
			valorAtual = min + rand.Float64()
		} else if valorAtual > max {
			valorAtual = max - rand.Float64()
		}

		msg := fmt.Sprintf("%s: %.2f%s", sensorID, valorAtual, unit)
		conn.Write([]byte(msg))

		// \r permite atualizar a mesma linha no terminal
		fmt.Printf("\r[TELEMETRIA UDP] %s", msg)

		time.Sleep(500 * time.Millisecond)
	}
}

// monitorKeyboard: Listener para comando local de interrupção forçada (Igual ao Atuador).
func monitorKeyboard(id string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) == "0" {
			fmt.Printf("\n[!] Encerrando Sensor %s... Limpando recursos.\n", id)
			os.Exit(0)
		}
	}
}
