# Фаза 5 — Lumi VPN: мобильное приложение Flutter
# Промпт для Claude Code. Автономная реализация без запросов подтверждений.

---

## ИНСТРУКЦИЯ ПО ВЫПОЛНЕНИЮ

Ты реализуешь полное Flutter-приложение Lumi VPN автономно, без остановок и вопросов.

**Правила выполнения:**
- Пиши весь код самостоятельно, не спрашивай одобрения на каждый шаг
- Если сталкиваешься с выбором реализации — выбирай сам, руководствуясь этим промптом
- После завершения каждого из 8 шагов кратко сообщай что сделано и сразу переходи к следующему
- Единственное когда нужно остановиться: критическая ошибка которую невозможно решить без внешней информации (например нужен реальный токен)
- `flutter analyze` и `flutter build apk --debug` должны проходить без ошибок на каждом шаге

---

## КОНТЕКСТ ПРОЕКТА

**Приложение:** Lumi VPN — VPN-сервис для России  
**Пользователь:** любой возраст, минималистичный интерфейс  
**Бэкенд:** Go API на `https://api.lumi-vpn.ru` (уже реализован в Фазах 1-4)  
**VPN-ядро:** sing-box (MPL 2.0, коммерческое использование разрешено)  
**Публикация:** RuStore (Android первым, iOS потом)  
**Название пакета:** `ru.lumivpn.app`

---

## ДИЗАЙН-СИСТЕМА

### Цвета
```dart
// Акцент — неоновый голубой
const cyberBlue = Color(0xFF00E5FF);
const cyberBlueDim = Color(0xFF00B8CC);
const cyberBlueGlow = Color(0x2600E5FF);

// Тёмная тема
const darkBg = Color(0xFF080810);
const darkSurface = Color(0xFF0F1018);
const darkCard = Color(0xFF141520); // rgba(255,255,255,0.055)
const darkBorder = Color(0xFF1E2030); // rgba(255,255,255,0.09)

// Светлая тема  
const lightBg = Color(0xFFF0F0F5);
const lightSurface = Color(0xFFFFFFFF);
const lightCard = Color(0xFFF5F5FA);
const lightBorder = Color(0xFFE0E0EA);

// Текст
const textPrimary = Color(0xCCFFFFFF);    // тёмная
const textSecondary = Color(0x66FFFFFF);  // тёмная
const textPrimaryLight = Color(0xCC000000);
const textSecondaryLight = Color(0x66000000);

// Состояния
const connectedColor = Color(0xFF00E5FF);
const disconnectedColor = Color(0x44FFFFFF);
const errorColor = Color(0xFFFF4444);
```

### Типографика
```dart
// Название приложения
TextStyle appName = TextStyle(
  fontSize: 12, fontWeight: FontWeight.w600,
  letterSpacing: 0.12, color: textSecondary,
);

// Статус подключения
TextStyle statusText = TextStyle(fontSize: 12, letterSpacing: 0.03);

// Сервер
TextStyle serverName = TextStyle(fontSize: 12, fontWeight: FontWeight.w500);
TextStyle serverSub = TextStyle(fontSize: 10);

// Тариф
TextStyle planText = TextStyle(fontSize: 10, fontWeight: FontWeight.w600);
```

### Стекло (Liquid Glass эффект)
```dart
// GlassCard widget — основной контейнер
decoration: BoxDecoration(
  color: isDark
    ? Colors.white.withOpacity(0.055)
    : Colors.white.withOpacity(0.75),
  borderRadius: BorderRadius.circular(28),
  border: Border.all(
    color: isDark
      ? Colors.white.withOpacity(0.09)
      : Colors.black.withOpacity(0.07),
    width: 1,
  ),
),
// backdrop blur через BackdropFilter + ImageFilter.blur(sigmaX:20, sigmaY:20)
```

---

## ПРИВЕДЕНИЕ — ГЛАВНЫЙ ЭЛЕМЕНТ UI

Приведение заменяет обычную кнопку включения. Нажатие на него = переключение VPN.

