// main.go
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// 初始化数据库连接与表结构
// 1. 打开 SQLite 数据库文件 `cards.db`
// 2. 如表不存在则创建 `cards` 表，字段包含：
//   - id: 主键自增
//   - card_no: 卡号，唯一
//   - card_link: 远程查询接口链接
//   - query_url: 后端生成的本系统查询地址
//   - created_at: 创建时间
//   - card_code: 验证码（解析后写入）
//   - card_expired_date: 验证码过期时间（标准化）
//   - card_note: 原始响应保存便于审计
//   - card_check: 是否已查询
func init() {
	var err error
	db, err = sql.Open("sqlite3", "./cards.db")
	if err != nil {
		log.Fatal("数据库打开失败:", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS cards (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_no TEXT NOT NULL,
		card_link TEXT NOT NULL,
		phone TEXT,
		remark TEXT,
		query_url TEXT,
		query_token TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		card_code TEXT,
		card_expired_date TEXT,
		card_note TEXT,
		card_check BOOLEAN DEFAULT FALSE
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("建表失败:", err)
	}
}

// ==================== 响应结构体 ====================
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ==================== 卡密结构体 ====================
type Card struct {
	ID              int     `json:"id"`
	CardNo          string  `json:"card_no"`
	CardLink        string  `json:"card_link"`
	Phone           *string `json:"phone"`
	Remark          *string `json:"remark"`
	QueryURL        *string `json:"query_url"`
	QueryToken      *string `json:"query_token"`
	CreatedAt       string  `json:"created_at"`
	CardCode        *string `json:"card_code"`
	CardExpiredDate *string `json:"card_expired_date"`
	CardNote        *string `json:"card_note"`
	CardCheck       bool    `json:"card_check"`
}

// ==================== 请求结构体 ====================
type AddCardRequest struct {
	CardNo   string `json:"card_no" binding:"required"`
	CardLink string `json:"card_link" binding:"required,url"`
	Phone    string `json:"phone"`
}

type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

type BatchDeleteRequest struct {
	IDs []int `json:"ids" binding:"required"`
}

type BatchExportRequest struct {
	IDs []int `json:"ids" binding:"required"`
}

// ==================== API 接口 ====================

// 登录接口（明文口令校验）
// 请求体：{ "password": string }
// 处理：校验密码是否为 "admin123"
// 返回：成功 -> { code:0, data:{ token:"admin" } }；失败 -> 401
func adminLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{Code: -1, Message: "请求格式错误"})
		return
	}

	if req.Password != "admin123" {
		c.JSON(401, Response{Code: -1, Message: "密码错误"})
		return
	}

	c.JSON(200, Response{Code: 0, Message: "登录成功", Data: map[string]string{"token": "admin"}})
}

// 管理员 Token 校验接口
// 输入：请求头 `Authorization: Bearer admin`
// 处理：解析 Bearer Token，与约定的固定值 "admin" 比较
// 输出：通过 -> 200 { code:0 }；未授权 -> 401 { code:-1 }
func adminVerify(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if token != "admin" {
		c.JSON(401, Response{Code: -1, Message: "未授权"})
		return
	}
	c.JSON(200, Response{Code: 0, Message: "ok"})
}

