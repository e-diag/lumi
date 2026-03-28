# CDN-нода: gRPC inbound (без TLS-over-TLS)

## Дополнительный inbound на EU-ноде

```json
{
  "tag": "grpc-cdn-in",
  "port": 8443,
  "protocol": "vless",
  "settings": {
    "clients": [],
    "decryption": "none"
  },
  "streamSettings": {
    "network": "grpc",
    "security": "none",
    "grpcSettings": {
      "serviceName": "vless"
    }
  }
}
```

ВАЖНО: `security=none` на inbound потому что TLS терминируется на Яндекс CDN,  
а не на нашей ноде. Это устраняет проблему TLS-over-TLS.  
Яндекс CDN -> наша нода соединение уже в plain HTTP/2 внутри датацентра.

