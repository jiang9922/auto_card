#!/usr/bin/env python3
"""
auto_card API 测试脚本
测试按需查询模式（同步/异步）
"""

import requests
import json
import time
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed

# 配置
BASE_URL = "https://auto-receive-card-production.up.railway.app"
ADMIN_PASSWORD = "jc123"

# 颜色输出
class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    END = '\033[0m'

def print_success(msg):
    print(f"{Colors.GREEN}✓ {msg}{Colors.END}")

def print_error(msg):
    print(f"{Colors.RED}✗ {msg}{Colors.END}")

def print_info(msg):
    print(f"{Colors.BLUE}ℹ {msg}{Colors.END}")

def print_warning(msg):
    print(f"{Colors.YELLOW}⚠ {msg}{Colors.END}")

class APITester:
    def __init__(self, base_url):
        self.base_url = base_url
        self.session = requests.Session()
        self.token = None
        self.passed = 0
        self.failed = 0
        
    def test(self, name, method, expected_pass=True):
        """装饰器：运行测试用例"""
        def decorator(func):
            def wrapper(*args, **kwargs):
                print(f"\n{'='*60}")
                print(f"测试: {name}")
                print(f"{'='*60}")
                try:
                    result = func(*args, **kwargs)
                    if expected_pass and result:
                        print_success(f"通过: {name}")
                        self.passed += 1
                    elif not expected_pass and not result:
                        print_success(f"通过（预期失败）: {name}")
                        self.passed += 1
                    else:
                        print_error(f"失败: {name}")
                        self.failed += 1
                    return result
                except Exception as e:
                    print_error(f"异常: {name} - {str(e)}")
                    self.failed += 1
                    return False
            return wrapper
        return decorator
    
    def report(self):
        """输出测试报告"""
        print(f"\n{'='*60}")
        print("测试报告")
        print(f"{'='*60}")
        print(f"通过: {self.passed}")
        print(f"失败: {self.failed}")
        print(f"总计: {self.passed + self.failed}")
        print(f"通过率: {self.passed/(self.passed+self.failed)*100:.1f}%")
        return self.failed == 0

# ========== 测试用例 ==========

tester = APITester(BASE_URL)

@tester.test("API-001: 异步模式查询-有效卡号", "异步查询")
def test_async_query_valid():
    """测试异步模式查询有效卡号"""
    # 注意：这里使用一个假设的卡号，实际测试时需要使用真实的
    card_no = "test_card_001"
    response = requests.get(f"{BASE_URL}/api/cards/query?card={card_no}", timeout=5)
    
    if response.status_code == 200:
        data = response.json()
        if data.get("code") == 0:
            print_info(f"响应: {json.dumps(data, indent=2, ensure_ascii=False)}")
            return "task_status" in str(data)
    
    print_info(f"状态码: {response.status_code}")
    print_info(f"响应: {response.text}")
    return False

@tester.test("API-002: 异步模式查询-无效卡号", "异步查询")
def test_async_query_invalid():
    """测试异步模式查询无效卡号"""
    card_no = "invalid_card_99999"
    response = requests.get(f"{BASE_URL}/api/cards/query?card={card_no}", timeout=5)
    
    print_info(f"状态码: {response.status_code}")
    print_info(f"响应: {response.text}")
    
    if response.status_code == 404:
        data = response.json()
        return data.get("code") == -1 and "不存在" in data.get("message", "")
    return False

@tester.test("API-003: 同步模式查询-有效卡号", "同步查询")
def test_sync_query_valid():
    """测试同步模式查询有效卡号"""
    card_no = "test_card_001"
    start_time = time.time()
    
    response = requests.get(f"{BASE_URL}/api/cards/query?card={card_no}&sync=1", timeout=30)
    elapsed = time.time() - start_time
    
    print_info(f"响应时间: {elapsed:.2f}s")
    print_info(f"响应: {response.text[:500]}")
    
    if response.status_code == 200:
        data = response.json()
        if data.get("code") == 0:
            return True
    return False

@tester.test("API-004: 查询状态接口", "状态查询")
def test_query_status():
    """测试查询状态接口"""
    card_no = "test_card_001"
    response = requests.get(f"{BASE_URL}/api/cards/status?card={card_no}", timeout=5)
    
    print_info(f"状态码: {response.status_code}")
    print_info(f"响应: {response.text[:500]}")
    
    # 可能返回任务状态或"未找到"
    if response.status_code in [200, 404]:
        return True
    return False

