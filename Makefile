DOCKER_USER=italloguimaraes

# 1. Constrói e envia todas as imagens para o Docker Hub
push-all:
	@echo "--- Preparando Imagens para a UEFS ---"
	docker build --build-arg TARGET=broker -t $(DOCKER_USER)/broker-dc .
	docker push $(DOCKER_USER)/broker-dc

	docker build --build-arg TARGET=client -t $(DOCKER_USER)/client-dc .
	docker push $(DOCKER_USER)/client-dc

	docker build --build-arg TARGET=sensor -t $(DOCKER_USER)/sensor-dc .
	docker push $(DOCKER_USER)/sensor-dc

	docker build --build-arg TARGET=actuator -t $(DOCKER_USER)/actuator-dc .
	docker push $(DOCKER_USER)/actuator-dc

# 2. Limpa contêineres e imagens antigas para não lotar o PC do lab
clean:
	docker system prune -f