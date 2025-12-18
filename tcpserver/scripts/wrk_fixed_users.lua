-- 固定用户压测脚本
-- 使用方法: 
--   200并发: wrk -t 10 -c 200 -d 60s --script wrk_fixed_users.lua http://localhost:8080/api/v1/profile
--   2000并发: wrk -t 20 -c 2000 -d 60s --script wrk_fixed_users.lua http://localhost:8080/api/v1/profile

-- 加载token列表（根据并发数修改这一行）
-- 200并发使用: local tokens = dofile("tokens_200.lua")
-- 2000并发使用: local tokens = dofile("tokens_2000.lua")
local tokens = dofile("tokens_2000.lua")

local token_count = #tokens
local counter = 0

-- 初始化
init = function(args)
    math.randomseed(os.time())
    print("========================================")
    print("固定用户压测")
    print(string.format("Token数量: %d", token_count))
    print("========================================")
end

-- 每个请求前调用
request = function()
    -- 顺序轮询token（模拟固定用户反复访问）
    counter = counter + 1
    local token_index = (counter % token_count) + 1
    local token = tokens[token_index]
    
    -- 设置请求头
    wrk.headers["Cookie"] = "auth_token=" .. token
    wrk.headers["Content-Type"] = "application/json"
    
    -- 返回请求
    return wrk.format("GET", "/api/v1/profile")
end

-- 响应处理
response = function(status, headers, body)
    if status ~= 200 then
        print("Error: status=" .. status .. ", body=" .. body)
    end
end

-- 完成后统计
done = function(summary, latency, requests)
    io.write("========================================\n")
    io.write("固定用户压测结果\n")
    io.write("========================================\n")
    io.write(string.format("总请求数: %d\n", summary.requests))
    io.write(string.format("总耗时: %.2fs\n", summary.duration / 1000000))
    io.write(string.format("QPS: %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write(string.format("平均延迟: %.2fms\n", latency.mean / 1000))
    io.write(string.format("P50延迟: %.2fms\n", latency:percentile(50) / 1000))
    io.write(string.format("P90延迟: %.2fms\n", latency:percentile(90) / 1000))
    io.write(string.format("P99延迟: %.2fms\n", latency:percentile(99) / 1000))
    io.write(string.format("最大延迟: %.2fms\n", latency.max / 1000))
    io.write("========================================\n")
end

