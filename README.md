# Redis Analyzer

📊 CLI-инструмент для анализа ключей в Redis:
- Детектирует дубликаты по содержимому
- Считает статистику TTL (в т.ч. ключи без TTL)
- Генерирует отчёты `.csv`

## 🔧 Установка

### Через shell (Linux/macOS)

```bash
curl -sS https://raw.githubusercontent.com/alex3ax/redis-analyzer/main/scripts/install.sh | bash
```

или с помощью `wget`:

```bash
wget -q https://raw.githubusercontent.com/alex3ax/redis-analyzer/main/scripts/install.sh -O - | bash
```

> Установка скачает бинарник для вашей ОС из [GitHub Releases](https://github.com/alex3ax/redis-analyzer/releases) и положит в `/usr/local/bin`.

---

## 🚀 Использование

```bash
redis-analyzer --addr localhost:6379 --match "cache:*" --export report.csv
```

### Аргументы

| Флаг            | Описание                                           | Значение по умолчанию        |
|------------------|----------------------------------------------------|-------------------------------|
| `--addr`         | Адрес Redis                                        | `localhost:6379`              |
| `--password`     | Пароль Redis (если требуется)                      | `""`                          |
| `--db`           | Номер базы                                         | `0`                           |
| `--match`        | Шаблон ключей для сканирования                     | `"*"`                         |
| `--workers`      | Кол-во воркеров (параллельных сканеров)            | `5`                           |
| `--short-ttl`    | Порог для “коротких” TTL (в секундах)              | `86400` (сутки)               |
| `--export`       | Путь для экспорта CSV-отчёта по дубликатам         | `""` (только вывод в терминал)|
| `--tls`          | Enable TLS connection to Redis                     | `false`                       |

---

## 📦 Сборка вручную

Требуется установленный Go 1.21+.

### Локальная сборка:
```bash
make build
```

### Сборка под все платформы:
```bash
make release
```

Бинарники и архивы будут находиться в папке `build/`:
```
build/
├── redis-analyzer-linux-amd64.tar.gz
├── redis-analyzer-darwin-arm64.zip
...
```

---

## 📋 Пример отчёта `report.csv`

```csv
count,size_kb,sample
5,123.45,cache:product:latest
3,98.76,cache:variant:234
```

---

## 🔐 Примечания

- Инструмент безопасен для использования на продакшене (чтение только `SCAN`, `GET`, `TTL`)
- Не изменяет данные в Redis
- Может создавать нагрузку при большом объёме ключей — используйте `--match`, `--workers` осознанно

---

## 📄 License

MIT © [Alex "alex3ax" Zakharov](https://github.com/alex3ax)
