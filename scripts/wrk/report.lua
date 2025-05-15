-- 多线程共享的模拟数据
tokens = {
    "Bearer token_user_1",
    "Bearer token_user_2",
    "Bearer token_user_3",
    "Bearer token_user_4"
}

users = {
    {id = "42", name = "顾娟"},
    {id = "43", name = "张伟"},
    {id = "44", name = "李雷"},
    {id = "45", name = "韩梅梅"}
}

local api_url = "/v1/audit/report"

-- 用于给每个线程分配不同数据
counter = 0
thread_data = {}  -- key 为线程对象

-- init：所有线程运行前调用一次
init = function(args)
    print("init: 启动测试准备完毕")
end

-- 每个线程启动时调用一次
setup = function(thread)
    local index = counter
    counter = counter + 1

    local token = tokens[(index % #tokens) + 1]
    local user = users[(index % #users) + 1]

    thread_data[thread] = {
        user = user,
        token = token
    }

    print(string.format("线程 #%d 初始化: 用户=%s, token=%s", index, user.name, token))
end

-- 每次请求前调用
request = function()
    local user = "lisi"
    local token = "Bearer token_user_1"

    local body = string.format([[
    {
        "tenant_id": "97",
        "user_id": "%s",
        "username": "%s",
        "action": "amet fugiat officia",
        "resource_type": "eu Lorem",
        "resource_id": "%d",
        "resource_name": "比转系号",
        "result": "commodo qui anim ea",
        "message": "ad commodo cillum adipisicing sed",
        "client_ip": "137.88.%d.%d",
        "module": "ea id laboris est",
        "trace_id": "%d"
    }
    ]],
        user.id,
        user.name,
        math.random(1, 1000),
        math.random(0, 255),
        math.random(0, 255),
        math.random(1000, 9999)
    )

    -- 设置 Header
    wrk.headers["Content-Type"] = "application/json"
    wrk.headers["Authorization"] = token
    wrk.headers["X-Custom-Header"] = "my-custom-value"

    return wrk.format("POST", api_url, nil, body)
end

-- 每次响应返回后调用
response = function(status, headers, body)
    if status ~= 200 then
        print(string.format("请求失败 - 状态码: %d", status))
    end
end