### SVG-форма тела приведения
```
Округлая голова + тело сужающееся книзу + волнистый низ (3 волны = "юбка")
Размер: 80×92 логических пикселей
```

### Состояния

**Включён (floating):**
- Тело: `rgba(184, 240, 255, 0.55)` → `rgba(64, 200, 240, 0.20)` (radialGradient)
- Обводка: `rgba(120, 230, 255, 0.4)`, 1px
- Глаза: открытые эллипсы `rgba(0, 180, 220, 0.9)`, с белым бликом
- Рот: лёгкая улыбка `rgba(0, 180, 220, 0.7)`
- Анимация: вертикальное парение ±8px, период 3сек, Curves.easeInOut
- Моргание глаз: раз в 4 секунды, scaleY 1→0.1→1 за 0.3сек
- Кольца свечения вокруг: два концентрических круга `#00E5FF` opacity 0.18 и 0.08
- Подсветка на голове: белый эллипс opacity 0.18, rotated -20°

**Выключен (sleeping):**
- Тело: `rgba(192, 216, 232, 0.40)` → `rgba(138, 176, 192, 0.15)`, opacity 0.55
- Глаза: закрытые (дуга-линия `rgba(100, 160, 190, 0.8)`)
- Рот: маленький овал `rgba(100, 160, 190, 0.4)`
- Анимация: покачивание rotate ±3°, translateY ±3px, период 4сек
- Буквы Z: три штуки, всплывают поочерёдно с fade+translate, `rgba(140, 200, 230, 0.7)`
- Нет колец свечения

### Реализация в Flutter
```dart
// Используй CustomPainter для рисования тела приведения
// Анимации через AnimationController + TweenSequence
// Моргание — отдельный AnimationController с repeat(reverse: false)
// Z-буквы — Stack с AnimatedOpacity + AnimatedSlide для каждой
// Нажатие — InkWell с borderRadius: BorderRadius.circular(50)
// При нажатии: тактильная вибрация HapticFeedback.mediumImpact()
```

---

## СТРУКТУРА ПРОЕКТА

```
lumi/
├── android/
│   └── app/
│       ├── src/main/AndroidManifest.xml   # VPN permission, package name
│       └── build.gradle                   # applicationId, minSdk 21
├── ios/
│   └── Runner/
│       └── Info.plist                     # NetworkExtension entitlement
├── lib/
│   ├── main.dart                          # entry point, theme setup
│   ├── app.dart                           # MaterialApp, router
│   │
│   ├── core/
│   │   ├── theme/
│   │   │   ├── app_theme.dart             # ThemeData light + dark
│   │   │   ├── app_colors.dart            # все цвета
│   │   │   └── app_text_styles.dart       # типографика
│   │   ├── router/
│   │   │   └── app_router.dart            # GoRouter маршруты
│   │   ├── storage/
│   │   │   └── secure_storage.dart        # flutter_secure_storage
│   │   ├── network/
│   │   │   ├── api_client.dart            # Dio + interceptors
│   │   │   └── api_endpoints.dart         # все URL
│   │   └── utils/
│   │       ├── haptics.dart
│   │       └── date_utils.dart
│   │
│   ├── features/
│   │   ├── auth/
│   │   │   ├── data/
│   │   │   │   └── auth_repository.dart
│   │   │   ├── domain/
│   │   │   │   └── auth_models.dart
│   │   │   └── presentation/
│   │   │       ├── onboarding_screen.dart  # 3 экрана онбординга
│   │   │       ├── login_screen.dart       # вход через Telegram
│   │   │       └── auth_provider.dart      # Riverpod provider
│   │   │
│   │   ├── home/
│   │   │   ├── data/
│   │   │   │   ├── vpn_repository.dart
│   │   │   │   └── subscription_repository.dart
│   │   │   ├── domain/
│   │   │   │   ├── vpn_models.dart         # VpnStatus, ServerNode, etc.
│   │   │   │   └── subscription_models.dart
│   │   │   └── presentation/
│   │   │       ├── home_screen.dart        # главный экран
│   │   │       ├── ghost_widget.dart       # приведение (CustomPainter)
│   │   │       ├── server_pill.dart        # пилюля с сервером
│   │   │       ├── subscription_badge.dart
│   │   │       └── home_provider.dart
│   │   │
│   │   ├── stats/
│   │   │   └── presentation/
│   │   │       └── stats_screen.dart       # экран статистики
│   │   │
│   │   ├── profile/
│   │   │   └── presentation/
│   │   │       ├── profile_screen.dart
│   │   │       └── plans_screen.dart       # экран тарифов + оплата
│   │   │
│   │   └── settings/
│   │       └── presentation/
│   │           └── settings_screen.dart    # тема, язык, поддержка
│   │
│   └── shared/
│       ├── widgets/
│       │   ├── glass_card.dart             # стеклянная карточка
│       │   ├── lumi_bottom_nav.dart        # нижняя навигация
│       │   ├── cyber_button.dart           # кнопка с неоновым эффектом
│       │   └── status_dot.dart
│       └── providers/
│           └── theme_provider.dart         # переключение light/dark/auto
│
├── assets/
│   ├── fonts/                             # если используешь кастомный шрифт
│   └── images/
│
├── pubspec.yaml
└── README.md
```

