# API-тесты для проекта https://testboard.avito.com/

## Полезные ссылки

1. [Swagger-спецификация](https://testboard.avito.com/swagger/index.html)
2. [Техническое задание](https://docs.google.com/document/d/1IZ9SxEcePoQT6lyEmukycp1qNok4HZtok73UQnPwKSA/edit?usp=sharing)
3. [Библиотека для написания API-тестов](https://github.com/steinfletcher/apitest)
4. [Мощная библиотека анализа JSON для Go](https://github.com/tidwall/gjson)

## Подготовка окружения

1. Установите любое IDE для разработки: [JetBrains GoLand](https://www.jetbrains.com/ru-ru/go/) (платное или по студенческой лицензии), [VS Code](https://code.visualstudio.com/) (бесплатное).
2. Установите [Git](https://git-scm.com/install/).
3. [Установите](https://go.dev/doc/install) Go версии 1.25.1.
4. [Настройте](https://docs.github.com/ru/authentication/connecting-to-github-with-ssh) доступ к своему GitHub-аккаунту по SSH-ключу.
5. Склонируйте репозиторий: `git clone <путь к вашему репозиторию>`.
6. Подтяните зависимости в корне проекта:

```bash
go mod tidy
```

## Переменные окружения

Значения задаются в файле `.env.override` в корне проекта (создайте его, если нет).
Файл с реальными данными в git не коммитится.

| Переменная | Обязательна | Назначение |
|---|---|---|
| `API_URL` | да | Адрес стенда |
| `TEST_LOGIN` | да | Основной пользователь: у него проверяются избранное и его объявления |
| `TEST_PASSWORD` | да | Пароль основного пользователя |
| `TEST_SECOND_LOGIN` | для пакета `favorites` | Второй пользователь: владелец объявлений, которые основной добавляет в избранное. **Должен отличаться от `TEST_LOGIN`** |
| `TEST_SECOND_PASSWORD` | для пакета `favorites` | Пароль второго пользователя |
| `DEBUG` | нет | `true` включает вывод запросов и ответов |

Пример `.env.override`:

```
API_URL=https://testboard.avito.com
TEST_LOGIN=your_login
TEST_PASSWORD=your_password
TEST_SECOND_LOGIN=your_second_login
TEST_SECOND_PASSWORD=your_second_password
```

Оба пользователя должны быть заранее зарегистрированы на стенде.
Тесты создают и удаляют объявления этих пользователей, поэтому не используйте рабочие учётные записи.

## Запуск тестов

Все тесты:

```bash
go test -v -p 1 -count=1 ./...
```

Один пакет:

```bash
go test -v -count=1 ./tests/scenarios/favorites/
```

Без долгих тестов (например, лимит избранного в 50 объявлений создаёт 50 объявлений):

```bash
go test -v -short -count=1 ./tests/scenarios/favorites/
```

Один тест:

```bash
go test -v -count=1 ./tests/scenarios/favorites/ -run TestSuiteRun/TestFavoritesFullCycle
```

Пояснения к флагам:

- `-p 1`: пакеты запускаются по очереди. Стенд общий и чувствителен к частоте запросов, а тестовые пользователи общие для пакетов.
- `-count=1`: отключает кеш результатов Go.
- `-short`: пропускает тесты, помеченные как долгие.

## Особенности стенда

- При большой частоте запросов nginx на стенде отвечает `503 Service Unavailable` с HTML-страницей вместо JSON. В тесте это выглядит как `invalid character '<' looking for beginning of value`.
- Если тесты падают сериями с такой ошибкой, не запускайте пакеты параллельно и уменьшите число лишних запросов в подготовке и очистке.
- Объявления с фото создаются дольше остальных запросов, поэтому тесты с большим числом объявлений идут заметно медленнее.

## Логирование

Для вывода request/response в консоль выставьте `DEBUG=true` в `.env.override`.

> Внимание: в лог попадает заголовок `Authorization` с токеном. Перед тем как делиться логом, замените токен на `Bearer ***`.

## Запуск линтера

1. Установите [golangci-lint](https://golangci-lint.run/docs/welcome/install/local/).
2. Запустите из корня репозитория:

```bash
golangci-lint run
```

## Если тесты не запускаются

| Ошибка | Причина |
|---|---|
| `Отсутствует логин пользователя` | Не заданы `TEST_LOGIN` или `TEST_SECOND_LOGIN` в `.env.override` |
| `API_URL не задан` | Не задан адрес стенда |
| `invalid character '<'` и статус 503 | Стенд отклонил запрос из-за частоты, см. «Особенности стенда» |