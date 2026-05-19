-- wrk Lua 脚本 - 认证用户基准测试
-- 模拟用户登录并携带 JWT Token 访问接口

local token = nil
local email_counter = 0
local thread_id = 0

function setup(thread)
    thread_id = thread_id + 1
end

function init()
    -- 每个连接初始化时尝试登录
    if token == nil then
        -- 先尝试登录已有用户
        local login_body = '{"username":"benchuser_' .. thread_id .. '","password":"benchpass123"}'
        wrk.method = "POST"
        wrk.path = "/api/v1/auth/login"
        wrk.headers["Content-Type"] = "application/json"
        wrk.body = login_body
    else
        wrk.method = "GET"
        wrk.headers["Authorization"] = "Bearer " .. token
        wrk.headers["Content-Type"] = ""
        wrk.body = nil
    end
end

function response(status, headers, body)
    if token == nil and status == 200 then
        -- 登录成功，解析 token
        local ok, parsed = pcall(function()
            return body:match('"token"%s*:%s*"([^"]+)"')
        end)
        if ok and parsed then
            token = parsed
            -- 后续请求访问推荐流
            wrk.method = "GET"
            wrk.path = "/api/v1/videos/feed"
            wrk.headers["Authorization"] = "Bearer " .. token
            wrk.headers["Content-Type"] = ""
            wrk.body = nil
        end
    elseif status == 404 or status == 401 or status == 400 then
        -- 用户不存在，注册新用户
        email_counter = email_counter + 1
        local ts = wrk.format("%d", 0)  -- 无法获取时间戳，使用计数器
        local register_body = '{"username":"benchuser_' .. thread_id .. '_' .. email_counter .. '","password":"benchpass123","email":"bench' .. thread_id .. '_' .. email_counter .. '@test.com"}'
        wrk.method = "POST"
        wrk.path = "/api/v1/auth/register"
        wrk.headers["Content-Type"] = "application/json"
        wrk.body = register_body
    end
end
