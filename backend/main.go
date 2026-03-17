// main.go
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// 全局 HTTP Client，带超时设置
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// 获取数据库路径，优先从环境变量读取
func getDBPath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return "./cards.db"
}

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
	dbPath := getDBPath()
	db, err = sql.Open("sqlite3", dbPath)
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

	// 创建索引优化查询性能
	createIndexesSQL := `
	CREATE INDEX IF NOT EXISTS idx_card_no ON cards(card_no);
	CREATE INDEX IF NOT EXISTS idx_query_token ON cards(query_token);
	CREATE INDEX IF NOT EXISTS idx_created_at ON cards(created_at);
	CREATE INDEX IF NOT EXISTS idx_card_check ON cards(card_check);
	CREATE INDEX IF NOT EXISTS idx_card_no_check ON cards(card_no, card_check);
	`
	_, err = db.Exec(createIndexesSQL)
	if err != nil {
		log.Printf("创建索引失败: %v", err)
	} else {
		log.Println("数据库索引创建成功")
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

// ==================== XSS 防护工具函数 ====================

// escapeCard 对 Card 结构体中的字符串字段进行 HTML 转义，防止 XSS
func escapeCard(card Card) Card {
	card.CardNo = html.EscapeString(card.CardNo)
	card.CardLink = html.EscapeString(card.CardLink)
	if card.Phone != nil {
		escaped := html.EscapeString(*card.Phone)
		card.Phone = &escaped
	}
	if card.Remark != nil {
		escaped := html.EscapeString(*card.Remark)
		card.Remark = &escaped
	}
	if card.QueryURL != nil {
		escaped := html.EscapeString(*card.QueryURL)
		card.QueryURL = &escaped
	}
	if card.QueryToken != nil {
		escaped := html.EscapeString(*card.QueryToken)
		card.QueryToken = &escaped
	}
	if card.CardCode != nil {
		escaped := html.EscapeString(*card.CardCode)
		card.CardCode = &escaped
	}
	if card.CardExpiredDate != nil {
		escaped := html.EscapeString(*card.CardExpiredDate)
		card.CardExpiredDate = &escaped
	}
	if card.CardNote != nil {
		escaped := html.EscapeString(*card.CardNote)
		card.CardNote = &escaped
	}
	return card
}

// 登录接口（明文口令校验）
// 请求体：{ "password": string }
// 处理：校验密码（优先从环境变量 ADMIN_PASSWORD 读取，默认 jc123）
// 返回：成功 -> { code:0, data:{ token:"admin" } }；失败 -> 401
func adminLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{Code: -1, Message: "请求格式错误"})
		return
	}

	// 从环境变量读取密码，默认 jc123
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "jc123"
	}

	if req.Password != adminPassword {
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
// 处理：构造 WHERE 条件，查询总数与当前页数据（支持跨表查询）
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
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 1000 {
			pageSize = ps
		}
	}

	// 获取筛选参数
	dateFilter := c.Query("date")       // 日期筛选 (YYYY-MM-DD)
	statusFilter := c.Query("status")   // 状态筛选 (all/checked/unchecked)
	phoneFilter := c.Query("phone")     // 手机号筛选
	cardNoFilter := c.Query("card_no")  // 卡号搜索

	// 获取需要查询的所有表（主表+分表）
	tables := getQueryTables()

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

	// 查询所有表的总数
	total := 0
	for _, table := range tables {
		var count int
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", table, whereClause)
		err := db.QueryRow(countQuery, args...).Scan(&count)
		if err != nil {
			log.Printf("查询表 %s 总数失败: %v", table, err)
			continue
		}
		total += count
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建 UNION ALL 查询
	var unionQueries []string
	for _, table := range tables {
		query := fmt.Sprintf(
			"SELECT id, card_no, card_link, phone, remark, query_url, query_token, created_at, card_code, card_expired_date, card_note, card_check FROM %s %s",
			table, whereClause,
		)
		unionQueries = append(unionQueries, query)
	}

	// 合并查询并排序、分页
	unionSQL := strings.Join(unionQueries, " UNION ALL ")
	finalQuery := unionSQL + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	dataArgs := append(args, pageSize, offset)

	rows, err := db.Query(finalQuery, dataArgs...)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "查询失败: " + err.Error()})
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
		// XSS 防护：对返回数据进行 HTML 转义
		cards = append(cards, escapeCard(card))
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
// 【方案3：按需查询】后台不做全量自动刷新，只在用户查询单个卡密时实时获取数据
func getLiveCodes(c *gin.Context) {
	// 方案3：按需查询模式，不再自动全量刷新所有卡密
	// 用户查询单个卡密时，/api/cards/query 接口会实时获取最新数据

	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// 只返回最近2分钟内有验证码的数据
	query := `SELECT id, card_no, phone, card_code, card_expired_date, created_at
		FROM cards
		WHERE card_check = 1 AND card_code IS NOT NULL AND card_code != ''
		AND (card_expired_date IS NULL OR datetime(card_expired_date) > datetime('now', '-2 minutes'))
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
		// XSS 防护：对返回数据进行 HTML 转义
		cards = append(cards, escapeCard(card))
	}

	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data:    cards,
	})
}

// 自动查询所有卡密（异步版本，带并发控制）
func autoQueryPendingCardsAsync() {
	// 使用带缓冲的 channel 限制并发数（最多10个并发请求）
	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup

	// 查询所有卡密（去掉50条限制，支持大量卡密）
	// 查询所有需要自动查询的卡密（跨表查询）
	tables := getQueryTables()
	var allRows []struct {
		cardNo      string
		cardLink    string
		queryToken  string
	}
	
	for _, table := range tables {
		rows, err := db.Query(fmt.Sprintf(`
			SELECT card_no, card_link, query_token
			FROM %s
			WHERE card_link IS NOT NULL
			AND card_link != ''
			ORDER BY created_at DESC`, table))
		if err != nil {
			log.Printf("自动查询失败(表 %s): %v", table, err)
			continue
		}
		
		for rows.Next() {
			var cardNo, cardLink, queryToken string
			if err := rows.Scan(&cardNo, &cardLink, &queryToken); err != nil {
				continue
			}
			allRows = append(allRows, struct {
				cardNo      string
				cardLink    string
				queryToken  string
			}{cardNo, cardLink, queryToken})
		}
		rows.Close()
	}

	for _, row := range allRows {
		// 使用 query_token 或 card_no 查询
		token := row.queryToken
		if token == "" {
			token = row.cardNo
		}

		// 异步查询，使用信号量控制并发
		wg.Add(1)
		go func(link, t string) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量
			queryRemoteCard(link, t)
		}(row.cardLink, token)
	}

	// 等待所有查询完成
	wg.Wait()
	log.Println("异步自动查询完成")
}

// 查询远程卡密信息（内部使用）
func queryRemoteCard(cardLink, cardNo string) {
	resp, err := httpClient.Get(cardLink)
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

	log.Printf("远程接口返回: code=%d, msg=%s, data=%+v", remoteResp.Code, remoteResp.Msg, remoteResp.Data)

	rawNote, _ := json.Marshal(remoteResp)
	note := string(rawNote)

	// 校验验证码与过期时间（code == 1 或 code == 0 表示成功）
	if (remoteResp.Code == 1 || remoteResp.Code == 0) && remoteResp.Data.Code != "" {
		code := extractVerificationCode(remoteResp.Data.Code)
		expired := convertTimeFormat(remoteResp.Data.ExpiredDate)
		log.Printf("获取到验证码: card=%s, code=%s, expired=%s", cardNo, code, expired)
		
		// 跨表更新
		tables := getQueryTables()
		for _, table := range tables {
			_, err = db.Exec(fmt.Sprintf("UPDATE %s SET card_code=?, card_expired_date=?, card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", table),
				code, expired, note, cardNo, cardNo)
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Printf("更新数据库失败: %v", err)
		}
	} else {
		log.Printf("未获取到验证码: card=%s, code=%d", cardNo, remoteResp.Code)
		// 只标记已查，不更新验证码
		tables := getQueryTables()
		for _, table := range tables {
			_, err = db.Exec(fmt.Sprintf("UPDATE %s SET card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", table),
				note, cardNo, cardNo)
			if err == nil {
				break
			}
		}
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

	// 跨表更新
	tables := getQueryTables()
	updated := false
	for _, table := range tables {
		result, err := db.Exec(fmt.Sprintf("UPDATE %s SET remark = ? WHERE id = ?", table), req.Remark, id)
		if err == nil {
			rowsAffected, _ := result.RowsAffected()
			if rowsAffected > 0 {
				updated = true
				break
			}
		}
	}
	
	if !updated {
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

// ==================== 数据库备份/恢复功能 ====================

const backupDir = "./backups"

// 创建数据库备份
func createBackup(c *gin.Context) {
	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		c.JSON(500, Response{Code: -1, Message: "创建备份目录失败: " + err.Error()})
		return
	}

	// 生成备份文件名: cards_20060102_150405.db
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("cards_%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	// 关闭当前数据库连接（SQLite 需要独占访问）
	db.Close()

	// 复制数据库文件
	src, err := os.Open(getDBPath())
	if err != nil {
		// 重新打开数据库
		db, _ = sql.Open("sqlite3", getDBPath())
		c.JSON(500, Response{Code: -1, Message: "打开源数据库失败: " + err.Error()})
		return
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		db, _ = sql.Open("sqlite3", getDBPath())
		c.JSON(500, Response{Code: -1, Message: "创建备份文件失败: " + err.Error()})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		db, _ = sql.Open("sqlite3", getDBPath())
		c.JSON(500, Response{Code: -1, Message: "复制数据库失败: " + err.Error()})
		return
	}

	// 重新打开数据库连接
	db, err = sql.Open("sqlite3", getDBPath())
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "重新打开数据库失败: " + err.Error()})
		return
	}

	// 获取备份文件信息
	info, _ := os.Stat(backupPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}

	c.JSON(200, Response{
		Code:    0,
		Message: "备份成功",
		Data: map[string]interface{}{
			"name":      backupName,
			"path":      backupPath,
			"size":      size,
			"createdAt": time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}

// 列出所有备份
func listBackups(c *gin.Context) {
	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		c.JSON(500, Response{Code: -1, Message: "创建备份目录失败: " + err.Error()})
		return
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "读取备份目录失败: " + err.Error()})
		return
	}

	type BackupInfo struct {
		Name      string `json:"name"`
		Size      int64  `json:"size"`
		CreatedAt string `json:"createdAt"`
	}

	backups := []BackupInfo{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Name:      entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data:    backups,
	})
}

// 恢复数据库备份
func restoreBackup(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{Code: -1, Message: "请求格式错误: 需要提供备份文件名"})
		return
	}

	backupPath := filepath.Join(backupDir, req.Name)

	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		c.JSON(404, Response{Code: -1, Message: "备份文件不存在"})
		return
	}

	// 关闭当前数据库连接
	db.Close()

	// 备份当前数据库（防止恢复失败丢失数据）
	timestamp := time.Now().Format("20060102_150405")
	currentBackup := fmt.Sprintf("./cards_before_restore_%s.db", timestamp)
	
	src, err := os.Open(getDBPath())
	if err == nil {
		dst, _ := os.Create(currentBackup)
		if dst != nil {
			io.Copy(dst, src)
			dst.Close()
		}
		src.Close()
	}

	// 恢复备份
	src, err = os.Open(backupPath)
	if err != nil {
		db, _ = sql.Open("sqlite3", getDBPath())
		c.JSON(500, Response{Code: -1, Message: "打开备份文件失败: " + err.Error()})
		return
	}
	defer src.Close()

	dst, err := os.Create(getDBPath())
	if err != nil {
		db, _ = sql.Open("sqlite3", getDBPath())
		c.JSON(500, Response{Code: -1, Message: "创建数据库文件失败: " + err.Error()})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		db, _ = sql.Open("sqlite3", getDBPath())
		c.JSON(500, Response{Code: -1, Message: "恢复数据库失败: " + err.Error()})
		return
	}

	// 重新打开数据库连接
	db, err = sql.Open("sqlite3", getDBPath())
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "重新打开数据库失败: " + err.Error()})
		return
	}

	c.JSON(200, Response{
		Code:    0,
		Message: "恢复成功",
		Data: map[string]interface{}{
			"restoredFrom": req.Name,
			"currentSaved": currentBackup,
		},
	})
}

// 删除备份文件
func deleteBackup(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, Response{Code: -1, Message: "备份文件名不能为空"})
		return
	}

	backupPath := filepath.Join(backupDir, name)

	// 安全检查：确保在备份目录内
	absBackupDir, _ := filepath.Abs(backupDir)
	absTarget, _ := filepath.Abs(backupPath)
	if !strings.HasPrefix(absTarget, absBackupDir) {
		c.JSON(403, Response{Code: -1, Message: "非法路径"})
		return
	}

	if err := os.Remove(backupPath); err != nil {
		c.JSON(500, Response{Code: -1, Message: "删除失败: " + err.Error()})
		return
	}

	c.JSON(200, Response{Code: 0, Message: "删除成功"})
}

// 下载备份文件
func downloadBackup(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(400, Response{Code: -1, Message: "备份文件名不能为空"})
		return
	}

	backupPath := filepath.Join(backupDir, name)

	// 安全检查：确保在备份目录内
	absBackupDir, _ := filepath.Abs(backupDir)
	absTarget, _ := filepath.Abs(backupPath)
	if !strings.HasPrefix(absTarget, absBackupDir) {
		c.JSON(403, Response{Code: -1, Message: "非法路径"})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		c.JSON(404, Response{Code: -1, Message: "备份文件不存在"})
		return
	}

	// 设置下载响应头
	c.Header("Content-Disposition", "attachment; filename="+name)
	c.Header("Content-Type", "application/octet-stream")
	c.File(backupPath)
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

	// 如果不允许重复，批量检查哪些卡号已存在
	existingCards := make(map[string]bool)
	if !req.AllowDuplicates && len(cards) > 0 {
		// 提取所有卡号
		cardNos := make([]string, len(cards))
		for i, card := range cards {
			cardNos[i] = card.CardNo
		}

		// 批量查询已存在的卡号（使用 IN 子句，一次查询）
		placeholders := make([]string, len(cardNos))
		args := make([]interface{}, len(cardNos))
		for i, no := range cardNos {
			placeholders[i] = "?"
			args[i] = no
		}

		query := "SELECT card_no FROM cards WHERE card_no IN (" + strings.Join(placeholders, ",") + ")"
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("批量检查重复卡号失败: %v", err)
			// 查询失败时，假设所有卡号都可能存在（安全起见）
			for _, no := range cardNos {
				existingCards[no] = true
			}
		} else {
			defer rows.Close()
			for rows.Next() {
				var existingNo string
				if err := rows.Scan(&existingNo); err == nil {
					existingCards[existingNo] = true
				}
			}
		}
		log.Printf("不允许重复添加，发现 %d 个重复卡号", len(existingCards))
	}

	// 使用事务批量插入
	tx, err := db.Begin()
	if err != nil {
		c.JSON(500, Response{Code: -1, Message: "开启事务失败: " + err.Error()})
		return
	}
	defer tx.Rollback()

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

		result, err := tx.Exec(
			"INSERT INTO cards (card_no, card_link, phone, remark, query_url, query_token, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
			card.CardNo, card.CardLink, card.Phone, req.Remark, queryURL, queryToken,
		)
		if err != nil {
			log.Printf("添加失败 %s: %v", card.CardNo, err)
			continue
		}
		// 获取插入的 ID
		lastID, _ := result.LastInsertId()
		log.Printf("成功添加卡号: %s, ID: %d", card.CardNo, lastID)
		added = append(added, Card{ID: int(lastID), CardNo: card.CardNo, QueryURL: &queryURL})
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, Response{Code: -1, Message: "提交事务失败: " + err.Error()})
		return
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
	
	// 跨表删除
	tables := getQueryTables()
	deletedCount := 0
	
	for _, id := range req.IDs {
		deleted := false
		for _, table := range tables {
			result, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", table), id)
			if err == nil {
				rowsAffected, _ := result.RowsAffected()
				if rowsAffected > 0 {
					deleted = true
					deletedCount++
					break
				}
			}
		}
		if !deleted {
			log.Printf("删除失败 ID=%d: 未找到", id)
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

// 查询验证码并回写结果（异步模式）
// 输入：`GET /api/cards/query?card=卡号`
// 处理：
//  1. 立即返回当前状态，不等待远程查询完成
//  2. 异步触发远程查询（如果该卡号近期未查询过）
//  3. 前端可通过 /api/cards/status?card=卡号 轮询查询结果
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
		// 使用 query_token 字段精确匹配查询参数（跨表查询）
		log.Printf("Query debug - searching for query_token: %s", cardNo)
		
		// 在所有表中查询
		tables := getQueryTables()
		for _, table := range tables {
			err := db.QueryRow(fmt.Sprintf("SELECT card_link FROM %s WHERE query_token = ?", table), cardNo).Scan(&cardLink)
			if err == nil {
				break
			}
		}
		
		if cardLink == "" {
			log.Printf("Query debug - query_token not found: %s", cardNo)
			c.JSON(404, Response{Code: -1, Message: "卡号不存在"})
			return
		}
	}

	// 获取或创建查询任务
	task := getOrCreateQueryTask(cardNo)

	// 如果任务不是正在查询中，异步触发远程查询
	if task.Status != "querying" {
		task.Status = "querying"
		task.UpdatedAt = time.Now()
		go func(cardNo, cardLink string) {
			queryRemoteCardAsync(cardNo, cardLink)
		}(cardNo, cardLink)
	}

	// 立即返回当前状态，不等待
	// 尝试获取当前数据库中的数据（跨表查询）
	var card Card
	var queryURL, queryToken, code, expired, note, phone, remark sql.NullString
	
	// 在所有表中查询
	tables := getQueryTables()
	for _, table := range tables {
		err := db.QueryRow(fmt.Sprintf(`
			SELECT id, card_no, card_link, phone, remark, query_url, query_token, created_at, card_code, card_expired_date, card_note, card_check 
			FROM %s WHERE query_token = ? OR card_no = ?`, table), cardNo, cardNo).Scan(
			&card.ID, &card.CardNo, &card.CardLink, &phone, &remark, &queryURL, &queryToken, &card.CreatedAt, &code, &expired, &note, &card.CardCheck)
		
		if err == nil {
			break // 找到了
		}
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

	c.JSON(200, Response{
		Code:    0, 
		Message: "查询已触发",
		Data: map[string]interface{}{
			"card":       card,
			"task_status": task.Status,
		},
	})
}

// 异步查询远程卡密信息
func queryRemoteCardAsync(cardNo, cardLink string) {
	resp, err := httpClient.Get(cardLink)
	if err != nil {
		log.Printf("远程接口错误: %v", err)
		updateQueryTask(cardNo, "failed", nil, "远程接口错误")
		return
	}
	defer resp.Body.Close()

	var remoteResp RemoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&remoteResp); err != nil {
		log.Printf("解析响应失败: %v", err)
		updateQueryTask(cardNo, "failed", nil, "解析响应失败")
		return
	}

	rawNote, _ := json.Marshal(remoteResp)
	note := string(rawNote)

	// 校验验证码与过期时间
	if remoteResp.Code == 1 && remoteResp.Data.Code != "" {
		code := extractVerificationCode(remoteResp.Data.Code)
		expired := convertTimeFormat(remoteResp.Data.ExpiredDate)
		
		// 更新数据库（跨表更新）
		tables := getQueryTables()
		updated := false
		for _, table := range tables {
			result, err := db.Exec(fmt.Sprintf("UPDATE %s SET card_code=?, card_expired_date=?, card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", table),
				code, expired, note, cardNo, cardNo)
			if err == nil {
				rowsAffected, _ := result.RowsAffected()
				if rowsAffected > 0 {
					updated = true
					break
				}
			}
		}
		
		if !updated {
			log.Printf("更新数据库失败: 未找到卡号 %s", cardNo)
		}
		
		// 更新任务状态
		updateQueryTask(cardNo, "completed", &CardInfo{
			CardNo:          cardNo,
			CardCode:        code,
			CardExpiredDate: expired,
			CardNote:        note,
		}, "")
		
		log.Printf("异步查询完成: card=%s, code=%s", cardNo, code)
	} else {
		// 只标记已查，不更新验证码
		tables := getQueryTables()
		for _, table := range tables {
			_, err = db.Exec(fmt.Sprintf("UPDATE %s SET card_note=?, card_check=1 WHERE query_token = ? OR card_no = ?", table),
				note, cardNo, cardNo)
			if err == nil {
				break
			}
		}
		
		updateQueryTask(cardNo, "completed", nil, "暂未获取验证码")
		log.Printf("异步查询完成（无验证码）: card=%s", cardNo)
	}
}

// 获取查询任务状态
// 输入：`GET /api/cards/status?card=卡号`
// 返回：查询任务的当前状态和结果
func getCardQueryStatus(c *gin.Context) {
	cardNo := c.Query("card")
	if cardNo == "" {
		c.JSON(400, Response{Code: -1, Message: "缺少 card 参数"})
		return
	}

	queryTasksMutex.RLock()
	task, exists := queryTasks[cardNo]
	queryTasksMutex.RUnlock()

	if !exists {
		c.JSON(404, Response{Code: -1, Message: "未找到查询任务"})
		return
	}

	// 同时返回数据库中的最新数据（跨表查询）
	var card Card
	var queryURL, queryToken, code, expired, note, phone, remark sql.NullString
	
	found := false
	tables := getQueryTables()
	for _, table := range tables {
		err := db.QueryRow(fmt.Sprintf(`
			SELECT id, card_no, card_link, phone, remark, query_url, query_token, created_at, card_code, card_expired_date, card_note, card_check 
			FROM %s WHERE query_token = ? OR card_no = ?`, table), cardNo, cardNo).Scan(
			&card.ID, &card.CardNo, &card.CardLink, &phone, &remark, &queryURL, &queryToken, &card.CreatedAt, &code, &expired, &note, &card.CardCheck)
		
		if err == nil {
			found = true
			break
		}
	}
	
	if !found {
		log.Printf("查询本地卡密失败: %s", cardNo)
	} else {
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
	}

	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"task_status": task.Status,
			"task_error":  task.Error,
			"card":        card,
		},
	})
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
	Code    int        `json:"code"`
	Msg     string     `json:"msg"`
	Message string     `json:"message"`
	Data    RemoteData `json:"data"`
}

type RemoteData struct {
	Code        string `json:"code"`
	CodeTime    string `json:"code_time"`
	ExpiredDate string `json:"expired_date"`
}

// ==================== 短信验证码存储 ====================
// 内存存储最近的短信验证码（用于实时面板）
type SMSCode struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	Code      string    `json:"code"`
	Msg       string    `json:"msg"`
	From      string    `json:"from"`
	UserID    string    `json:"user_id"` // 用户标识
	CodeTime  string    `json:"code_time"`
	CreatedAt time.Time `json:"created_at"`
}

// 短信验证码缓存（最多保存100条，2分钟后过期）
var smsCodeCache = make(map[string]*SMSCode)
var smsCacheMutex sync.RWMutex

// 短信推送请求结构
type SMSSyncRequest struct {
	MsgID     string      `json:"msgid"`
	From      interface{} `json:"from"`  // 兼容数字和字符串
	Tel       interface{} `json:"tel"`   // 兼容数字和字符串
	Msg       string      `json:"msg"`
	IsVoice   interface{} `json:"is_voice"`
	CodeTime  string      `json:"code_time"`
	EndTime   string      `json:"end_time"`
	OrderID   interface{} `json:"order_id"`
	OrderNum  string      `json:"ordernum"`
	APIID     interface{} `json:"api_id"`
	APIToken  string      `json:"api_token"`
	UserID    interface{} `json:"user_id"`
	AgentID   interface{} `json:"agent_id"`
	OrderToken string     `json:"order_token"`
}

// 将 interface{} 转换为字符串
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// 接收短信推送
func receiveSMSPush(c *gin.Context) {
	// 打印原始请求体用于调试
	body, _ := c.GetRawData()
	log.Printf("收到短信推送原始数据: %s", string(body))

	var req SMSSyncRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("短信推送解析失败: %v, 原始数据: %s", err, string(body))
		// 即使是测试请求也返回 200，避免对方平台报错
		c.JSON(200, Response{Code: 0, Message: "received"})
		return
	}

	// 如果关键字段为空，可能是测试请求，直接返回成功
	if req.MsgID == "" || req.Tel == nil {
		log.Printf("收到测试请求或空数据")
		c.JSON(200, Response{Code: 0, Message: "received"})
		return
	}

	// 从短信内容中提取验证码
	code := extractCodeFromSMS(req.Msg)
	if code == "" {
		log.Printf("未从短信中提取到验证码: %s", req.Msg)
		// 仍然保存，但验证码为空
	}

	// 转换字段类型
	fromStr := toString(req.From)
	telStr := toString(req.Tel)

	// 清理手机号（去掉+86等前缀）
	phone := cleanPhoneNumber(telStr)

	smsCacheMutex.Lock()
	defer smsCacheMutex.Unlock()

	// 保存到缓存
	smsCodeCache[req.MsgID] = &SMSCode{
		ID:        req.MsgID,
		Phone:     phone,
		Code:      code,
		Msg:       req.Msg,
		From:      fromStr,
		UserID:    toString(req.UserID), // 用户标识
		CodeTime:  req.CodeTime,
		CreatedAt: time.Now(),
	}

	log.Printf("收到短信推送: phone=%s, code=%s, from=%s", phone, code, fromStr)

	// 清理过期数据（2分钟前）
	cleanExpiredSMSCodes()

	c.JSON(200, Response{Code: 0, Message: "success"})
}

// 清理过期的短信验证码
func cleanExpiredSMSCodes() {
	cutoff := time.Now().Add(-1 * time.Minute) // 1分钟过期，与前端保持一致
	for id, sms := range smsCodeCache {
		if sms.CreatedAt.Before(cutoff) {
			delete(smsCodeCache, id)
		}
	}
}

// 获取实时短信验证码列表
func getLiveSMSCodes(c *gin.Context) {
	smsCacheMutex.RLock()
	defer smsCacheMutex.RUnlock()

	// 清理过期数据
	cleanExpiredSMSCodes()

	// 获取过滤参数
	userID := c.Query("user_id")

	// 收集所有唯一的 user_id
	userIDSet := make(map[string]bool)

	// 转换为数组并按时间排序
	var codes []*SMSCode
	for _, sms := range smsCodeCache {
		// 收集所有 user_id
		if sms.UserID != "" {
			userIDSet[sms.UserID] = true
		}
		// 如果指定了 user_id，只返回匹配的
		if userID != "" && sms.UserID != userID {
			continue
		}
		codes = append(codes, sms)
	}

	// 将 user_id 集合转换为数组
	var allUserIDs []string
	for uid := range userIDSet {
		allUserIDs = append(allUserIDs, uid)
	}

	// 按创建时间倒序排列
	sort.Slice(codes, func(i, j int) bool {
		return codes[i].CreatedAt.After(codes[j].CreatedAt)
	})

	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"codes":     codes,
			"user_ids":  allUserIDs,
		},
	})
}

// 从短信内容中提取验证码（6位数字）
func extractCodeFromSMS(msg string) string {
	// 优先匹配常见的验证码格式
	patterns := []string{
		`验证码[是为:：\s]*([0-9]{4,8})`,
		`code[是为:：\s]*([0-9]{4,8})`,
		`([0-9]{4,8})[是为]?验证码`,
		`([0-9]{6})`, // 默认匹配6位数字
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(msg)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// 清理手机号格式
func cleanPhoneNumber(phone string) string {
	// 去掉+86前缀和空格
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.TrimPrefix(phone, "86")
	phone = strings.TrimSpace(phone)
	// 如果手机号是10位且不以1开头，可能在前面补1
	if len(phone) == 10 && !strings.HasPrefix(phone, "1") {
		phone = "1" + phone
	}
	return phone
}

// ==================== 主函数 ====================
// 应用入口：初始化静态托管、路由与 CORS，并启动服务
func main() {
	// 使用环境变量 PORT，默认 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if _, err := os.Stat(getDBPath()); os.IsNotExist(err) {
		os.Create(getDBPath())
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
		api.GET("/admin/backup", createBackup)              // 创建备份
		api.GET("/admin/backups", listBackups)              // 列出备份
		api.GET("/admin/backup/download", downloadBackup)   // 下载备份
		api.POST("/admin/restore", restoreBackup)           // 恢复备份
		api.DELETE("/admin/backup/:name", deleteBackup)     // 删除备份
		api.GET("/cards/query", queryCard)
		api.GET("/cards/status", getCardQueryStatus)        // 查询任务状态
		api.GET("/cards/live", getLiveCodes)
		api.POST("/sms/push", receiveSMSPush)
		api.GET("/sms/live", getLiveSMSCodes)
	}

	// 启动定时清理过期查询任务
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			cleanupExpiredQueryTasks()
			log.Println("清理过期查询任务完成")
		}
	}()

	// 健康检查接口 - 根路径，用于 Railway 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, Response{Code: 0, Message: "OK"})
	})

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
