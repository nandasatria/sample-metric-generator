package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/joho/godotenv"
)

type ServerConfig struct {
	ID        string
	Hostname  string
	IPAddress string
	Location  struct {
		Country   string
		City      string
		Latitude  float64
		Longitude float64
	}
}

type MetricData struct {
	Timestamp   time.Time `json:"@timestamp"`
	ServerID    string    `json:"server_id"`
	Hostname    string    `json:"hostname"`
	IPAddress   string    `json:"ip_address"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
}

type MetricGenerator struct {
	servers       []ServerConfig
	esClient      *elasticsearch.Client
	metricTracker map[string]MetricData
	esIndex       string
	rnd           *rand.Rand // Add a local random number generator
	mu            sync.Mutex
}

func loadConfiguration() (int, string, string, string, string) {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found")
	}

	// Get environment variables
	serverCount, _ := strconv.Atoi(os.Getenv("SERVER_COUNT"))
	if serverCount == 0 {
		serverCount = 100
	}

	esServer := os.Getenv("ES_SERVER")
	if esServer == "" {
		esServer = "http://localhost:9200"
	}

	esUsername := os.Getenv("ES_USERNAME")
	esPassword := os.Getenv("ES_PASSWORD")
	esIndex := os.Getenv("ES_INDEX")
	if esIndex == "" {
		esIndex = "server-metrics"
	}

	return serverCount, esServer, esUsername, esPassword, esIndex
}

func generateRandomServers(count int, rnd *rand.Rand) []ServerConfig {
	locations := []struct {
		Country   string
		City      string
		Latitude  float64
		Longitude float64
	}{
		{"United States", "New York", 40.7128, -74.0060},
		{"United States", "Los Angeles", 34.0522, -118.2437},
		{"United Kingdom", "London", 51.5074, -0.1278},
		{"Germany", "Berlin", 52.5200, 13.4050},
		{"Japan", "Tokyo", 35.6762, 139.6503},
	}

	servers := make([]ServerConfig, count)
	for i := 0; i < count; i++ {
		loc := locations[rnd.Intn(len(locations))]

		servers[i] = ServerConfig{
			ID: fmt.Sprintf("server-%03d", i+1),
			Hostname: fmt.Sprintf("%s-host-%03d",
				[]string{"web", "db", "app", "cache", "worker"}[rnd.Intn(5)],
				i+1),
			IPAddress: fmt.Sprintf("10.%d.%d.%d",
				rnd.Intn(256),
				rnd.Intn(256),
				rnd.Intn(256)),
			Location: struct {
				Country   string
				City      string
				Latitude  float64
				Longitude float64
			}{
				Country:   loc.Country,
				City:      loc.City,
				Latitude:  loc.Latitude + (rnd.Float64()*0.5 - 0.25),
				Longitude: loc.Longitude + (rnd.Float64()*0.5 - 0.25),
			},
		}
	}

	return servers
}

func (mg *MetricGenerator) generateConsistentServerMetric(server ServerConfig) MetricData {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	prevMetric, exists := mg.metricTracker[server.ID]

	var cpuUsage, memoryUsage, diskUsage float64

	if exists {
		cpuBase := prevMetric.CPUUsage
		memBase := prevMetric.MemoryUsage
		diskBase := prevMetric.DiskUsage

		cpuUsage = math.Max(0, math.Min(100,
			cpuBase+(mg.rnd.Float64()*10-5)+
				math.Sin(float64(time.Now().Unix()/60))*5))

		memoryUsage = math.Max(0, math.Min(100,
			memBase+(mg.rnd.Float64()*8-4)+
				math.Cos(float64(time.Now().Unix()/120))*3))

		diskUsage = math.Max(0, math.Min(100,
			diskBase+(mg.rnd.Float64()*6-3)+
				math.Tan(float64(time.Now().Unix()/180))*2))
	} else {
		cpuUsage = 10 + mg.rnd.Float64()*40
		memoryUsage = 20 + mg.rnd.Float64()*50
		diskUsage = 5 + mg.rnd.Float64()*30
	}

	metric := MetricData{
		Timestamp:   time.Now().UTC(),
		ServerID:    server.ID,
		Hostname:    server.Hostname,
		IPAddress:   server.IPAddress,
		Country:     server.Location.Country,
		City:        server.Location.City,
		Latitude:    server.Location.Latitude,
		Longitude:   server.Location.Longitude,
		CPUUsage:    roundFloat(cpuUsage, 2),
		MemoryUsage: roundFloat(memoryUsage, 2),
		DiskUsage:   roundFloat(diskUsage, 2),
	}

	mg.metricTracker[server.ID] = metric
	return metric
}

func (mg *MetricGenerator) sendMetricToElasticsearch(metric MetricData) {
	jsonMetric, err := json.Marshal(metric)
	if err != nil {
		log.Printf("Error marshaling metric: %v", err)
		return
	}

	req := esapi.IndexRequest{
		Index:      mg.esIndex,
		DocumentID: fmt.Sprintf("%s-%d", metric.ServerID, time.Now().Unix()),
		Body:       bytes.NewReader(jsonMetric),
	}

	_, err = req.Do(context.Background(), mg.esClient)
	if err != nil {
		log.Printf("Error indexing metric: %v", err)
	}
}

func (mg *MetricGenerator) GenerateConsistentMetrics() {
	for {
		var wg sync.WaitGroup

		for _, server := range mg.servers {
			wg.Add(1)
			go func(srv ServerConfig) {
				defer wg.Done()

				metric := mg.generateConsistentServerMetric(srv)
				mg.sendMetricToElasticsearch(metric)
			}(server)
		}

		wg.Wait()
		time.Sleep(1 * time.Minute)
	}
}

func main() {
	// Load configuration
	serverCount, esServer, esUsername, esPassword, esIndex := loadConfiguration()

	// Create a new random number generator seeded with the current time
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate random servers
	servers := generateRandomServers(serverCount, rnd)

	// Configure Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{esServer},
		Username:  esUsername,
		Password:  esPassword,
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %v", err)
	}

	// Create metric generator
	generator := &MetricGenerator{
		servers:       servers,
		esClient:      esClient,
		metricTracker: make(map[string]MetricData),
		esIndex:       esIndex,
		rnd:           rnd, // Set the local random number generator
	}

	// Run metric generation
	// log.Printf("metric: %v\n ", servers)
	generator.GenerateConsistentMetrics()
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