// 获取卡密列表（分页+筛选）
// 查询参数：
//   - page, page_size：分页控制
//   - date：按 `YYYY-MM-DD` 过滤 created_at
//   - status：`all|checked|unchecked` 按已查状态过滤
//
// 处理：构造 WHERE 条件，查询总数与当前页数据
// 返回：{ cards:Card[], pagination:{ page,page_size,total,total_pages } }
func getAllCards(c *gin.Context) {
	// 获取分页参数
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	page := 1
	pageSize := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// 获取筛选参数
	dateFilter := c.Query("date")       // 日期筛选 (YYYY-MM-DD)
	statusFilter := c.Query("status")   // 状态筛选 (all/checked/unchecked)
	phoneFilter := c.Query("phone")     // 手机号筛选
	cardNoFilter := c.Query("card_no")  // 卡号搜索

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}

	// 卡号搜索（支持模糊匹配）
	if cardNoFilter != "" {
		whereClause += " AND card_no LIKE ?"
		args = append(args, "%"+cardNoFilter+"%")
	}

	// 手机号筛选
	if phoneFilter != "" {
		whereClause += " AND phone = ?"
		args = append(args, phoneFilter)
	}

	// 日期筛选
	if dateFilter != "" {
		whereClause += " AND DATE(created_at) = ?"
		args = append(args, dateFilter)
	}

	// 状态筛选
	if statusFilter == "checked" {
		whereClause += " AND card_check = 1"
	} else if statusFilter == "unchecked" {
		whereClause += " AND card_check = 0"
	}

	// 如果有条件，添加 WHERE 子句
	if whereClause != "" {
		whereClause = "WHERE 1=1" + whereClause
	} else {
		whereClause = "WHERE 1=1"
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数
	var total int
	countQuery := "SELECT COUNT(*) FROM cards " + whereClause
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "查询总数失败"})
		return
	}

	// 查询当前页数据
	query := "SELECT id, card_no, card_link, phone, remark, query_url, query_token, created_at, card_code, card_expired_date, card_note, card_check FROM cards " +
		whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	dataArgs := append(args, pageSize, offset)
	rows, err := db.Query(query, dataArgs...)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "查询失败"})
		return
	}
	defer rows.Close()

	cards := []Card{}
	for rows.Next() {
		var card Card
		var queryURL, queryToken, code, expired, note, phone, remark sql.NullString
		err := rows.Scan(&card.ID, &card.CardNo, &card.CardLink, &phone, &remark, &queryURL, &queryToken, &card.CreatedAt, &code, &expired, &note, &card.CardCheck)
		if err != nil {
			log.Printf("扫描失败: %v", err)
			continue
		}
		if queryURL.Valid {
			card.QueryURL = &queryURL.String
		}
		if queryToken.Valid {
			card.QueryToken = &queryToken.String
		}
		if phone.Valid {
			card.Phone = &phone.String
		}
		if remark.Valid {
			card.Remark = &remark.String
		}
		if code.Valid {
			card.CardCode = &code.String
		}
		if expired.Valid {
			card.CardExpiredDate = &expired.String
		}
		if note.Valid {
			card.CardNote = &note.String
		}
		cards = append(cards, card)
	}

	// 返回分页数据
	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"cards": cards,
			"pagination": map[string]interface{}{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + pageSize - 1) / pageSize,
			},
		},
	})
}

// 获取最新验证码（实时面板用）
// 查询参数：
//   - limit：返回条数，默认 20
// 返回：最近获取的验证码列表
// 自动查询未获取验证码的卡密
func getLiveCodes(c *gin.Context) {
	// 先自动查询未获取验证码的卡密（同步查询，确保获取到最新数据）
	autoQueryPendingCardsSync()

	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	query := `SELECT id, card_no, phone, card_code, card_expired_date, created_at 
		FROM cards 
		WHERE card_check = 1 AND card_code IS NOT NULL AND card_code != ''
		ORDER BY card_expired_date DESC, created_at DESC 
		LIMIT ?`
	
	rows, err := db.Query(query, limit)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "查询失败"})
		return
	}
	defer rows.Close()

	cards := []Card{}
	for rows.Next() {
		var card Card
		var code, expired, created string
		var phone sql.NullString
		err := rows.Scan(&card.ID, &card.CardNo, &phone, &code, &expired, &created)
		if err != nil {
			continue
		}
		card.CardCode = &code
		card.CardExpiredDate = &expired
		card.CreatedAt = created
		if phone.Valid {
			maskedPhone := maskPhone(phone.String)
			card.Phone = &maskedPhone
		}
		cards = append(cards, card)
	}

	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data:    cards,
	})
}

