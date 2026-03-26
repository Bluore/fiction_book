#!/bin/bash
# 创建20个测试用户

BASE_URL="http://localhost:9929/api/v1/users"
CONTENT_TYPE="Content-Type: application/json"

echo "===== 开始创建20个用户 ====="

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user001","email":"user001@example.com","password":"123456","age":22,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user002","email":"user002@example.com","password":"123456","age":25,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user003","email":"user003@example.com","password":"123456","age":28,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user004","email":"user004@example.com","password":"123456","age":20,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user005","email":"user005@example.com","password":"123456","age":30,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user006","email":"user006@example.com","password":"123456","age":24,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user007","email":"user007@example.com","password":"123456","age":26,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user008","email":"user008@example.com","password":"123456","age":22,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user009","email":"user009@example.com","password":"123456","age":29,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user010","email":"user010@example.com","password":"123456","age":21,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user011","email":"user011@example.com","password":"123456","age":27,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user012","email":"user012@example.com","password":"123456","age":23,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user013","email":"user013@example.com","password":"123456","age":31,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user014","email":"user014@example.com","password":"123456","age":19,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user015","email":"user015@example.com","password":"123456","age":33,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user016","email":"user016@example.com","password":"123456","age":25,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user017","email":"user017@example.com","password":"123456","age":28,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user018","email":"user018@example.com","password":"123456","age":20,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user019","email":"user019@example.com","password":"123456","age":32,"verify":"123456"}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"username":"user020","email":"user020@example.com","password":"123456","age":24,"verify":"123456"}'
echo ""

echo "===== 20个用户创建完成 ====="