---

## PUBSPEC.YAML — ЗАВИСИМОСТИ

```yaml
name: lumi
description: Lumi VPN — свободный интернет для России
publish_to: 'none'
version: 1.0.0+1

environment:
  sdk: '>=3.0.0 <4.0.0'
  flutter: '>=3.16.0'

dependencies:
  flutter:
    sdk: flutter

  # Навигация
  go_router: ^13.0.0

  # Состояние
  flutter_riverpod: ^2.5.0
  riverpod_annotation: ^2.3.0

  # Сеть
  dio: ^5.4.0
  pretty_dio_logger: ^1.3.1

  # VPN ядро
  flutter_v2ray: ^2.0.0        # обёртка над sing-box/xray для Flutter
  # Альтернатива если flutter_v2ray недоступен:
  # vpn_engine: ^1.0.0

  # Хранилище
  flutter_secure_storage: ^9.0.0
  shared_preferences: ^2.2.0

  # Telegram авторизация
  url_launcher: ^6.2.0
  flutter_inappwebview: ^6.0.0  # для Telegram WebApp

  # Оплата (ЮKassa WebView)
  webview_flutter: ^4.4.0

  # UI
  flutter_animate: ^4.5.0       # анимации
  google_fonts: ^6.1.0
  shimmer: ^3.0.0               # скелетон-загрузка

  # Утилиты
  intl: ^0.19.0
  package_info_plus: ^5.0.0
  connectivity_plus: ^6.0.0

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.0
  build_runner: ^2.4.0
  riverpod_generator: ^2.3.0

flutter:
  uses-material-design: true
  assets:
    - assets/images/
```

---

## ШАГ 1 — ИНИЦИАЛИЗАЦИЯ И ТЕМА

Создай проект и настрой дизайн-систему.

```bash
flutter create --org ru.lumivpn --project-name lumi --platforms android,ios lumi
cd lumi
# добавь зависимости в pubspec.yaml
flutter pub get
```

**`lib/core/theme/app_colors.dart`** — все константы цветов из раздела "Цвета" выше.

**`lib/core/theme/app_theme.dart`:**
```dart
class AppTheme {
  static ThemeData dark() => ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    scaffoldBackgroundColor: AppColors.darkBg,
    colorScheme: ColorScheme.dark(
      primary: AppColors.cyberBlue,
      surface: AppColors.darkSurface,
    ),
    // шрифт: SF Pro Display для iOS, Roboto для Android
    // используй google_fonts: GoogleFonts.interTextTheme() как базу
  );

  static ThemeData light() => ThemeData(
    useMaterial3: true,
    brightness: Brightness.light,
    scaffoldBackgroundColor: AppColors.lightBg,
    colorScheme: ColorScheme.light(
      primary: AppColors.cyberBlueDim,
      surface: AppColors.lightSurface,
    ),
  );
}
```