@tester.test("API-005: 实时验证码列表", "实时列表")
def test_live_codes():
    """测试实时验证码列表接口"""
    response = requests.get(f"{BASE_URL}/api/cards/live", timeout=5)
    
    print_info(f"状态码: {response.status_code}")
    
    if response.status_code == 200:
        data = response.json()
        if data.get("code") == 0:
            codes = data.get("data", [])
            print_info(f"返回 {len(codes)} 条验证码")
            return True
    return False

@tester.test("API-006: 管理后台登录-正确密码", "登录")
def test_admin_login_valid():
    """测试管理后台正确密码登录"""
    response = requests.post(
        f"{BASE_URL}/api/admin/login",
        json={"password": ADMIN_PASSWORD},
        timeout=5
    )
    
    print_info(f"响应: {response.text}")
    
    if response.status_code == 200:
        data = response.json()
        if data.get("code") == 0:
            return True
    return False

@tester.test("API-007: 管理后台登录-错误密码", "登录")
def test_admin_login_invalid():
    """测试管理后台错误密码登录"""
    response = requests.post(
        f"{BASE_URL}/api/admin/login",
        json={"password": "wrong_password"},
        timeout=5
    )
    
    print_info(f"响应: {response.text}")
    
    if response.status_code == 401:
        return True
    return False

@tester.test("API-008: 健康检查", "健康")
def test_health_check():
    """测试健康检查接口"""
    response = requests.get(f"{BASE_URL}/health", timeout=5)
    
    print_info(f"响应: {response.text}")
    
    if response.status_code == 200:
        data = response.json()
        return data.get("code") == 0
    return False

@tester.test("API-009: 并发查询测试", "并发")
def test_concurrent_query():
    """测试并发查询性能"""
    card_no = "test_card_001"
    concurrent = 10
    
    def single_query():
        try:
            response = requests.get(
                f"{BASE_URL}/api/cards/query?card={card_no}", 
                timeout=5
            )
            return response.status_code == 200
        except:
            return False
    
    start_time = time.time()
    
    with ThreadPoolExecutor(max_workers=concurrent) as executor:
        futures = [executor.submit(single_query) for _ in range(concurrent)]
        results = [f.result() for f in as_completed(futures)]
    
    elapsed = time.time() - start_time
    success_count = sum(results)
    
    print_info(f"并发数: {concurrent}")
    print_info(f"成功: {success_count}/{concurrent}")
    print_info(f"总耗时: {elapsed:.2f}s")
    print_info(f"平均响应: {elapsed/concurrent:.3f}s")
    
    return success_count == concurrent and elapsed < 10

@tester.test("API-010: 响应时间测试", "性能")
def test_response_time():
    """测试API响应时间"""
    card_no = "test_card_001"
    times = []
    
    for i in range(5):
        start = time.time()
        try:
            response = requests.get(
                f"{BASE_URL}/api/cards/query?card={card_no}", 
                timeout=5
            )
            elapsed = time.time() - start
            times.append(elapsed)
        except:
            times.append(999)
    
    avg_time = sum(times) / len(times)
    max_time = max(times)
    min_time = min(times)
    
    print_info(f"平均响应: {avg_time:.3f}s")
    print_info(f"最大响应: {max_time:.3f}s")
    print_info(f"最小响应: {min_time:.3f}s")
    
    return avg_time < 1.0  # 平均响应小于1秒

# ========== 主函数 ==========

def main():
    print(f"{'='*60}")
    print("auto_card API 测试套件")
    print(f"{'='*60}")
    print(f"测试地址: {BASE_URL}")
    print(f"测试时间: {time.strftime('%Y-%m-%d %H:%M:%S')}")
    
    # 运行所有测试
    test_async_query_valid()
    test_async_query_invalid()
    test_sync_query_valid()
    test_query_status()
    test_live_codes()
    test_admin_login_valid()
    test_admin_login_invalid()
    test_health_check()
    test_concurrent_query()
    test_response_time()
    
    # 输出报告
    success = tester.report()
    
    return 0 if success else 1

if __name__ == "__main__":
    sys.exit(main())
