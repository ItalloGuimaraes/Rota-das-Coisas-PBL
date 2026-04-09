# 🌐 A Rota das Coisas: Serviço de Integração de Data Center

<div align="center">

![Go](https://img.shields.io/badge/Language-Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Architecture](https://img.shields.io/badge/Architecture-Distributed-green?style=for-the-badge)

<p><i>Solução para o Problema 1 da disciplina TEC502 - Sistemas Operacionais e Conectividade (UEFS)</i></p>

</div>

---

## 📌 Descrição

Este projeto implementa um **serviço de integração (Broker)** para sistemas IoT industriais, resolvendo o problema de **alto acoplamento (ponto-a-ponto)** entre dispositivos.

A solução centraliza a comunicação entre sensores, atuadores e aplicações cliente, permitindo:

- Escalabilidade
- Redução de sobrecarga nos dispositivos
- Separação entre produtores e consumidores de dados

---

## 🎯 Objetivo

Eliminar conexões diretas entre dispositivos IoT, utilizando um **Broker centralizado** que:

- Recebe telemetria via **UDP (baixa latência)**
- Processa comandos via **TCP (alta confiabilidade)**
- Gerencia múltiplas conexões simultâneas com segurança

---

## 🏗️ Arquitetura do Sistema

O sistema adota uma arquitetura em **topologia estrela**, onde todos os componentes se comunicam exclusivamente com o Broker.

### Componentes:

- **Broker**
  - Centraliza comunicação
  - Realiza parsing e roteamento
  - Mantém estado dos dispositivos

- **Sensores (UDP)**
  - Envio contínuo de telemetria
  - Baixo custo computacional

- **Atuadores (TCP)**
  - Recebem comandos críticos
  - Executam ações controladas

- **Cliente (TCP)**
  - Interface de monitoramento
  - Controle remoto de dispositivos

---

## 🏗️ Diagrama da Arquitetura

```
              +-----------------------+
              |   Cliente (TCP)       |
              | Monitoramento/Controle|
              +----------+------------+
                         |
                         | TCP (Comandos / Status)
                         |
               +---------v-----------+
               |      BROKER         |
               | Serviço Central     |
               | (Goroutines + Mutex)|
               +----+---------+------+
                    |         |
        UDP (Dados) |         | TCP (Ações)
                    |         |
        +-----------+         +-----------+
        |                                 |
+-------v------+                 +--------v------+
|   Sensor 1   |                 |   Atuador 1   |
| Temperatura  |                 |    Alarme     |
+--------------+                 +---------------+

+--------------+                 +---------------+
|   Sensor 2   |                 |   Atuador 2   |
|   Umidade    |                 |   Ventilação  |
+--------------+                 +---------------+

Legenda:
- UDP → Telemetria contínua (rápida, sem garantia)
- TCP → Comandos críticos (confiável)
```

## 🔌 Comunicação e Protocolo

O sistema utiliza **sockets nativos TCP/UDP**, conforme exigido pelo problema :contentReference[oaicite:0]{index=0}.

### 📡 Estratégia de Comunicação

| Tipo         | Protocolo | Justificativa |
|--------------|----------|--------------|
| Telemetria   | UDP      | Baixa latência, tolerância a perdas |
| Comandos     | TCP      | Confiabilidade e integridade |

---

## 📑 Especificação do Protocolo (API)

O protocolo foi desenvolvido com base em **texto delimitado (`|` e `:`)** para facilitar parsing manual eficiente.

### 📥 Formatos de Mensagem

| Tipo            | Canal       | Formato                         | Exemplo |
|-----------------|------------|----------------------------------|--------|
| Telemetria      | UDP:9001   | `ID: VALOR_UNIDADE`             | `TEMP_01: 24.5°C` |
| Identificação   | TCP:9000   | `IDENTIFY\|TIPO\|ID`            | `IDENTIFY\|ACTUATOR\|AC_SUL` |
| Comando         | TCP:9000   | `COMMAND\|ALVO\|ACAO`           | `COMMAND\|AC_SUL\|LIGAR` |
| Ação            | TCP:9000   | `ACTION\|ACAO`                  | `ACTION\|LIGAR` |
| Status          | TCP:9000   | `STATUS\|ID\|ESTADO`            | `STATUS\|AC_SUL\|true` |

---

## ⚙️ Concorrência e Desempenho

Para atender múltiplas requisições simultâneas:

- Uso de **Goroutines (Go)**
- Controle de acesso com **Mutex**
- Processamento paralelo de conexões TCP
- Recepção contínua via UDP sem bloqueio

🔒 Isso evita:
- Race conditions
- Gargalos no Broker
- Perda de integridade dos dados

---

## 🛡️ Confiabilidade

O sistema implementa mecanismos básicos de tolerância a falhas:

- ⏱️ **Watchdog (5 segundos)**:
  - Remove dispositivos inativos automaticamente

- 🔌 Tratamento de desconexões:
  - Cliente é informado ao enviar comando para dispositivo offline

- ⚠️ Validação de mensagens:
  - Parsing com checagem de formato
  - Ignora mensagens inválidas

---

## 📊 Qualidade de Serviço (QoS)

Separação clara entre tipos de tráfego:

- **Telemetria (UDP)** → prioriza velocidade
- **Comandos (TCP)** → prioriza confiabilidade

Evita que alto volume de dados impacte operações críticas :contentReference[oaicite:1]{index=1}

---

## 🖥️ Interface e Interação

- Dashboard via terminal em tempo real
- Visualização de:
  - Telemetria agregada
  - Estado dos atuadores
- Envio de comandos manual pelo cliente

---

## 🧪 Testes e Validação

O sistema inclui:

- 🤖 **Bot de testes de estresse**
  - Simula múltiplos clientes simultâneos

- 📈 Testes realizados:
  - Alta taxa de envio UDP
  - Conexões concorrentes TCP
  - Simulação de falhas

---

## 🐳 Emulação com Docker

Todo o sistema é executado em containers Docker, garantindo:

- Reprodutibilidade
- Escalabilidade
- Execução distribuída no laboratório

---

## 📂 Estrutura do Projeto

```bash
├── cmd/
│   ├── actuator/   # Atuador (TCP)
│   ├── bot/        # Testes de carga
│   ├── broker/     # Serviço central
│   ├── client/     # Cliente
│   └── sensor/     # Sensor (UDP)
├── .env
├── Dockerfile
└── docker-compose.yml

```

## 🚀 Execução

### 1. Configuração

No .env defina o IP da máquina, onde o Broker será executado:

```env
IP_BROKER=172.16.201.11
````

---

### 2. Subir o Broker

```bash
docker compose up -d broker
```

---

### 3. Subir Dispositivos (Sensores e Atuadores)

```bash
docker compose up -d (Nome do Container)
```

---

### 4. Executar Testes de Carga

```bash
docker compose --profile teste up -d
```

---

### 5. Cliente Interativo

```bash
docker compose run --rm cliente_ti
```

---

### 6. Criar Dispositivo Manual

```bash
docker run -d --network host \
  -e BROKER_ADDR=172.16.201.11:9001 \
  -e SENSOR_ID=SENSOR_EXTRA \
  -e SENSOR_TYPE=TEMPERATURA \
  -e SENSOR_UNIT=°C \
  italloguimaraes/sensor-dc:latest
```

---

## 👨‍💻 Autor

**Ítallo de Santana Guimarães**
Estudante de Engenharia de Computação - UEFS
📧 [italloguimaraes1@uefs.br](mailto:italloguimaraes1@uefs.br)

---

## 📚 Referências

* 📘 Documentação oficial da linguagem Go (pacote net)
  [https://pkg.go.dev/net](https://pkg.go.dev/net)

* 📘 Documentação oficial da linguagem Go (concorrência)
  [https://go.dev/doc/effective_go#concurrency](https://go.dev/doc/effective_go#concurrency)

* 🐳 Documentação Docker
  [https://docs.docker.com](https://docs.docker.com)

* 🐳 Docker Compose
  [https://docs.docker.com/compose/](https://docs.docker.com/compose/)

* 🌐 Modelo TCP/IP (conceitos de rede)
  [https://www.cloudflare.com/learning/ddos/glossary/tcp-ip/](https://www.cloudflare.com/learning/ddos/glossary/tcp-ip/)

* 📡 Comunicação via Sockets (TCP/UDP)
  [https://www.geeksforgeeks.org/socket-programming-in-go/](https://www.geeksforgeeks.org/socket-programming-in-go/)

---

## 📄 Licença

Este projeto está licenciado sob os termos da **MIT License**.

🔗 Veja o arquivo completo em:
[LICENSE](./LICENSE)
