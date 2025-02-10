package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"03_GoLoadTester/internal/loadtester"
)

func main() {
	// Определяем аргументы командной строки
	url := flag.String("url", "", "URL для тестирования")
	rps := flag.Int("rps", 10, "Количество запросов в секунду (если не указан, можно задать через бенчмарк)")
	duration := flag.Int("duration", 10, "Длительность теста в секундах")
	benchmark := flag.Bool("benchmark", false, "Выполнить бенчмарк для определения максимального RPS перед тестом")
	benchDuration := flag.Int("benchDuration", 5, "Длительность бенчмарка в секундах")
	// Флаг для указания локальных адресов (через запятую, например: "192.168.1.100,192.168.1.101")
	localAddrsFlag := flag.String("localAddrs", "", "Локальные адреса для отправки запросов, разделённые запятой")
	flag.Parse()

	if *url == "" {
		log.Fatal("Параметр -url обязателен")
	}

	var localAddrs []string
	if *localAddrsFlag != "" {
		localAddrs = strings.Split(*localAddrsFlag, ",")
	}

	// Если включён бенчмарк, выполняем его и устанавливаем rps в значение, определённое бенчмарком
	if *benchmark {
		maxRPS := loadtester.BenchmarkMaxRequests(*url, *benchDuration, localAddrs)
		fmt.Printf("Максимально возможный RPS на этой машине: %d\n", maxRPS)
		*rps = maxRPS
	}

	loadtester.StartTest(*url, *rps, *duration, localAddrs)
}
