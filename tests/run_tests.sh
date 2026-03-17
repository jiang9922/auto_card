#!/bin/bash
# auto_card 测试脚本入口

set -e

# 颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "======================================"
echo "auto_card 测试套件"
echo "======================================"

# 检查 Python
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}错误: 需要安装 Python3${NC}"
    exit 1
fi

# 安装依赖
echo -e "${YELLOW}检查依赖...${NC}"
pip3 install requests -q 2>/dev/null || true

# 运行目录
TEST_DIR="$(dirname "$0")"
cd "$TEST_DIR"

# 运行 API 测试
echo ""
echo -e "${YELLOW}运行 API 接口测试...${NC}"
python3 test_api.py || echo -e "${RED}API 测试完成，有失败项${NC}"

# 运行大数据量测试
echo ""
echo -e "${YELLOW}运行大数据量测试...${NC}"
python3 test_bigdata.py || echo -e "${RED}大数据量测试完成，有失败项${NC}"

echo ""
echo "======================================"
echo -e "${GREEN}所有测试执行完毕${NC}"
echo "======================================"
