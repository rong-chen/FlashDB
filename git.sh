#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  FlashDB Git Push Script${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if there are changes
if [[ -z $(git status -s) ]]; then
    echo -e "${YELLOW}No changes to commit${NC}"
    exit 0
fi

echo -e "${YELLOW}Changes to commit:${NC}"
git status -s
echo ""

# Get commit message
read -p "请输入 commit 信息: " MESSAGE
if [[ -z "$MESSAGE" ]]; then
    echo -e "${RED}Error: commit 信息不能为空${NC}"
    exit 1
fi

# Add all changes
echo -e "${YELLOW}Adding changes...${NC}"
git add .

# Commit
echo -e "${YELLOW}Committing...${NC}"
git commit -m "$MESSAGE"

# Push
echo -e "${YELLOW}Pushing to origin...${NC}"
git push

echo ""
echo -e "${GREEN}✓ Push Complete!${NC}"
echo ""

# Ask if create release tag
read -p "是否创建 release tag? (y/N): " CREATE_TAG
if [[ "$CREATE_TAG" == "y" || "$CREATE_TAG" == "Y" ]]; then
    read -p "请输入 tag 版本 (例如 v1.0.0): " VERSION
    if [[ -z "$VERSION" ]]; then
        echo -e "${RED}Error: tag 版本不能为空${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}Creating release tag: $VERSION${NC}"
    git tag -a "$VERSION" -m "Release $VERSION"
    git push origin "$VERSION"
    
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Release tag $VERSION 已推送!${NC}"
    echo -e "${GREEN}  GitHub Actions 将自动构建并发布${NC}"
    echo -e "${GREEN}========================================${NC}"
fi
