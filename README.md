ФИО: Чебряков Данила Сергеевич
Группа: БПМИ216
Вариант: Трекер

# REST API Service

## Требования

Для работы сервиса требуется установленное на машине:
- Docker
- Docker Compose

## Запуск сервиса

Чтобы запустить сервис, выполните следующие шаги:

1. Склонируйте репозиторий с кодом сервиса:

git clone https://github.com/Danatelllo/HSE-SOA.git

2. Перейдите на ветку с сервисом авторизации:

git checkout rest-api-server

3. Выполните:

в папке ~/soa 

docker-compose build
docker-compose up 

4. В браузере перейдите по ссылке:

http://localhost:8081/swagger/index.html#/default