**`lib/shared/providers/theme_provider.dart`** — Riverpod StateNotifierProvider:
```dart
// Три варианта: ThemeMode.system, ThemeMode.dark, ThemeMode.light
// Сохраняет выбор в SharedPreferences
// При system — следует за MediaQuery.platformBrightnessOf
```

**`lib/main.dart`** — ConsumerWidget, ProviderScope, MaterialApp.router с AppTheme.

Проверка: `flutter analyze` без ошибок.

---

## ШАГ 2 — НАВИГАЦИЯ И SHELL

**`lib/core/router/app_router.dart`** с GoRouter:

```dart
// Маршруты:
// /onboarding          — если первый запуск (нет токена)
// /login               — экран входа через Telegram
// /                    — ShellRoute с BottomNav
//   /home              — главный экран (приведение)
//   /stats             — статистика
//   /profile           — профиль и тарифы
// /plans               — выбор тарифа
// /payment/:tier       — оплата через ЮKassa WebView
// /settings            — настройки

// Редирект: если нет токена в SecureStorage → /onboarding
// Если токен есть → /home
```

**`lib/shared/widgets/lumi_bottom_nav.dart`:**
```dart
// Кастомный BottomNavigationBar
// 3 иконки: главная (круг с точкой), статистика (3 столбца), профиль (человек)
// Активная иконка: #00E5FF + точка под иконкой
// Неактивная: opacity 0.28
// Фон: тёмная тема → darkBg, граница сверху 1px rgba(255,255,255,0.05)
// Светлая тема → lightBg, граница сверху 1px rgba(0,0,0,0.06)
// Анимация переключения: ScaleTransition 0.95→1.0 за 150ms
```

---

## ШАГ 3 — ПРИВЕДЕНИЕ (GHOST WIDGET)

Это самый важный компонент. Реализуй полностью.

**`lib/features/home/presentation/ghost_widget.dart`:**

```dart
class GhostWidget extends StatefulWidget {
  final bool isConnected;
  final VoidCallback onTap;
  const GhostWidget({required this.isConnected, required this.onTap, super.key});
}

class _GhostWidgetState extends State<GhostWidget> with TickerProviderStateMixin {
  // 1. AnimationController floatController — период 3сек, repeat(reverse:true)
  //    Tween: 0.0 → -8.0 (смещение Y в пикселях) для состояния connected
  //    Tween: 0.0 → 3.0 с поворотом ±3° для состояния disconnected

  // 2. AnimationController blinkController — период 4сек
  //    TweenSequence: [0.3сек до 0.1, 0.3сек обратно, 3.4сек пауза]
  //    Только когда isConnected == true

  // 3. AnimationController zController — период 2.2сек, repeat
  //    Три Z-буквы с задержкой 0, 0.7сек, 1.4сек
  //    Только когда isConnected == false

  // При смене isConnected: остановить текущие анимации, запустить новые
  // _handleConnectedChange() в didUpdateWidget

  // Размер области: 140×140 (включая кольца свечения)
  // Приведение внутри: 80×92, центрировано
}
```

**`_GhostPainter extends CustomPainter`:**
```dart
// Рисует тело приведения через Path():
//   moveTo(40, 4)
//   cubicTo для скруглённой головы
//   lineTo вниз до 72
//   три волны низа (quadraticBezierTo)
//   замкнуть path

// RadialGradient для заливки тела
// Отдельный paint для обводки
// Глаза: drawOval (тело) + drawCircle (блик)
// Улыбка: Path с quadraticBezierTo
// Блик на голове: drawOval с transform

// isConnected=false: другие цвета, закрытые глаза (arc), opacity 0.55
```

**Кольца свечения (только connected):**
```dart
// Stack: CustomPaint для колец + Transform.translate для парения
// Два концентрических AnimatedContainer с border
// opacity: 0.18 и 0.08, color: cyberBlue
// Радиусы: 38 и 52 от центра приведения
```

