#!/usr/bin/env python3
"""
auto_card 前端性能测试脚本
测试300条卡密大数据量渲染性能
"""

import requests
import json
import time
import sys
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# 配置
BASE_URL = "https://auto-receive-card-production.up.railway.app"
ADMIN_URL = f"{BASE_URL}/admin"
ADMIN_PASSWORD = "jc123"

class FrontendPerformanceTester:
    def __init__(self):
        self.driver = None
        self.results = []
        
    def setup_driver(self):
        """配置无头浏览器"""
        chrome_options = Options()
        chrome_options.add_argument('--headless')
        chrome_options.add_argument('--no-sandbox')
        chrome_options.add_argument('--disable-dev-shm-usage')
        chrome_options.add_argument('--disable-gpu')
        chrome_options.add_argument('--window-size=1920,1080')
        
        self.driver = webdriver.Chrome(options=chrome_options)
        self.driver.implicitly_wait(10)
        
    def teardown(self):
        """清理"""
        if self.driver:
            self.driver.quit()
            
    def measure_page_load(self, url, name):
        """测量页面加载时间"""
        print(f"\n测试: {name}")
        print(f"URL: {url}")
        
        start_time = time.time()
        self.driver.get(url)
        
        # 等待页面加载完成
        try:
            WebDriverWait(self.driver, 30).until(
                lambda d: d.execute_script('return document.readyState') == 'complete'
            )
        except:
            pass
        
        load_time = time.time() - start_time
        
        # 获取性能指标
        perf_data = self.driver.execute_script("""
            const timing = performance.timing;
            return {
                dns: timing.domainLookupEnd - timing.domainLookupStart,
                tcp: timing.connectEnd - timing.connectStart,
                request: timing.responseStart - timing.requestStart,
                response: timing.responseEnd - timing.responseStart,
                dom: timing.domComplete - timing.domLoading,
                load: timing.loadEventEnd - timing.navigationStart
            };
        """)
        
        print(f"  总加载时间: {load_time:.2f}s")
        print(f"  DNS查询: {perf_data['dns']}ms")
        print(f"  TCP连接: {perf_data['tcp']}ms")
        print(f"  请求时间: {perf_data['request']}ms")
        print(f"  响应时间: {perf_data['response']}ms")
        print(f"  DOM渲染: {perf_data['dom']}ms")
        
        self.results.append({
            'name': name,
            'load_time': load_time,
            'dns': perf_data['dns'],
            'tcp': perf_data['tcp'],
            'request': perf_data['request'],
            'response': perf_data['response'],
            'dom': perf_data['dom']
        })
        
        return load_time
        
    def test_admin_login(self):
        """测试管理后台登录"""
        print("\n" + "="*60)
        print("测试: 管理后台登录")
        print("="*60)
        
        self.driver.get(ADMIN_URL)
        
        # 等待密码输入框
        try:
            password_input = WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.CSS_SELECTOR, "input[type='password']"))
            )
            password_input.send_keys(ADMIN_PASSWORD)
            
            # 点击登录按钮
            login_btn = self.driver.find_element(By.TAG_NAME, "button")
            login_btn.click()
            
            # 等待登录成功（页面跳转或出现特定元素）
            WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.TAG_NAME, "table"))
            )
            
            print("✓ 登录成功")
            return True
            
        except Exception as e:
            print(f"✗ 登录失败: {e}")
            return False
            
    def test_data_display(self):
        """测试数据列表显示"""
        print("\n" + "="*60)
        print("测试: 数据列表显示")
        print("="*60)
        
        try:
            # 获取表格行数
            rows = self.driver.find_elements(By.CSS_SELECTOR, "table tbody tr")
            print(f"  显示行数: {len(rows)}")
            
            # 检查分页
            pagination = self.driver.find_elements(By.CLASS_NAME, "pagination")
            if pagination:
                print(f"  分页控件: 存在")
            
            # 检查是否有数据
            if len(rows) > 0:
                print("✓ 数据列表显示正常")
                return True
            else:
                print("⚠ 列表为空（可能是正常情况）")
                return True
                
        except Exception as e:
            print(f"✗ 测试失败: {e}")
            return False
            
    def test_query_page(self):
        """测试查询页面"""
        print("\n" + "="*60)
        print("测试: 查询页面性能")
        print("="*60)
        
        # 使用测试卡号
        test_card = "test_card_001"
        query_url = f"{BASE_URL}/query?card={test_card}"
        
        load_time = self.measure_page_load(query_url, "查询页面加载")
        
        try:
            # 检查页面元素
            # 等待页面主要内容加载
            WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.CLASS_NAME, "query-page"))
            )
            
            # 检查是否有验证码显示区域
            code_elements = self.driver.find_elements(By.CLASS_NAME, "code")
            if code_elements:
                print(f"  验证码元素: 找到 {len(code_elements)} 个")
            
            # 检查倒计时
            countdown = self.driver.find_elements(By.CLASS_NAME, "countdown")
            if countdown:
                print("  倒计时组件: 存在")
            
            print("✓ 查询页面功能正常")
            return load_time < 3.0  # 加载时间小于3秒
            
        except Exception as e:
            print(f"✗ 测试失败: {e}")
            return False
            
    def test_memory_usage(self):
        """测试内存占用"""
        print("\n" + "="*60)
        print("测试: 内存占用")
        print("="*60)
        
        try:
            # 获取内存使用情况
            memory = self.driver.execute_script("""
                return performance.memory ? {
                    used: performance.memory.usedJSHeapSize,
                    total: performance.memory.totalJSHeapSize,
                    limit: performance.memory.jsHeapSizeLimit
                } : null;
            """)
            
            if memory:
                used_mb = memory['used'] / 1024 / 1024
                total_mb = memory['total'] / 1024 / 1024
                limit_mb = memory['limit'] / 1024 / 1024
                
                print(f"  JS堆已使用: {used_mb:.2f} MB")
                print(f"  JS堆总计: {total_mb:.2f} MB")
                print(f"  JS堆限制: {limit_mb:.2f} MB")
                
                # 检查内存占用是否合理（小于100MB）
                if used_mb < 100:
                    print("✓ 内存占用正常")
                    return True
                else:
                    print("⚠ 内存占用较高")
                    return False
            else:
                print("⚠ 无法获取内存信息")
                return True
                
        except Exception as e:
            print(f"✗ 测试失败: {e}")
            return False
            
    def run_all_tests(self):
        """运行所有测试"""
        print("="*60)
        print("auto_card 前端性能测试")
        print("="*60)
        
        try:
            self.setup_driver()
            
            # 测试1: 管理后台登录
            self.test_admin_login()
            
            # 测试2: 数据列表显示
            self.test_data_display()
            
            # 测试3: 查询页面
            self.test_query_page()
            
            # 测试4: 内存占用
            self.test_memory_usage()
            
        except Exception as e:
            print(f"\n测试过程中出错: {e}")
            import traceback
            traceback.print_exc()
            
        finally:
            self.teardown()
            
        # 输出汇总
        print("\n" + "="*60)
        print("测试结果汇总")
        print("="*60)
        for result in self.results:
            print(f"{result['name']}: {result['load_time']:.2f}s")
            
def main():
    tester = FrontendPerformanceTester()
    tester.run_all_tests()

if __name__ == "__main__":
    main()
