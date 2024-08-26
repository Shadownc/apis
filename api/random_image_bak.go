package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"lmyself-api/utils"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// 返回结果太慢了 换一个方式
var (
	imageData     []string
	currentIndex  int
	mutex         sync.Mutex
	lastFetchTime time.Time
)

type Response struct {
	Code int    `json:"code"`
	URL  string `json:"url"`
}

func init() {
	// 使用公共函数加载环境变量
	utils.LoadEnvFiles()
}

func fetchNewData() error {
	log.Println("Fetching new data...")

	// 构建 URL 对象
	baseURL := "https://capi.lmyself.cloudns.be/list"
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("Invalid base URL: %v", err)
	}

	// 添加查询参数
	params := url.Values{}
	params.Add("sorting", "random")
	params.Add("categories", "010")
	params.Add("purity", "101")
	params.Add("resolutions", "1920x1080")

	// 从环境变量中读取 API 密钥，并在存在时添加到查询参数
	apiKey := os.Getenv("IMG_APIKEY")
	if apiKey != "" {
		params.Add("apikey", apiKey)
	}

	// 将查询参数附加到 URL
	u.RawQuery = params.Encode()

	// 打印最终的 URL 以供调试
	log.Printf("Final API URL: %s", u.String())

	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout: 30 * time.Second,
	}

	client := &http.Client{
		Timeout:   60 * time.Second,
		Transport: tr,
	}

	var resp *http.Response
	for retries := 0; retries < 5; retries++ {
		log.Printf("Attempt %d to fetch data", retries+1)
		resp, err = client.Get(u.String())
		if err == nil {
			break
		}
		log.Printf("Error fetching data: %v. Retrying...", err)
		time.Sleep(time.Second * time.Duration(retries+1))
	}
	if err != nil {
		log.Printf("Failed to fetch data after 5 attempts: %v", err)
		return err
	}
	defer resp.Body.Close()

	log.Println("Reading response body...")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return err
	}

	log.Printf("Response body length: %d bytes", len(body))

	log.Println("Unmarshaling JSON...")
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return err
	}

	log.Println("Extracting image data...")
	imageList, ok := data["data"].([]interface{})
	if !ok {
		log.Printf("Invalid data format. 'data' is not an array.")
		return fmt.Errorf("invalid data format")
	}

	log.Printf("Number of items in data array: %d", len(imageList))

	mutex.Lock()
	defer mutex.Unlock()

	imageData = []string{}
	for i, item := range imageList {
		log.Printf("Processing item %d", i)
		imgMap, ok := item.(map[string]interface{})
		if !ok {
			log.Printf("Item %d is not a map", i)
			continue
		}
		path, ok := imgMap["path"].(string)
		if !ok {
			log.Printf("Item %d does not have a valid 'path' field", i)
			continue
		}
		imageData = append(imageData, path)
		log.Printf("Added path: %s", path)
	}
	currentIndex = 0
	lastFetchTime = time.Now()

	log.Printf("Fetched %d images", len(imageData))
	return nil
}

func getNextImage() (string, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(imageData) == 0 || currentIndex >= len(imageData) {
		log.Println("No images available or all images used. Fetching new data...")
		mutex.Unlock()
		if err := fetchNewData(); err != nil {
			mutex.Lock()
			log.Printf("Error fetching new data: %v", err)
			return "", err
		}
		mutex.Lock()
	}

	if len(imageData) == 0 {
		log.Println("No images available after fetch attempt")
		return "", fmt.Errorf("no images available")
	}

	image := imageData[currentIndex]
	currentIndex++
	log.Printf("Returning image %d: %s", currentIndex, image)
	return image, nil
}

func RandomImageHandlerBak(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for random image")

	imageURL, err := getNextImage()
	if err != nil {
		log.Printf("Error getting image: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get image: %v. Current image data length: %d", err, len(imageData)), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("type") == "json" {
		response := Response{
			Code: 200,
			URL:  imageURL,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		} else {
			log.Println("Sent JSON response")
		}
	} else {
		// Modify the image URL to use the new endpoint
		capiURL := fmt.Sprintf("https://capi.lmyself.cloudns.be/img?path=%s", url.QueryEscape(imageURL))
		log.Printf("Fetching image from: %s", capiURL)

		tr := &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: 30 * time.Second,
			// Keep-Alive 设置可以重用 TCP 连接
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 10,
		}

		client := &http.Client{
			Timeout:   60 * time.Second, // 增加超时时间到 60 秒
			Transport: tr,
		}

		resp, err := client.Get(capiURL)
		if err != nil {
			log.Printf("Error fetching image: %v", err)
			http.Error(w, "Failed to fetch image", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error fetching image, status code: %d", resp.StatusCode)
			http.Error(w, "Failed to fetch image", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			log.Printf("Error copying image content: %v", err)
			http.Error(w, "Failed to send image", http.StatusInternalServerError)
			return
		}

		log.Println("Sent image content")

	}
}
