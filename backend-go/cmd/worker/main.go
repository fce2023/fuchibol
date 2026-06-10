package main

import (
	"log"

	"fuchibol-backend-go/database"
	"fuchibol-backend-go/workers"
)

func main() {
	log.Println("Initializing Background Worker process...")

	// Inicializar la conexión a BD si los workers lo requieren
	database.ConnectDB()

	// Arrancar el consumidor de tareas (equivalente al demonio de Celery)
	workers.RunWorkerServer()
}
