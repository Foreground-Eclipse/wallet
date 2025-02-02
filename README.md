# wallet
## Run Locally

Edit the local.env in wallet/config

Build with docker

```bash
docker compose up --build
```

Start the server locally

```bash
go run cmd/server/main.go
```
Or just run it in docker


## API Reference

#### Get wallet balance

```http
  GET /api/v1/wallets/{UUID}
```

#### Post operation

```http
  POST /api/v1/wallet
```

| Parameter | Type     | Description                       |
| :-------- | :------- | :-------------------------------- |
| `valletid`      | `string` | **Required**. Id of wallet |
| `operationType`      | `string DEPOSIT or WITHDRAW` | **Required**. Type of operation |
| `amount`      | `int` | **Required**. Amount to withdraw/deposit |