// 自动查询未获取验证码的卡密（同步版本）
func autoQueryPendingCardsSync() {
	// 查询最近添加的、还没有验证码的卡密（最多10条，同步查询）
	rows, err := db.Query(`
		SELECT card_no, card_link, query_token 
		FROM cards 
		WHERE (card_code IS NULL OR card_code = '') 
		AND card_link IS NOT NULL 
		AND card_link != ''
		ORDER BY created_at DESC 
		LIMIT 10`)
	if err != nil {
		log.Printf("自动查询失败: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cardNo, cardLink, queryToken string
		if err := rows.Scan(&cardNo, &cardLink, &queryToken); err != nil {
			continue
		}

		// 使用 query_token 或 card_no 查询
		token := queryToken
		if token == "" {
			token = cardNo
		}

		// 同步查询，确保获取到结果
		queryRemoteCard(cardLink, token)
	}
}

// 查询远程卡密信息（内部使用）
func queryRemoteCard(cardLink, cardNo string) {
	resp, err := http.Get(cardLink)
	if err != nil {
		log.Printf("远程接口错误: %v", err)
		return
	}
	defer resp.Body.Close()

	var remoteResp RemoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&remoteResp); err != nil {
		log.Printf("解析响应失败: %v", err)
		return
	}

	rawNote, _ := json.Marshal(remoteResp)
	note := string(rawNote)

	// 校验验证码与过期时间
	if remoteResp.Code == 1 && remoteResp.Data.Code != "" {
		code := extractVerificationCode(remoteResp.Data.Code)
		expired := convertTimeFormat(remoteResp.Data.ExpiredDate)
		_, err = db.Exec("UPDATE cards SET card_code=?, card_expired_date=?, card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", 
			code, expired, note, cardNo, cardNo)
		if err != nil {
			log.Printf("更新数据库失败: %v", err)
		}
	} else {
		// 只标记已查，不更新验证码
		_, err = db.Exec("UPDATE cards SET card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", 
			note, cardNo, cardNo)
		if err != nil {
			log.Printf("标记已查失败: %v", err)
		}
	}
}

// 更新备注
func updateRemark(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{Code: -1, Message: "请求格式错误"})
		return
	}

	_, err := db.Exec("UPDATE cards SET remark = ? WHERE id = ?", req.Remark, id)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "更新备注失败"})
		return
	}

	c.JSON(200, Response{
		Code:    0,
		Message: "备注已保存",
	})
}

// 获取系统设置
func getSettings(c *gin.Context) {
	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data:    map[string]interface{}{},
	})
}

// 批量添加卡密（按行解析）
// 请求体：{ text:"卡号----链接\n卡号----链接", allow_duplicates: true, remark: "" }
// 处理：逐行解析出 card_no、card_link；为每条生成本系统 `query_url`；
//
//	以 INSERT 写入，allow_duplicates 控制是否允许重复卡号，remark 为批量备注
//
// 返回：成功写入的卡密简要信息（含 query_url）
func addCard(c *gin.Context) {
	var req struct {
		Text            string `json:"text" binding:"required"`
		AllowDuplicates bool   `json:"allow_duplicates"`
		Remark          string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{Code: -1, Message: "请求格式错误"})
		return
	}

	lines := strings.Split(req.Text, "\n")
	cards := []AddCardRequest{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "----")
		if len(parts) != 2 {
			continue
		}
		cards = append(cards, AddCardRequest{CardNo: strings.TrimSpace(parts[0]), CardLink: strings.TrimSpace(parts[1])})
	}

	if len(cards) == 0 {
		c.JSON(400, Response{Code: -1, Message: "未解析到有效卡密"})
		return
	}

	baseURL := getBaseURL(c)
	added := []Card{}
	
	// 如果不允许重复，先检查哪些卡号已存在
	existingCards := make(map[string]bool)
	if !req.AllowDuplicates {
		for _, card := range cards {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM cards WHERE card_no = ?", card.CardNo).Scan(&count)
			if err != nil {
				log.Printf("检查重复卡号失败 %s: %v", card.CardNo, err)
				// 查询失败时，假设卡号已存在（安全起见）
				existingCards[card.CardNo] = true
			} else if count > 0 {
				existingCards[card.CardNo] = true
			}
		}
		log.Printf("不允许重复添加，发现 %d 个重复卡号", len(existingCards))
	}
	
	for _, card := range cards {
		// 如果不允许重复且卡号已存在，则跳过
		if !req.AllowDuplicates && existingCards[card.CardNo] {
			log.Printf("跳过重复卡号: %s", card.CardNo)
			continue
		}
		
		// 生成随机字母后缀，格式：卡号_随机6位字母
		randomSuffix := generateRandomString(6)
		queryToken := fmt.Sprintf("%s_%s", card.CardNo, randomSuffix)
		queryURL := fmt.Sprintf("%s/query?card=%s", baseURL, url.QueryEscape(queryToken))
		
		_, err := db.Exec(
			"INSERT INTO cards (card_no, card_link, phone, remark, query_url, query_token, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
			card.CardNo, card.CardLink, card.Phone, req.Remark, queryURL, queryToken,
		)
		if err != nil {
			log.Printf("添加失败 %s: %v", card.CardNo, err)
			continue
		}
		log.Printf("成功添加卡号: %s", card.CardNo)
		added = append(added, Card{CardNo: card.CardNo, QueryURL: &queryURL})
	}

	log.Printf("批量添加完成: 请求添加 %d 条，成功添加 %d 条，allow_duplicates=%v", len(cards), len(added), req.AllowDuplicates)

	skipped := len(cards) - len(added)
	message := fmt.Sprintf("成功添加 %d 条", len(added))
	if skipped > 0 {
		message = fmt.Sprintf("成功添加 %d 条，跳过 %d 条重复", len(added), skipped)
	}
	
	c.JSON(200, Response{
		Code:    0,
		Message: message,
		Data:    added,
	})
}

