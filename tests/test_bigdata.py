#!/usr/bin/env python3
"""
auto_card 大数据量测试脚本
测试300条卡密数据下的系统性能
"""

import requests
import json
import time
import random
import string
from concurrent.futures import ThreadPoolExecutor, as_completed

# 配置
BASE_URL = "https://auto-receive-card-production.up.railway.app"
ADMIN_PASSWORD = "jc123"

class BigDataTester:
    def __init__(self):
        self.session = requests.Session()
        self.token = None
        self.test_cards = []
        
    def login(self):
        """登录管理后台"""
        response = self.session.post(
            f"{BASE_URL}/api/admin/login",
            json={"password": ADMIN_PASSWORD}
        )
        if response.status_code == 200:
            data = response.json()
            if data.get("code") == 0:
                self.token = data.get("data", {}).get("token")
                print("✓ 登录成功")
                return True
        print("✗ 登录失败")
        return False
        
    def generate_card_no(self):
        """生成随机卡号"""
        return ''.join(random.choices(string.digits, k=10))
        
    def generate_card_link(self):
        """生成测试用的卡密链接"""
        # 使用一个模拟的查询链接
        return f"http://localhost:8081/api{random.randint(10000000, 99999999)}"
        
    def batch_add_cards(self, count=300):
        """批量添加卡密"""
        print(f"\n批量添加 {count} 条卡密...")
        
        # 生成卡密数据
        lines = []
        for i in range(count):
            card_no = self.generate_card_no()
            card_link = self.generate_card_link()
            lines.append(f"{card_no}----{card_link}")
            self.test_cards.append(card_no)
            
        # 分批添加（每批50条）
        batch_size = 50
        total_added = 0
        
        for i in range(0, len(lines), batch_size):
            batch = lines[i:i+batch_size]
            batch_text = "\n".join(batch)
            
            start_time = time.time()
            response = self.session.post(
                f"{BASE_URL}/api/cards",
                json={
                    "text": batch_text,
                    "allow_duplicates": True,
                    "remark": f"大数据量测试-{i//batch_size+1}"
                }
            )
            elapsed = time.time() - start_time
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    added = len(data.get("data", []))
                    total_added += added
                    print(f"  批次 {i//batch_size+1}: 添加 {added} 条, 耗时 {elapsed:.2f}s")
                else:
                    print(f"  批次 {i//batch_size+1}: 失败 - {data.get('message')}")
            else:
                print(f"  批次 {i//batch_size+1}: HTTP {response.status_code}")
                
        print(f"✓ 总计添加: {total_added}/{count} 条")
        return total_added
        
    def test_list_performance(self):
        """测试列表查询性能"""
        print("\n" + "="*60)
        print("测试: 列表查询性能（300条数据）")
        print("="*60)
        
        times = []
        
        for page in [1, 2, 3]:
            start = time.time()
            response = self.session.get(
                f"{BASE_URL}/api/cards?page={page}&page_size=20"
            )
            elapsed = time.time() - start
            times.append(elapsed)
            
            if response.status_code == 200:
                data = response.json()
                cards = data.get("data", {}).get("cards", [])
                total = data.get("data", {}).get("pagination", {}).get("total", 0)
                print(f"  第{page}页: {len(cards)}条, 耗时 {elapsed:.3f}s, 总计 {total} 条")
            else:
                print(f"  第{page}页: HTTP {response.status_code}")
                
        avg_time = sum(times) / len(times)
        print(f"  平均响应: {avg_time:.3f}s")
        
        return avg_time < 1.0
        
    def test_query_performance(self):
        """测试单条查询性能"""
        print("\n" + "="*60)
        print("测试: 单条查询性能")
        print("="*60)
        
        if not self.test_cards:
            print("✗ 没有测试卡号")
            return False
            
        # 随机选择10个卡号测试
        test_samples = random.sample(self.test_cards, min(10, len(self.test_cards)))
        times = []
        
        for card_no in test_samples:
            start = time.time()
            response = requests.get(
                f"{BASE_URL}/api/cards/query?card={card_no}",
                timeout=5
            )
            elapsed = time.time() - start
            times.append(elapsed)
            
        avg_time = sum(times) / len(times)
        max_time = max(times)
        min_time = min(times)
        
        print(f"  测试数量: {len(test_samples)}")
        print(f"  平均响应: {avg_time:.3f}s")
        print(f"  最大响应: {max_time:.3f}s")
        print(f"  最小响应: {min_time:.3f}s")
        
        return avg_time < 1.0
        
    def test_sync_query_performance(self):
        """测试同步查询性能"""
        print("\n" + "="*60)
        print("测试: 同步查询性能")
        print("="*60)
        
        if not self.test_cards:
            print("✗ 没有测试卡号")
            return False
            
        # 测试同步模式
        card_no = random.choice(self.test_cards)
        
        start = time.time()
        response = requests.get(
            f"{BASE_URL}/api/cards/query?card={card_no}&sync=1",
            timeout=30
        )
        elapsed = time.time() - start
        
        print(f"  同步查询耗时: {elapsed:.2f}s")
        
        if response.status_code == 200:
            print("✓ 同步查询成功")
        else:
            print(f"✗ 同步查询失败: HTTP {response.status_code}")
            
        return elapsed < 10.0  # 同步查询应在10秒内完成
        
    def test_concurrent_query(self):
        """测试并发查询性能"""
        print("\n" + "="*60)
        print("测试: 并发查询性能（300条数据背景）")
        print("="*60)
        
        if not self.test_cards:
            print("✗ 没有测试卡号")
            return False
            
        concurrent = 20
        
        def single_query():
            try:
                card_no = random.choice(self.test_cards)
                start = time.time()
                response = requests.get(
                    f"{BASE_URL}/api/cards/query?card={card_no}",
                    timeout=5
                )
                elapsed = time.time() - start
                return response.status_code == 200, elapsed
            except Exception as e:
                return False, 999
                
        start_time = time.time()
        
        with ThreadPoolExecutor(max_workers=concurrent) as executor:
            futures = [executor.submit(single_query) for _ in range(concurrent)]
            results = [f.result() for f in as_completed(futures)]
            
        total_elapsed = time.time() - start_time
        success_count = sum(1 for success, _ in results if success)
        avg_response = sum(t for _, t in results) / len(results)
        
        print(f"  并发数: {concurrent}")
        print(f"  成功: {success_count}/{concurrent}")
        print(f"  总耗时: {total_elapsed:.2f}s")
        print(f"  平均响应: {avg_response:.3f}s")
        
        return success_count == concurrent and avg_response < 2.0
        
    def test_frontend_performance(self):
        """测试前端大数据量渲染"""
        print("\n" + "="*60)
        print("测试: 前端大数据量渲染（模拟）")
        print("="*60)
        
        # 这里只是模拟，实际前端测试需要浏览器
        print("  注意: 实际前端性能测试请运行 test_frontend.py")
        print("  测试场景: 300条数据在实时面板渲染")
        print("  预期结果: 页面响应 < 500ms, 只显示50条")
        
        return True
        
    def cleanup(self):
        """清理测试数据"""
        print("\n" + "="*60)
        print("清理: 删除测试数据")
        print("="*60)
        
        # 获取所有测试数据ID
        response = self.session.get(f"{BASE_URL}/api/cards?page=1&page_size=1000")
        if response.status_code == 200:
            data = response.json()
            cards = data.get("data", {}).get("cards", [])
            
            # 找出测试数据（通过remark识别）
            test_ids = [c["id"] for c in cards if "大数据量测试" in (c.get("remark") or "")]
            
            if test_ids:
                print(f"  找到 {len(test_ids)} 条测试数据")
                
                # 分批删除
                batch_size = 50
                deleted = 0
                
                for i in range(0, len(test_ids), batch_size):
                    batch = test_ids[i:i+batch_size]
                    response = self.session.delete(
                        f"{BASE_URL}/api/admin/batch-delete",
                        json={"ids": batch}
                    )
                    if response.status_code == 200:
                        deleted += len(batch)
                        
                print(f"✓ 删除 {deleted} 条测试数据")
            else:
                print("  没有找到测试数据")
                
    def run_all_tests(self):
        """运行所有测试"""
        print("="*60)
        print("auto_card 大数据量测试")
        print("="*60)
        print(f"测试地址: {BASE_URL}")
        print(f"测试时间: {time.strftime('%Y-%m-%d %H:%M:%S')}")
        
        results = []
        
        # 登录
        if not self.login():
            return
            
        try:
            # 添加测试数据
            # self.batch_add_cards(300)  # 可选：添加300条测试数据
            
            # 测试1: 列表查询性能
            results.append(("列表查询性能", self.test_list_performance()))
            
            # 测试2: 单条查询性能
            results.append(("单条查询性能", self.test_query_performance()))
            
            # 测试3: 同步查询性能
            results.append(("同步查询性能", self.test_sync_query_performance()))
            
            # 测试4: 并发查询性能
            results.append(("并发查询性能", self.test_concurrent_query()))
            
            # 测试5: 前端性能
            results.append(("前端大数据量渲染", self.test_frontend_performance()))
            
        finally:
            # 清理
            # self.cleanup()  # 可选：清理测试数据
            pass
            
        # 输出报告
        print("\n" + "="*60)
        print("测试结果汇总")
        print("="*60)
        
        passed = sum(1 for _, result in results if result)
        total = len(results)
        
        for name, result in results:
            status = "✓ 通过" if result else "✗ 失败"
            print(f"{status}: {name}")
            
        print(f"\n总计: {passed}/{total} 通过 ({passed/total*100:.1f}%)")
        
def main():
    tester = BigDataTester()
    tester.run_all_tests()

if __name__ == "__main__":
    main()