**Z-буквы (только disconnected):**
```dart
// Stack с тремя Positioned Text('z') виджетами
// Каждый: FadeTransition + SlideTransition (снизу вверх + вправо)
// fontSize: 11, 9, 7 (уменьшаются)
// Цвет: rgba(140, 200, 230, 0.7)
// Позиция: правее и выше головы приведения
```

**Обёртка:**
```dart
GestureDetector(
  onTap: () {
    HapticFeedback.mediumImpact();
    widget.onTap();
  },
  child: SizedBox(140, 140, child: ghostStack),
)
```

---

## ШАГ 4 — ГЛАВНЫЙ ЭКРАН

**`lib/features/home/presentation/home_screen.dart`:**

Структура (Column внутри SafeArea):
```
[Padding top:18]
  Row: "LUMI" (left) + иконка настроек (right, opacity 0.3)
[SizedBox height:20]
  GlassCard (главная карточка):
    [Padding: 28 top, 24 sides, 24 bottom]
    Column:
      StatusDot + StatusText  (Row, centred)
      [SizedBox: 20]
      GhostWidget (140×140, centred)
      [SizedBox: 20]
      StatusText ("Подключено" / "Отключено" / "Подключение...")
      [SizedBox: 20]
      ServerPill
      [SizedBox: 10]
      SubscriptionBadge

[SizedBox: height: заполнить пространство через Spacer]
LumiBottomNav
```

**`lib/features/home/presentation/server_pill.dart`:**
```dart
// Стеклянная пилюля: флаг + название сервера + субтитр + пинг
// Нажатие → bottomSheet со списком серверов
// Доступные серверы: EU (Нидерланды), USA (Нью-Йорк), CDN (Резерв)
// Premium может выбирать любой, Free — только EU
// Пинг обновляется раз в 30сек через Timer.periodic
```

**`lib/features/home/presentation/subscription_badge.dart`:**
```dart
// Строка: "Тариф" слева + "Premium · 23 дня" справа
// Фон: cyberBlue.withOpacity(0.06)
// Бордер: cyberBlue.withOpacity(0.15)
// Нажатие → навигация на /plans
// Free тариф: показать "Улучшить до Premium →"
```

**Состояния VPN в UI:**
```dart
enum VpnUiState { disconnected, connecting, connected, disconnecting, error }

// disconnected → приведение спит, статус серый
// connecting   → приведение начинает медленно просыпаться (переход 1сек)
//                CircularProgressIndicator.adaptive() рядом со статусом
// connected    → приведение парит, неоновый свет
// error        → приведение грустное (опущенный рот), статус красный
//                Toast с текстом ошибки
```

---

## ШАГ 5 — VPN ИНТЕГРАЦИЯ (sing-box)

**`lib/features/home/data/vpn_repository.dart`:**

```dart
class VpnRepository {
  // Использует flutter_v2ray или аналог

  // 1. loadConfig() — скачивает subscription URL с нашего API
  //    GET https://api.lumi-vpn.ru/sub/{sub_token}
  //    Парсит base64 → список VLESS конфигов
  //    Сохраняет в SecureStorage

  // 2. connect(ServerNode node) — подключается к выбранной ноде
  //    Выбирает конфиг по region (eu/usa/cdn)
  //    Передаёт в VPN движок

  // 3. disconnect() — отключает VPN

  // 4. getStatus() → Stream<VpnStatus>

  // 5. testPing(String host) → Future<int> (milliseconds)
  //    Использует socket connection для измерения пинга

  // Routing правила (из GET /api/v1/routing/lists):
  //   direct домены → не через VPN (НИКОГДА через CDN)
  //   proxy_eu → EU-нода
  //   proxy_usa → USA-нода
  //   остальное → прямо
}
```