// 批量删除卡密
// 请求体：{ ids:number[] }
// 处理：开启事务，逐个按 id 删除
// 返回：操作结果
func batchDelete(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(400, Response{Code: -1, Message: "无效请求"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "事务启动失败"})
		return
	}
	stmt, err := tx.Prepare("DELETE FROM cards WHERE id = ?")
	if err != nil {
		tx.Rollback()
		c.JSON(500, Response{Code: -1, Message: "准备语句失败"})
		return
	}
	for _, id := range req.IDs {
		_, err := stmt.Exec(id)
		if err != nil {
			log.Printf("删除失败 ID=%d: %v", id, err)
		}
	}
	tx.Commit()
	stmt.Close()

	c.JSON(200, Response{Code: 0, Message: "删除成功"})
}

// 批量导出卡密
// 请求体：{ ids:number[] }
// 处理：按 ids 查询 `card_no` 与 `query_url`，生成 `卡号----查询地址` 文本
// 返回：以附件下载的纯文本内容（Content-Disposition）
func batchExport(c *gin.Context) {
	var req BatchExportRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(400, Response{Code: -1, Message: "无效请求"})
		return
	}

	// 安全拼接 SQL
	placeholders := strings.Repeat("?,", len(req.IDs))
	placeholders = placeholders[:len(placeholders)-1] // 去掉末尾逗号
	query := "SELECT card_no, query_url FROM cards WHERE id IN (" + placeholders + ")"
	args := make([]interface{}, len(req.IDs))
	for i, id := range req.IDs {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var no, url sql.NullString
		if err := rows.Scan(&no, &url); err != nil {
			continue
		}
		if no.Valid && url.Valid {
			// Markdown 格式：卡号 [验证码查询](查询链接)
			lines = append(lines, fmt.Sprintf("%s [验证码查询](%s)", no.String, url.String))
		}
	}

	content := strings.Join(lines, "\n")
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"cards_export_%s.txt\"", time.Now().Format("20060102_150405")))
	c.String(200, content)
}

