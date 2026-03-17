# auto_card 测试套件

## 测试文件说明

| 文件 | 说明 | 运行方式 |
|------|------|----------|
| `test_api.py` | API 接口测试（异步/同步模式） | `python3 test_api.py` |
| `test_bigdata.py` | 大数据量性能测试（300条卡密） | `python3 test_bigdata.py` |
| `test_frontend.py` | 前端性能测试（需要 Selenium） | `python3 test_frontend.py` |
| `run_tests.sh` | 一键运行所有测试 | `./run_tests.sh` |

## 快速开始

### 1. 安装依赖

```bash
pip3 install requests

# 前端测试需要（可选）
pip3 install selenium webdriver-manager
```

### 2. 运行测试

```bash
# 进入测试目录
cd tests

# 运行所有测试
./run_tests.sh

# 或单独运行
python3 test_api.py
python3 test_bigdata.py
```

## 测试内容

### API 测试 (`test_api.py`)

- ✓ 异步模式查询（有效/无效卡号）
- ✓ 同步模式查询
- ✓ 查询状态轮询
- ✓ 实时验证码列表
- ✓ 管理后台登录
- ✓ 并发查询性能
- ✓ 响应时间测试

### 大数据量测试 (`test_bigdata.py`)

- ✓ 批量添加300条卡密
- ✓ 列表查询性能（分页）
- ✓ 单条查询性能
- ✓ 同步查询性能
- ✓ 并发查询（20并发）
- ✓ 数据清理

### 前端测试 (`test_frontend.py`)

- ✓ 页面加载时间
- ✓ 管理后台登录
- ✓ 数据列表渲染
- ✓ 查询页面功能
- ✓ 内存占用监控

## 关键测试场景

### 场景1: 按需查询模式验证

```bash
# 测试异步模式 - 立即返回，后台查询
python3 -c "
import requests
r = requests.get('https://auto-receive-card-production.up.railway.app/api/cards/query?card=test_card')
print('异步模式响应:', r.json())
"

# 测试同步模式 - 等待远程查询完成
python3 -c "
import requests
import time
start = time.time()
r = requests.get('https://auto-receive-card-production.up.railway.app/api/cards/query?card=test_card&sync=1', timeout=30)
print(f'同步模式响应 ({time.time()-start:.2f}s):', r.json())
"
```

### 场景2: 大数据量性能验证

```bash
# 运行大数据量测试
python3 test_bigdata.py
```

预期结果：
- 列表查询 < 1s
- 单条查询 < 1s
- 并发查询（20并发）< 2s平均响应

### 场景3: 前端50条限制验证

1. 准备100条验证码数据
2. 打开实时验证码面板
3. 验证：
   - 只显示50条
   - 状态栏提示"显示最新 50/100 条"
   - 页面无卡顿

## 测试报告

测试完成后会输出：

```
======================================
测试报告
======================================
通过: 8
失败: 0
总计: 8
通过率: 100.0%
```

## 注意事项

1. **测试环境**: 默认测试生产环境 `auto-receive-card-production.up.railway.app`
2. **测试数据**: 大数据量测试会添加300条测试数据，测试后可选择清理
3. **密码**: 默认使用 `jc123`，如已修改请更新测试脚本
4. **网络**: 确保网络稳定，部分测试有超时限制

## 问题排查

### 测试失败常见原因

1. **网络连接超时**
   - 检查网络连接
   - 确认服务是否在线

2. **认证失败**
   - 确认密码正确
   - 检查 session 是否过期

3. **数据不存在**
   - 先运行大数据量测试添加测试数据
   - 或使用已有卡号测试

## 更新日志

- 2026-03-17: 添加按需查询模式测试
- 2026-03-17: 添加大数据量性能测试
- 2026-03-17: 添加前端渲染测试
