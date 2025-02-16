# Используем официальный образ Golang
FROM golang:1.22

# Устанавливаем рабочую директорию
WORKDIR /go/src/avito-shop

# Копируем все файлы в контейнер
COPY . .

# Сборка приложения
RUN go build -o /build ./cmd \
    && go clean -cache -modcache

# Открываем порт 8080
EXPOSE 8080

# Команда для запуска приложения
CMD ["/build"] 