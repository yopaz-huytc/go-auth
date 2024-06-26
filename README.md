# GO Auth Service
This is a simple authentication service that use OAuth2.0 to authenticate users. It is built using Golang and the Gin framework.

## Installation

1. Clone the repository
2. Run `go install github.com/cosmtrek/air@latest` to install the air package
3. Run `go mod tidy` to install the dependencies
4. Run `air` to start the server

## Docker Installation

1. Clone the repository
2. Run `docker-compose up --build` to start the server
3. The server will be running on `localhost:3333`
4. To stop the server, run `docker-compose down`
5. To remove the images, run `docker-compose down --rmi all`

## Usage
1. cp .env.example .env
2. config your .env file
3. `localhost:3333/login` to login to app, json request is: { "uid": "current_uid" } 
4. `localhost:3333/user-info` to test if you are logged in with jwt token
5. `localhost:3333/refresh` to refresh jwt token
6. `localhost:3333/logout` to logout
