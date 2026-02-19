#!/bin/bash

# Cores para o output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Testando Microserviço de Pagamento ===${NC}"

# 1. Criar um pagamento
echo -e "
${BLUE}1. Criando um pagamento (POST /v1/payments)...${NC}"
CREATE_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/payments 
  -H "Content-Type: application/json" 
  -d '{
    "external_reference": "ORDER-'$(date +%s)'",
    "amount": 15.90,
    "description": "Combo Burger Teste"
  }')

echo -e "${GREEN}Resposta do Servidor:${NC}"
echo $CREATE_RESPONSE | jq . 2>/dev/null || echo $CREATE_RESPONSE

# Extrair ID e External Reference para o próximo passo
PAYMENT_ID=$(echo $CREATE_RESPONSE | jq -r '.id')
EXT_REF=$(echo $CREATE_RESPONSE | jq -r '.external_reference')

if [ "$PAYMENT_ID" != "null" ]; then
    echo -e "
${GREEN}Pagamento criado com ID: $PAYMENT_ID${NC}"
    
    # 2. Simular um Webhook de Aprovação
    # Nota: Em um cenário real, o ID no data.id viria do Mercado Pago
    echo -e "
${BLUE}2. Simulando recebimento de Webhook (POST /v1/webhooks/mercadopago)...${NC}"
    WEBHOOK_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/webhooks/mercadopago 
      -H "Content-Type: application/json" 
      -d '{
        "type": "payment",
        "action": "payment.created",
        "data": {
            "id": "123456789"
        }
      }')
    
    echo -e "${GREEN}Resposta do Webhook:${NC}"
    echo $WEBHOOK_RESPONSE | jq . 2>/dev/null || echo $WEBHOOK_RESPONSE
    echo -e "
${BLUE}Verifique os logs do servidor para ver o processamento do status.${NC}"
else
    echo -e "
${BLUE}Erro ao criar pagamento. Verifique se o servidor está rodando e se as credenciais no .env estão corretas.${NC}"
fi
