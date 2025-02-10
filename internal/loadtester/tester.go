package loadtester

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// createHTTPClient создаёт HTTP-клиента, привязанного к заданному локальному адресу.
// Если в строке адреса отсутствует порт, автоматически добавляется ":0" для использования динамического порта.
func createHTTPClient(localAddrStr string) (*http.Client, error) {
	// Если порт не указан, добавляем ":0" для выбора случайного свободного порта.
	if !strings.Contains(localAddrStr, ":") {
		localAddrStr = localAddrStr + ":0"
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", localAddrStr)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{
		LocalAddr: tcpAddr,
		Timeout:   30 * time.Second,
	}
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	return client, nil
}

// BenchmarkMaxRequests выполняет бенчмарк, чтобы определить максимальное количество запросов в секунду,
// которое может выполнить ПК на указанном URL за период benchmarkDuration (в секундах).
// Если срез localAddrs не пустой, запросы выполняются через клиентов, привязанных к указанным адресам.
func BenchmarkMaxRequests(url string, benchmarkDuration int, localAddrs []string) int {
	fmt.Printf("Запуск бенчмарка для определения максимального RPS на %s в течение %d секунд...\n", url, benchmarkDuration)

	var count int64
	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Формируем пул HTTP-клиентов.
	var clients []*http.Client
	if len(localAddrs) > 0 {
		for _, addr := range localAddrs {
			client, err := createHTTPClient(addr)
			if err != nil {
				log.Printf("Ошибка создания HTTP-клиента для %s: %v", addr, err)
				continue
			}
			clients = append(clients, client)
		}
	}
	if len(clients) == 0 {
		clients = append(clients, http.DefaultClient)
	}

	var clientCounter int32 = 0
	numWorkers := runtime.NumCPU() * 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					// Выбираем клиента в циклическом порядке
					index := int(atomic.AddInt32(&clientCounter, 1) % int32(len(clients)))
					client := clients[index]
					resp, err := client.Get(url)
					if err != nil {
						log.Printf("Ошибка запроса в бенчмарке: %v", err)
					} else {
						resp.Body.Close()
					}
					atomic.AddInt64(&count, 1)
				}
			}
		}()
	}

	time.Sleep(time.Duration(benchmarkDuration) * time.Second)
	close(stop)
	wg.Wait()

	rps := int(count) / benchmarkDuration
	fmt.Printf("Бенчмарк завершён: достигнуто %d запросов в секунду\n", rps)
	return rps
}

// StartTest запускает нагрузочное тестирование на заданном URL с указанной частотой запросов (rps)
// и длительностью (duration) теста. Если срез localAddrs не пустой, запросы отправляются через клиентов,
// привязанных к соответствующим локальным адресам (используется схема round-robin).
func StartTest(url string, rps int, duration int, localAddrs []string) {
	fmt.Printf("Начало теста на %s: %d запросов в секунду, длительность %d секунд\n", url, rps, duration)
	var wg sync.WaitGroup

	// Формируем пул HTTP-клиентов.
	var clients []*http.Client
	if len(localAddrs) > 0 {
		for _, addr := range localAddrs {
			client, err := createHTTPClient(addr)
			if err != nil {
				log.Printf("Ошибка создания HTTP-клиента для %s: %v", addr, err)
				continue
			}
			clients = append(clients, client)
		}
	}
	if len(clients) == 0 {
		clients = append(clients, http.DefaultClient)
	}

	var clientCounter int32 = 0
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	quit := make(chan bool)

	go func() {
		time.Sleep(time.Duration(duration) * time.Second)
		quit <- true
	}()

	for {
		select {
		case <-quit:
			ticker.Stop()
			wg.Wait()
			fmt.Println("Тест завершён")
			return
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				index := int(atomic.AddInt32(&clientCounter, 1) % int32(len(clients)))
				client := clients[index]
				resp, err := client.Get(url)
				if err != nil {
					log.Printf("Ошибка запроса: %v", err)
					return
				}
				resp.Body.Close()
				log.Printf("Код ответа: %d", resp.StatusCode)
			}()
		}
	}
}
