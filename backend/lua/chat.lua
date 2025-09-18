local message = ARGV[1]
local max_len = tonumber(ARGV[2])

local now_time_ms = tonumber(ARGV[3])

local capacity_bucket = tonumber(ARGV[4])
local rate_per_min = tonumber(ARGV[5])
local rate_per_ms = rate_per_min / 60000.0

local rate_user_key = KEYS[1]
local list_messages_key = KEYS[2]

local expire_rate_sec = tonumber(ARGV[6])
redis.call("EXPIRE", rate_user_key, expire_rate_sec)

local last_refill_time = tonumber(redis.call("HGET", rate_user_key, "last_refill_time") or tostring(now_time_ms))
local current_tokens = tonumber(redis.call("HGET", rate_user_key, "current_tokens") or tostring(capacity_bucket))

local elapsed_time_ms = now_time_ms - last_refill_time
local new_tokens = math.floor(elapsed_time_ms * rate_per_ms)

local tokens = math.min(capacity_bucket, current_tokens + new_tokens)

if tokens < 1 then
    return 0
end

tokens = tokens - 1

redis.call("HSET", rate_user_key, "last_refill_time", now_time_ms, "current_tokens", tokens)

-- add the new message
redis.call("LPUSH", list_messages_key, message)
redis.call("LTRIM", list_messages_key, 0, max_len - 1)

return 1

