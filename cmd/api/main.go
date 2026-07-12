package main

import (
	"fmt"
	"log"
	"github.com/yugjain1212/crawliq/config"
)


func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	fmt.Println("Server Port: %d\n", cfg.server.port)
	fmt.Println("Database Host: %s\n", cfg.database.host)
	fmt.Println("Database Port:", cfg.database.port)
	fmt.Println("Database Username:", cfg.database.username)
	fmt.Println("Database Password:", cfg.database.password)
	fmt.Println("Database Name:", cfg.database.dbname)
	fmt.Println("Crawler Workers:", cfg.crawler.workers)
	fmt.Println("Crawler Timeout:", cfg.crawler.timeout)
	fmt.Println("Hello, CrawlIQ ")

}
