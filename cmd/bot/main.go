package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	endereco := os.Getenv("BROKER_ADDR")
	if endereco == "" {
		endereco = "127.0.0.1:9000"
	}

	const numUsuarios = 5     // Simula 5 pessoas clicando ao mesmo tempo
	const reqPorUsuario = 100 // Cada uma clica 100 vezes
	var wg sync.WaitGroup
	var totalEnviado uint64
	var mutex sync.Mutex

	fmt.Printf("🚀 Iniciando Estresse: %d usuários x %d cliques...\n", numUsuarios, reqPorUsuario)

	for i := 1; i <= numUsuarios; i++ {
		wg.Add(1)
		go func(idUsuario int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", endereco)
			if err != nil {
				return
			}
			defer conn.Close()

			fmt.Fprintf(conn, "IDENTIFY|CLIENT|BOT_%d\n", idUsuario)

			for seq := 1; seq <= reqPorUsuario; seq++ {
				// Alterna entre LIGAR e DESLIGAR para o interruptor oscilar
				comando := "LIGAR"
				if seq%2 == 0 {
					comando = "DESLIGAR"
				}

				// Enviamos o comando EXATO que o Broker espera
				_, err := fmt.Fprintf(conn, "COMMAND|ACT_RESILIENTE_01|%s\n", comando)
				if err != nil {
					break
				}

				mutex.Lock()
				totalEnviado++
				mutex.Unlock()

				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("\n\n✅ TESTE CONCLUÍDO")
	fmt.Printf("\nTotal Enviado pelo Bot: %d", totalEnviado)
	fmt.Printf("\nConfira se o 'Total de Requisições' no Broker é >= %d\n", totalEnviado)
}
