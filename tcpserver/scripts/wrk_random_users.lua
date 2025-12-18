-- 随机用户压测脚本
-- 使用方法: 
--   200并发: wrk -t 10 -c 200 -d 60s --script wrk_random_users.lua http://localhost:8080/api/v1/profile
--   2000并发: wrk -t 20 -c 2000 -d 60s --script wrk_random_users.lua http://localhost:8080/api/v1/profile

-- 加载token列表（使用大用户池模拟随机访问）
local tokens = dofile("tokens_10000000.lua")

local token_count = #tokens

-- 注意：wrk的response函数在多线程下计数器不准确
-- 改用summary.errors来统计

-- 初始化
init = function(args)
    math.randomseed(os.time() + os.clock() * 1000000)
    print("========================================")
    print("随机用户压测")
    print(string.format("Token池大小: %d", token_count))
    print("========================================")
end

-- 每个请求前调用
request = function()
    -- 完全随机选择token（模拟随机用户访问）
    local token_index = math.random(1, token_count)
    local token = tokens[token_index]
    
    -- 设置请求头
    wrk.headers["Cookie"] = "auth_token=" .. token
    wrk.headers["Content-Type"] = "application/json"
    
    -- 返回请求
    return wrk.format("GET", "/api/v1/profile")
end

-- 响应处理（打印错误信息）
response = function(status, headers, body)
    if status ~= 200 then
        print("Error: status=" .. status .. ", body=" .. body)
    end
end

-- 完成后统计
done = function(summary, latency, requests)
    -- 使用summary中的统计数据
    local total_requests = summary.requests
    local error_requests = summary.errors.connect + summary.errors.read + summary.errors.write + summary.errors.status + summary.errors.timeout
    local success_requests = total_requests - error_requests
    local success_rate = (success_requests / total_requests) * 100
    local duration_sec = summary.duration / 1000000
    
    io.write("========================================\n")
    io.write("随机用户压测结果\n")
    io.write("========================================\n")
    io.write(string.format("总请求数: %d\n", total_requests))
    io.write(string.format("成功请求: %d (%.1f%%)\n", success_requests, success_rate))
    io.write(string.format("失败请求: %d (%.1f%%)\n", error_requests, 100 - success_rate))
    io.write(string.format("  连接错误: %d\n", summary.errors.connect))
    io.write(string.format("  读取错误: %d\n", summary.errors.read))
    io.write(string.format("  写入错误: %d\n", summary.errors.write))
    io.write(string.format("  状态码错误: %d\n", summary.errors.status))
    io.write(string.format("  超时错误: %d\n", summary.errors.timeout))
    io.write(string.format("总耗时: %.2fs\n", duration_sec))
    io.write(string.format("总QPS: %.2f (包含失败)\n", total_requests / duration_sec))
    io.write(string.format("成功QPS: %.2f (仅成功请求)\n", success_requests / duration_sec))
    io.write(string.format("平均延迟: %.2fms\n", latency.mean / 1000))
    io.write(string.format("P50延迟: %.2fms\n", latency:percentile(50) / 1000))
    io.write(string.format("P90延迟: %.2fms\n", latency:percentile(90) / 1000))
    io.write(string.format("P99延迟: %.2fms\n", latency:percentile(99) / 1000))
    io.write(string.format("最大延迟: %.2fms\n", latency.max / 1000))
    io.write("========================================\n")
end

