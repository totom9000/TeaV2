run:
	@KO_DATA_PATH=cmd/kodata/ SAAS_ADDRESS="http://localhost:8080" go run cmd/main.go

staging:
	@KO_DATA_PATH=cmd/kodata/ SAAS_ADDRESS="http://144.126.148.104:8080" go run cmd/main.go

build:
	@cd ../ && yarn && yarn build
	@cd ../ && rm -rf backend/cmd/kodata  &&  mv build kodata && mv kodata backend/cmd/kodata 

debug:
	@make build
	@make run

image:
	@docker build -t imaybeagod/tea . && docker push imaybeagod/tea