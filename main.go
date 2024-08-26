package main

import (
	"fmt"
	"lmyself-api/api"
	"lmyself-api/db"
	"lmyself-api/utils"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func init() {
	// 加载环境变量文件
	utils.LoadEnvFiles()

	// 从环境变量中读取代理地址
	proxyAddress := os.Getenv("HTTP_PROXY")
	if proxyAddress == "" {
		fmt.Println("No proxy address set in environment variable HTTP_PROXY")
		return
	}

	// 解析代理地址
	proxyURL, err := url.Parse(proxyAddress)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse proxy URL from environment variable: %v", err))
	}

	// 设置全局HTTP客户端和代理
	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// 可选：如果需要自定义超时等配置，可以设置全局的http.DefaultClient
	http.DefaultClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	fmt.Println("Proxy is set to:", proxyAddress)
}

func main() {
	// 加载环境变量文件
	utils.LoadEnvFiles()

	// 从环境变量读取数据库连接信息
	username := os.Getenv("DB_USERNAME")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")
	tablename := os.Getenv("TABLE_NAME")
	serverport := os.Getenv("SERVER_PORT")

	// 打印读取到的环境变量
	// fmt.Printf("DB_USERNAME: %s\n", username)
	// fmt.Printf("DB_PASSWORD: %s\n", password)
	// fmt.Printf("DB_HOST: %s\n", host)
	// fmt.Printf("DB_PORT: %s\n", port)
	// fmt.Printf("DB_NAME: %s\n", dbname)
	// fmt.Printf("TABLE_NAME: %s\n", tablename)

	// 连接数据库并创建表
	dbConn, err := db.ConnectAndCreateDBIfNotExist(username, password, host, port, dbname)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	err = db.CreateTableIfNotExists(dbConn, tablename)
	if err != nil {
		log.Fatal(err)
	}

	// 初始化数据库连接和表名
	api.InitDB(dbConn, tablename)

	// 初始化随机数种子
	// rand.Seed(time.Now().UnixNano())

	// 设置路由和处理函数
	http.HandleFunc("/random-image", api.RandomImageHandler)

	// 启动服务器
	port = os.Getenv("PORT")
	if port == "" {
		port = serverport
		if port == "" {
			port = "8080"
		}
	}

	fmt.Printf("Server is running on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