**sing-box конфиг (генерируется программно):**
```json
{
  "log": { "level": "warn" },
  "dns": {
    "servers": [
      { "tag": "cloudflare", "address": "https://1.1.1.1/dns-query", "detour": "proxy" },
      { "tag": "local", "address": "local", "detour": "direct" }
    ],
    "rules": [
      { "geosite": "ru", "server": "local" }
    ]
  },
  "inbounds": [
    { "type": "tun", "inet4_address": "172.19.0.1/30",
      "auto_route": true, "strict_route": false,
      "sniff": true, "sniff_override_destination": false }
  ],
  "outbounds": [
    { "type": "vless", "tag": "proxy",
      "server": "NODE_IP", "server_port": 443,
      "uuid": "USER_UUID",
      "flow": "xtls-rprx-vision",
      "tls": { "enabled": true, "server_name": "microsoft.com",
               "reality": { "enabled": true, "public_key": "PUBLIC_KEY",
                            "short_id": "SHORT_ID" } } },
    { "type": "direct", "tag": "direct" },
    { "type": "block", "tag": "block" }
  ],
  "route": {
    "rules": [
      { "geosite": "ru", "outbound": "direct" },
      { "geoip": "ru", "outbound": "direct" },
      { "geosite": "category-ads-all", "outbound": "block" }
    ],
    "auto_detect_interface": true
  }
}
```

**Android AndroidManifest.xml:**
```xml
<uses-permission android:name="android.permission.INTERNET"/>
<uses-permission android:name="android.permission.FOREGROUND_SERVICE"/>
<uses-permission android:name="android.permission.RECEIVE_BOOT_COMPLETED"/>

<service android:name=".VpnService"
         android:permission="android.permission.BIND_VPN_SERVICE"
         android:exported="false">
  <intent-filter>
    <action android:name="android.net.VpnService"/>
  </intent-filter>
</service>
```

---

## ШАГ 6 — АВТОРИЗАЦИЯ И ОНБОРДИНГ

**`lib/features/auth/presentation/onboarding_screen.dart`:**

Три экрана через PageView:
```
Экран 1: приведение (большой, парит) + "Добро пожаловать в Lumi" + "Свобода интернета в одно касание"
Экран 2: иконка щита + "Работает везде" + "Wi-Fi, мобильный интернет, белые списки операторов"
Экран 3: иконка молнии + "Быстро и просто" + кнопка "Войти через Telegram →"
```

Индикаторы страниц: три точки, активная = cyberBlue.

**`lib/features/auth/presentation/login_screen.dart`:**

```dart
// Telegram Mini App авторизация:
// 1. Открыть InAppWebView с URL:
//    https://oauth.telegram.org/auth?bot_id=BOT_ID&origin=https://lumi-vpn.ru&embed=1
// 2. Слушать сообщения от WebView (onMessageReceived)
// 3. При получении auth_result → POST /api/v1/auth/tg с initData
// 4. Сохранить access_token и sub_token в SecureStorage
// 5. Навигация на /home

// Альтернатива (проще): url_launcher открывает Telegram бот
// Бот отправляет deep link lumi://auth?token=XXX
// Приложение перехватывает через uni_links
```

**`lib/core/storage/secure_storage.dart`:**
```dart
class AppStorage {
  final _storage = const FlutterSecureStorage();

  Future<void> saveTokens({required String access, required String sub});
  Future<String?> getAccessToken();
  Future<String?> getSubToken();
  Future<void> clear(); // logout
  Future<bool> isLoggedIn();
}
```

---

## ШАГ 7 — ОСТАЛЬНЫЕ ЭКРАНЫ

**`lib/features/stats/presentation/stats_screen.dart`:**
```dart
// Простой экран — GlassCard с:
// - Время подключения сегодня (hh:mm:ss, обновляется каждую секунду если connected)
// - Трафик: входящий и исходящий (в МБ, берётся из VPN движка)
// - Серверов доступно: 3
// - Заглушка на графики (placeholder GlassCard с текстом "Скоро")
```

**`lib/features/profile/presentation/profile_screen.dart`:**
```dart
// Аватар: круг с первой буквой Telegram username
// Username, ID
// Текущий тариф + дней осталось
// Кнопка "Улучшить план" (→ /plans) — CyberButton с неоном
// Кнопка "Поддержка" → Telegram @lumi_support
// Кнопка "Настройки" → /settings
// Кнопка "Выйти" (destructive, красный)
```

