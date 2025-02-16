# Merch Store

Это проект интернет-магазина мерча, написанный на Go. Приложение предоставляет API для аутентификации пользователей, просмотра информации, перевода монет между пользователями и покупки товаров

## Технологии

- **Язык программирования:** Go 1.22
- **Веб-фреймворк:** [Gorilla Mux](https://github.com/gorilla/mux)
- **База данных:** PostgreSQL (используется [pgx](https://github.com/jackc/pgx))
- **Аутентификация:** JWT
- **Логирование:** [Uber Zap](https://github.com/uber-go/zap)
- **Конфигурация:** YAML (gopkg.in/yaml.v2)
- **Контейнеризация:** Docker и Docker Compose
- **Тестирование:** unit-тесты, e2e-тесты, мокирование (github.com/golang/mock, testify)

---

## Дерево файлов

 ```
├── cmd
│ └── main.go
├── cfg
│ ├── config.go
│ └── config.yaml
├── internal
│ ├── delivery
│ │ └── http
│ │ └── handler.go
│ ├── entity
│ │ └── employee.go
│ ├── repository
│ │ ├── employee.go
│ │ ├── transaction.go
│ │ └── mock_employee.go
│ └── usecase
│ └── employee.go
├── migrations
│ ├── 0001_create_employees_table.up.sql
│ ├── 0002_create_merch_table.up.sql
│ ├── 0003_create_inventory_table.up.sql
│ ├── 0004_create_transactions_table.up.sql
│ ├── 0001_create_employees_table.down.sql
│ ├── 0003_create_inventory_table.down.sql
│ └── 0004_create_transactions_table.down.sql
└── pkg
  ├── jwt
  │ └── jwt.go
  ├── logger
  │ └── logger.go
  └── utils
    └── utils.go
├── README.md
├── Makefile
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── .golangci.yaml
```

---

## Запуск и тестирование

### Сборка и запуск приложения
Для сборки и запуска контейнеров в проекте используется Makefile. Выполните следующие команды:

- **Сборка Docker-контейнеров:**
```bash
make build
```
- **Запуск контейнеров в фоновом режиме:**
```bash
make up
```
- **Остановка контейнеров:**
``` bash
make down
```

### Тестирование

Чтобы запустить тесты проекта, выполните:
``` bash
make test
```
Команда запустит все unit-тесты с подробными логами

---

## Что можно улучшить

- **Обработка ошибок:** Улучшить обработку ошибок с более подробным логированием и кастомизированными сообщениями
- **Безопасность:** Обеспечить хранение секретных ключей и конфигурационных файлов (например, jwtKey) с использованием переменных окружения или секретных хранилищ
- **Документация и комментарии:** Улучшить inline документацию для упрощения поддержки и развития проекта
- **CI/CD:** Настроить автоматическую сборку и тестирование через систему CI/CD для гарантии качества кода
