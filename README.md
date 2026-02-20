# Pagamento Service (Go)

Serviço de processamento de pagamentos integrado ao **Mercado Pago** (Modelo QR Code Dinâmico) e **AWS (DynamoDB & SNS)**. Este microserviço faz parte do ecossistema de Oficina, sendo responsável por gerar cobranças e notificar outros sistemas sobre o status dos pagamentos.

## 🚀 Tecnologias

- **Go 1.23.4**
- **Gin Gonic** (Framework Web)
- **AWS SDK v2** (DynamoDB e SNS)
- **Mercado Pago API** (Integração v1/orders)
- **Resty** (Cliente HTTP)
- **Zap** (Structured Logging)
- **GitHub Actions** (CI/CD Pipeline)
- **Docker & Kubernetes** (Kustomize)

## 🏗️ Arquitetura

O projeto segue os princípios da **Clean Architecture**, organizado da seguinte forma:

- `cmd/server`: Ponto de entrada da aplicação.
- `internal/api`: Camada de transporte (Handlers e Roteamento).
- `internal/domain`: Entidades de negócio e interfaces (Ports).
- `internal/service`: Regras de negócio e casos de uso.
- `internal/repository`: Implementação de persistência (DynamoDB).
- `internal/integration`: Clientes para serviços externos (Mercado Pago, SNS).

## 🛠️ Fluxo de Integração

1.  **Ordem de Serviço** chama `POST /v1/pagamentos`.
2.  O serviço solicita um **QR Code Dinâmico** ao Mercado Pago.
3.  O QR Code é retornado e armazenado no **DynamoDB** com status `pending`.
4.  O cliente realiza o pagamento via App Mercado Pago.
5.  O Mercado Pago envia um **Webhook** para `POST /v1/webhooks/mercadopago`.
6.  O serviço **valida a assinatura** do webhook (HMAC-SHA256) para garantir a segurança.
7.  Após processar o status, o serviço publica um evento no **AWS SNS**.
8.  O serviço de **Ordem de Serviço** (ou outros) consome este evento via SQS para atualizar seu fluxo interno.

## ⚙️ Configuração

Crie um arquivo `.env` baseado no `.env.example`:

```env
MERCADO_PAGO_ACCESS_TOKEN=seu_token
MERCADO_PAGO_POS_ID=seu_pos_id
MERCADO_PAGO_WEBHOOK_SECRET=sua_chave_secreta
AWS_REGION=us-east-1
DYNAMODB_TABLE_NAME=Payments
AWS_SNS_TOPIC_ARN=arn:aws:sns:us-east-1:602900801621:sns-pagamentos-notifacoes
```

## 🏃 Como Rodar

### Localmente
```bash
go run cmd/server/main.go
```

### Com Docker Compose
```bash
docker-compose up -d
```

## 🧪 Testes
```bash
go test ./...
```

## 🔐 Segurança do Webhook
Este serviço implementa a validação de assinatura do Mercado Pago. Todas as requisições de webhook são verificadas usando a chave secreta configurada no `MERCADO_PAGO_WEBHOOK_SECRET` e o header `x-signature`, garantindo que apenas o Mercado Pago possa notificar atualizações de status.

## 📦 CI/CD
O projeto conta com pipelines automatizados no GitHub Actions:
- **CI**: Executa testes e valida o build do Docker em branches de feature.
- **CD**: Publica a imagem no Docker Hub e realiza o deploy no AWS EKS ao realizar push na branch `main`.

---
Desenvolvido por Alex Marques Fonseca


## Arquitetura
Você pode visualizar o desenho de arquitetura do sistema aqui: [Desenho de Arquitetura AWS](docs/aws.html)
