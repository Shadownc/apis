package api

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	db         *sql.DB
	tableName  string
	apiKey     string
	httpClient *http.Client
)

func init() {
	apiKey = os.Getenv("IMG_APIKEY")

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		TLSHandshakeTimeout: 20 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		ForceAttemptHTTP2: true,
	}

	httpClient = &http.Client{
		Timeout:   120 * time.Second,
		Transport: transport,
	}
}

func InitDB(database *sql.DB, tblName string) {
	db = database
	tableName = tblName
}

func RandomImageHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Println("收到获取随机图片的请求")

	ctx, cancel := context.WithTimeout(r.Context(), 110*time.Second)
	defer cancel()

	baseURL := "https://capi.lmyself.cloudns.be/img"
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("无效的基础URL: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	params := url.Values{}
	params.Add("sorting", "random")
	params.Add("categories", "010")
	params.Add("purity", "101")
	params.Add("resolutions", "1920x1080")
	if apiKey != "" {
		params.Add("apikey", apiKey)
	}
	u.RawQuery = params.Encode()

	finalURL := u.String()
	log.Printf("最终的 API URL: %s", finalURL)

	var resp *http.Response
	var getErr error
	for retries := 0; retries < 3; retries++ {
		req, err := http.NewRequestWithContext(ctx, "GET", finalURL, nil)
		if err != nil {
			log.Printf("创建请求失败: %v", err)
			http.Error(w, "内部服务器错误", http.StatusInternalServerError)
			return
		}

		resp, getErr = httpClient.Do(req)
		if getErr == nil {
			break
		}
		log.Printf("尝试 %d: 获取图片时出错: %v", retries+1, getErr)
		time.Sleep(time.Duration(retries+1) * time.Second)
	}

	if getErr != nil {
		log.Printf("获取图片失败，所有重试均失败: %v", getErr)
		http.Error(w, "获取图片失败", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("获取图片失败，状态码: %d", resp.StatusCode)
		http.Error(w, "获取图片失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Println("无法使用流式响应")
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	buffer := make([]byte, 32*1024)
	var written int64
	for {
		nr, er := resp.Body.Read(buffer)
		if nr > 0 {
			nw, ew := w.Write(buffer[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				log.Printf("写入响应时出错: %v", ew)
				return
			}
			if nr != nw {
				log.Printf("写入的字节数不匹配，预期 %d，实际 %d", nr, nw)
			}
			flusher.Flush()
		}
		if er != nil {
			if er != io.EOF {
				log.Printf("读取响应体时出错: %v", er)
			}
			break
		}
	}

	log.Printf("成功发送图片内容，大小: %d bytes", written)
	log.Printf("请求处理时间: %v", time.Since(start))

	if err := SaveAPICallDetails(r.URL.String()); err != nil {
		log.Printf("保存 API 调用详情时出错: %v", err)
	}
}

func SaveAPICallDetails(requestURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the api_url already exists in the table
	var count int
	queryCheck := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE api_url = ?`, tableName)
	err := db.QueryRowContext(ctx, queryCheck, requestURL).Scan(&count)
	if err != nil {
		return fmt.Errorf("查询 API 调用详情失败: %w", err)
	}

	if count > 0 {
		// If the api_url exists, update the call_count
		queryUpdate := fmt.Sprintf(`
            UPDATE %s
            SET call_count = call_count + 1, updated_at = CURRENT_TIMESTAMP
            WHERE api_url = ?
        `, tableName)
		_, err := db.ExecContext(ctx, queryUpdate, requestURL)
		if err != nil {
			return fmt.Errorf("更新 API 调用详情失败: %w", err)
		}
		log.Printf("成功更新 API 调用次数，URL: %s", requestURL)
	} else {
		// If the api_url does not exist, insert a new record
		queryInsert := fmt.Sprintf(`
            INSERT INTO %s (api_url, call_count, updated_at)
            VALUES (?, 1, CURRENT_TIMESTAMP)
        `, tableName)
		_, err := db.ExecContext(ctx, queryInsert, requestURL)
		if err != nil {
			return fmt.Errorf("插入 API 调用详情失败: %w", err)
		}
		log.Printf("成功插入新的 API 调用记录，URL: %s", requestURL)
	}

	return nil
}
