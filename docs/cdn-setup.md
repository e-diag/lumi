# Настройка Яндекс CDN для резервного подключения

## Зачем
Когда операторы РФ включают белые списки, прямое подключение к EU/USA-нодам может блокироваться.  
Яндекс Cloud CDN часто остается доступным — трафик можно вести через него как fallback.

## Настройка Яндекс Cloud CDN
1. Яндекс Cloud Console -> CDN -> Создать ресурс.
2. Источник: `https://{EU_NODE_IP}:8443`.
3. Домен: `vpn.freeway.app` (CNAME -> `*.edgecdn.ru`).
4. SSL: выпустить сертификат через Яндекс.
5. Передавать все заголовки, кэш отключить (`Cache-Control: no-store`).

## WebSocket inbound на EU-ноде (Xray)
```json
{
  "tag": "ws-cdn",
  "port": 8443,
  "protocol": "vless",
  "streamSettings": {
    "network": "ws",
    "security": "tls",
    "wsSettings": { "path": "/vless-yndx" },
    "tlsSettings": {
      "certificates": [{
        "certificateFile": "/etc/ssl/freeway.crt",
        "keyFile": "/etc/ssl/freeway.key"
      }]
    }
  }
}
```

## Проверка
- Premium-подписка содержит CDN-конфиг последним.
- Для Free/Basic CDN-конфиг не выдается.