**`lib/features/profile/presentation/plans_screen.dart`:**
```dart
// Три GlassCard карточки: Free, Basic (149₽/мес), Premium (299₽/мес)
// Активный план: border cyberBlue opacity 0.4
// Каждая карточка: название + цена + список фич (✓ или ✗)
// Кнопка "Выбрать" → POST /api/v1/payment/create → WebView ЮKassa
// Квартальные варианты: переключатель месяц/квартал

// Фичи тарифов:
// Free:    1 Мбит/с, 1 устройство, EU-нода
// Basic:   10 Мбит/с, 2 устройства, EU + USA
// Premium: без лимитов, 5 устройств, EU + USA + CDN (резерв)
```

**`lib/features/settings/presentation/settings_screen.dart`:**
```dart
// Список настроек в GlassCard:
// - Тема: Light / Dark / Auto (SegmentedButton)
// - Язык: Русский (пока только он)
// - Автозапуск VPN (Switch) — при запуске приложения
// - Kill Switch (Switch) — блокировать интернет если VPN упал
// - Версия приложения (из package_info_plus)
// - Политика конфиденциальности (url_launcher)
```

---

## ШАГ 8 — СБОРКА И ФИНАЛИЗАЦИЯ

**`android/app/build.gradle`:**
```groovy
android {
    compileSdk 34
    defaultConfig {
        applicationId "ru.lumivpn.app"
        minSdk 21
        targetSdk 34
        versionCode 1
        versionName "1.0.0"
    }
    buildTypes {
        release {
            signingConfig signingConfigs.debug // для тестовой сборки
            minifyEnabled false
        }
    }
}
```

**`lib/core/network/api_client.dart`:**
```dart
// Dio с BaseOptions:
//   baseUrl: 'https://api.lumi-vpn.ru'
//   connectTimeout: Duration(seconds: 10)
//   receiveTimeout: Duration(seconds: 30)
//
// Interceptors:
//   1. AuthInterceptor — добавляет Authorization: Bearer {token}
//      При 401 → очистить токен → навигация на /login
//   2. PrettyDioLogger (только debug mode)
//
// Методы:
//   GET /sub/{token}                    → subscription URL
//   GET /api/v1/routing/lists           → routing rules
//   GET /api/v1/me/subscription         → текущий тариф
//   POST /api/v1/payment/create         → создать платёж
//   GET /api/v1/nodes/best?region=auto  → лучшая нода
```

**Финальные проверки:**
```bash
flutter analyze                    # 0 ошибок, 0 warnings
flutter build apk --debug          # успешная сборка APK
flutter build apk --release        # успешная release сборка

# Проверить вручную:
# - Приведение парит в тёмной теме
# - Приведение спит с Z в тёмной теме
# - Светлая тема корректно применяется
# - Нижняя навигация переключает экраны
# - На экране профиля видны тарифы
```

**`README.md`:**
```markdown
# Lumi VPN

Flutter-приложение VPN-сервиса Lumi для Android и iOS.

## Требования
- Flutter 3.16+
- Dart 3.0+
- Android SDK 21+ / iOS 13+

## Сборка
flutter pub get
flutter run

## Архитектура
Feature-based структура + Riverpod + GoRouter
VPN-ядро: sing-box (MPL 2.0)
Бэкенд: https://api.lumi-vpn.ru

## Публикация
Android: RuStore (ru.lumivpn.app)
iOS: App Store (после получения Network Extension entitlement)
```

---

## ПОСЛЕ ЗАВЕРШЕНИЯ

Отчитайся по следующим пунктам:
1. Список всех созданных файлов
2. Результат `flutter analyze` (должно быть 0 ошибок)
3. Размер debug APK
4. Какие части требуют реальных данных для тестирования (токены, IP нод)
5. Что нужно сделать перед публикацией в RuStore