// 查询验证码并回写结果
// 输入：`GET /api/cards/query?card=卡号`
// 处理：
//  1. 读取本地库中的 `card_link` 远程接口地址
//  2. 请求远程接口，解析 JSON 响应
//  3. 若返回包含验证码与过期时间：
//     - 提取纯数字验证码
//     - 标准化过期时间为 RFC3339（UTC）
//     - 更新本地库：`card_code`、`card_expired_date`、`card_note`、`card_check=1`
//     - 返回 { code:0, data:{ card_no, card_code, card_expired_date, card_note } }
//  4. 否则仅保存原始响应到 `card_note` 并标记已查，返回业务失败信息
func queryCard(c *gin.Context) {
	cardNo := c.Query("card")
	log.Printf("Query debug - received card param: %s", cardNo)
	if cardNo == "" {
		c.JSON(400, Response{Code: -1, Message: "缺少 card 参数"})
		return
	}

	var cardLink string
	linkEnc := c.Query("link_enc")
	linkPlain := c.Query("link")
	if linkEnc != "" {
		if dec, err := base64.StdEncoding.DecodeString(linkEnc); err == nil {
			cardLink = string(dec)
		}
	} else if linkPlain != "" {
		cardLink = linkPlain
	}
	if cardLink == "" {
		// 使用 query_token 字段精确匹配查询参数
		log.Printf("Query debug - searching for query_token: %s", cardNo)
		err := db.QueryRow("SELECT card_link FROM cards WHERE query_token = ?", cardNo).Scan(&cardLink)
		if err != nil {
			log.Printf("Query debug - query_token not found: %v", err)
			c.JSON(404, Response{Code: -1, Message: "卡号不存在"})
			return
		}
	}

	resp, err := http.Get(cardLink)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "远程接口错误"})
		return
	}
	defer resp.Body.Close()

	var remoteResp RemoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&remoteResp); err != nil {
		c.JSON(500, Response{Code: -1, Message: "解析响应失败"})
		return
	}

	rawNote, _ := json.Marshal(remoteResp)
	note := string(rawNote)
	// 校验验证码与过期时间 返回code==1
	if remoteResp.Code == 1 && remoteResp.Data.Code != "" {
		code := extractVerificationCode(remoteResp.Data.Code)
		expired := convertTimeFormat(remoteResp.Data.ExpiredDate)
		// 使用 query_token 或纯卡号更新数据库
		_, err = db.Exec("UPDATE cards SET card_code=?, card_expired_date=?, card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", code, expired, note, cardNo, cardNo)
		if err != nil {
			log.Printf("更新数据库失败: %v", err)
		}
		// 提取纯卡号用于返回
		pureCardNo := cardNo
		if idx := strings.Index(cardNo, "_"); idx > 0 {
			pureCardNo = cardNo[:idx]
		}
		c.JSON(200, Response{Code: 0, Message: "success", Data: map[string]interface{}{
			"card_no": pureCardNo, "card_code": code, "card_expired_date": expired, "card_note": note,
		}})
	} else {
		_, err = db.Exec("UPDATE cards SET card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", note, cardNo, cardNo)
		if err != nil {
			log.Printf("标记已查失败: %v", err)
		}
		//c.JSON(200, Response{Code: -1, Message: "暂未获取验证码", Data: map[string]interface{}{"raw_response": note}})
		c.JSON(200, Response{Code: -1, Message: "暂未获取验证码，请在腾讯视频中点击获取，或者稍后重试。", Data: map[string]interface{}{"raw_response": note}})

	}
}

// ==================== 工具函数 ====================
// 生成随机字符串（大小写字母+数字）
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// 构造当前请求的基础地址（协议+主机）
func getBaseURL(c *gin.Context) string {
	// 优先使用环境变量设置的域名
	if host := os.Getenv("RAILWAY_PUBLIC_DOMAIN"); host != "" {
		return "https://" + host
	}
	// 回退到请求头中的 Host
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// 从字符串中提取连续数字作为验证码
func extractVerificationCode(s string) string {
	re := regexp.MustCompile(`\d+`)
	return re.FindString(s)
}

// 手机号脱敏：中间4位显示为****
// 格式：138****5678
func maskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// 将 `yyyy-MM-dd HH:mm:ss` 转为 `RFC3339 UTC`，失败返回空串
func convertTimeFormat(s string) string {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

type RemoteResponse struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data RemoteData `json:"data"`
}

type RemoteData struct {
	Code        string `json:"code"`
	CodeTime    string `json:"code_time"`
	ExpiredDate string `json:"expired_date"`
}

// ==================== 主函数 ====================
// 应用入口：初始化静态托管、路由与 CORS，并启动服务
func main() {
	// 使用环境变量 PORT，默认 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if _, err := os.Stat("./cards.db"); os.IsNotExist(err) {
		os.Create("./cards.db")
	}

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/admin/login", adminLogin)
		api.GET("/admin/verify", adminVerify)
		api.GET("/admin/settings", getSettings)
		api.GET("/cards", getAllCards)
		api.POST("/cards", addCard)
		api.PUT("/cards/:id/remark", updateRemark)
		api.DELETE("/admin/batch-delete", batchDelete)
		api.POST("/admin/export", batchExport)
		api.GET("/cards/query", queryCard)
		api.GET("/cards/live", getLiveCodes)
	}

	// 静态文件服务 - 支持 Railway 路径
	frontendDist := "./frontend/dist"
	if _, err := os.Stat(frontendDist); os.IsNotExist(err) {
		frontendDist = "/root/frontend/dist" // Docker 路径
	}

	// 静态资源
	r.Static("/assets", frontendDist+"/assets")
	r.StaticFile("/favicon.ico", frontendDist+"/favicon.ico")

	// SPA 路由处理：所有非 API 请求返回 index.html
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// API 请求直接返回 404
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"code": -1, "message": "API not found"})
			return
		}
		// 其他请求返回 index.html，让 Vue Router 处理
		c.File(frontendDist + "/index.html")
	})

	log.Printf("服务启动: http://0.0.0.0:%s", port)
	r.Run("0.0.0.0:" + port)
}
