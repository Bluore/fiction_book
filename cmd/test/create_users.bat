@echo off
chcp 65001 >nul
REM 创建20个测试用户

set "BASE_URL=http://localhost:9929/api/v1/users"

echo ===== 开始创建20个用户 =====

curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user001\",\"email\":\"user001@example.com\",\"password\":\"123456\",\"age\":22,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user002\",\"email\":\"user002@example.com\",\"password\":\"123456\",\"age\":25,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user003\",\"email\":\"user003@example.com\",\"password\":\"123456\",\"age\":28,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user004\",\"email\":\"user004@example.com\",\"password\":\"123456\",\"age\":20,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user005\",\"email\":\"user005@example.com\",\"password\":\"123456\",\"age\":30,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user006\",\"email\":\"user006@example.com\",\"password\":\"123456\",\"age\":24,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user007\",\"email\":\"user007@example.com\",\"password\":\"123456\",\"age\":26,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user008\",\"email\":\"user008@example.com\",\"password\":\"123456\",\"age\":22,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user009\",\"email\":\"user009@example.com\",\"password\":\"123456\",\"age\":29,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user010\",\"email\":\"user010@example.com\",\"password\":\"123456\",\"age\":21,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user011\",\"email\":\"user011@example.com\",\"password\":\"123456\",\"age\":27,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user012\",\"email\":\"user012@example.com\",\"password\":\"123456\",\"age\":23,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user013\",\"email\":\"user013@example.com\",\"password\":\"123456\",\"age\":31,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user014\",\"email\":\"user014@example.com\",\"password\":\"123456\",\"age\":19,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user015\",\"email\":\"user015@example.com\",\"password\":\"123456\",\"age\":33,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user016\",\"email\":\"user016@example.com\",\"password\":\"123456\",\"age\":25,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user017\",\"email\":\"user017@example.com\",\"password\":\"123456\",\"age\":28,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user018\",\"email\":\"user018@example.com\",\"password\":\"123456\",\"age\":20,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user019\",\"email\":\"user019@example.com\",\"password\":\"123456\",\"age\":32,\"verify\":\"123456\"}"
curl -s "%BASE_URL%" -X POST -H "Content-Type: application/json" -d "{\"username\":\"user020\",\"email\":\"user020@example.com\",\"password\":\"123456\",\"age\":24,\"verify\":\"123456\"}"

echo.
echo ===== 20个用户创建完成 =====
pause
