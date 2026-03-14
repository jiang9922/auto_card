package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"

	//"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 模拟的响应结构 —— 完全匹配 sms8.net 格式
type MockResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data CodeData `json:"data"`
}

type CodeData struct {
	Code        string `json:"code"`
	CodeTime    string `json:"code_time"`    // 格式："2025-11-07 13:21:52"
	ExpiredDate string `json:"expired_date"` // 格式："2026-01-15 00:00:00"
}

func main() {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	// 自定义时间格式布局
	const displayFormat = "2006-01-02 15:04:05" // 对应 "2025-11-07 13:21:52"
	const displayDateOnly = "2006-01-02"        // 用于生成过期日期（年月日）

	// 固定一个“长期有效的日期”作为 expired_date（如 2026-01-15）
	expiredDateString := "2026-01-15 00:00:00"

	// 用于循环获取模板消息的索引
	templateIndex := 0
	var mutex sync.Mutex

	r.GET("/api/record", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusOK, MockResponse{
				Code: 0,
				Msg:  "Missing token",
				Data: CodeData{
					Code:        "",
					CodeTime:    "",
					ExpiredDate: expiredDateString,
				},
			})
			return
		}

		// 决定是否返回验证码（模拟：50% 概率返回）
		hasCode := rand.Intn(2) == 1 // 50% 成功率

		now := time.Now()
		codeTime := now.Format(displayFormat) // 当前时间字符串

		if hasCode {
			// ✅ 成功：生成带描述的验证码文本
			codeValue := fmt.Sprintf("%06d", rand.Intn(1000000)) // 6位验证码
			println(codeValue)

			// 按顺序循环选择一条模板消息
			messages := []string{
				"Your verification code is %s.",
				"%s is your verification code. Don't share it with anyone.",
				"%s 腾讯视频验证码。60秒内有效，请勿向任何人泄露。如非本人操作，请忽略本短信。",
			}
			mutex.Lock()
			template := messages[templateIndex]
			templateIndex = (templateIndex + 1) % len(messages)
			mutex.Unlock()

			fullCode := fmt.Sprintf(template, codeValue)
			print("模版-------Sending code: " + fullCode)

			c.JSON(http.StatusOK, MockResponse{
				Code: 1,
				Msg:  "ok",
				Data: CodeData{
					Code:        fullCode,
					CodeTime:    codeTime,
					ExpiredDate: expiredDateString,
				},
			})

		} else {
			// ❌ 失败：未获取到验证码
			c.JSON(http.StatusOK, MockResponse{
				Code: 0,
				Msg:  "No verification code",
				Data: CodeData{
					Code:        "",
					CodeTime:    "",
					ExpiredDate: expiredDateString,
				},
			})
		}
	})

	// 健康检查
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "✅ 模拟 sms8.net 接口已启动\n📌 示例:\n   /api/record?token=test (50%% 概率返回验证码)")
	})

	// 启动服务
	fmt.Println("✅ 模拟短信接口已启动：http://localhost:8081/api/record?token=test")
	fmt.Println("📌 成功示例: http://localhost:8081/api/record?token=abc123")
	fmt.Println("📌 失败示例: 多次刷新，约一半概率返回 code=0")

	if err := r.Run(":8081"); err != nil {
		panic(err)
	}
}
