package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnvFiles 加载环境变量文件
func LoadEnvFiles() {
	// 读取环境变量来确定是否指定了特定的 .env 文件
	envFile := os.Getenv("ENV_FILE")

	if envFile != "" {
		// 如果设置了 ENV_FILE 环境变量，则只加载这个文件
		err := godotenv.Load(envFile)
		if err != nil {
			log.Fatalf("Error loading %s file: %v", envFile, err)
		}
		fmt.Printf("Loaded environment variables from %s\n", envFile)
	} else {
		// 如果有 .env.local 文件，优先加载它
		localEnvFile := ".env.local"
		if _, err := os.Stat(localEnvFile); err == nil {
			err := godotenv.Load(localEnvFile)
			if err != nil {
				log.Fatalf("Error loading %s file: %v", localEnvFile, err)
			}
			fmt.Printf("Loaded environment variables from %s\n", localEnvFile)
		} else {
			// 否则加载当前目录下所有的 .env 文件
			envFiles, err := filepath.Glob(".env*")
			if err != nil {
				log.Fatalf("Error finding .env files: %v", err)
			}

			if len(envFiles) == 0 {
				log.Fatalf("No .env files found in the current directory")
			}

			for _, file := range envFiles {
				// 排除已经尝试加载过的 .env.local 文件
				if file == localEnvFile {
					continue
				}
				err := godotenv.Load(file)
				if err != nil {
					fmt.Printf("Could not load %s, continuing...\n", file)
				} else {
					fmt.Printf("Loaded environment variables from %s\n", file)
				}
			}
		}
	}
}
