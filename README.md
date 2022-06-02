# downloader
Небольшая утилитка для получения потокового видео по HTTP с целью имитации нагрузки.

Основной концепт такой - есть условно "сервер", который получает через Unix-сокет или TCP-порт команды от клиентов. Клиент может передавать следующие команды:
* добавить поток для выгрузки (с указанием URL);
* получить статистику по текущим задачам выгрузки;
* остановить все задачи выгрузки.

## Интерфейс командной строки

### Запуск сервера

```shell
./downloader server [-timeount=<timeout>] [-endpoint=<endpoint>]
```

* `timeout` - таймаут на операции подключения и чтения из сокета при выгрузке потока;
* `endpoint` - какой адрес забиндить. Указывается через URL. Например: `unix:///tmp/socket.sock` - для Unix-сокета, `tcp://127.0.0.1:1100` - для TCP.

По умолчанию (если не указан `endpoint`) сервер создает Unix-сокет по пути `/tmp/downloader.sock` и слушает клиентские команды.

### Добавить задачу к выгрузке

```shell
./downloader task <URL> [-endpoint=<endpoint>]
```

* `URL` - ссылка на поток видео;
* `endpoint` - адрес сервера.

### Получить состояние сервера

```shell
./downloader status [-endpoint=<endpoint>]
```

* `endpoint` - адрес сервера.

В ответ вернется информация о кол-ве активных задач выгрузки, задач, которые завершились с ошибкой или еще не стартовали.

Пример:

```
2022/05/12 19:26:07 [Server] ok [Active: 3, Failed: 1, Pending: 0]
```

### Остановить все задачи

```shell
./downloader stop [-endpoint=<endpoint>]
```

* `endpoint` - адрес сервера.

### Остановить сервер

```shell
./downloader done [-endpoint=<endpoint>]
```

* `endpoint` - адрес сервера.