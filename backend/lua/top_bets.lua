local high_size = tonumber(ARGV[1])
local window_time = tonumber(ARGV[2])
local now_time = tonumber(ARGV[3])

local new_el_value = ARGV[4]
local new_el_wager = tonumber(ARGV[5])
local new_el_time = tonumber(ARGV[6])

local key_time = KEYS[1]
local key_wager = KEYS[2]

while redis.call("ZCARD", key_time) > high_size do
    local rm_el_cand = redis.call("ZRANGE", key_time, 0, 0, "WITHSCORES")
    local cand_val = rm_el_cand[1]
    local cand_time = tonumber(rm_el_cand[2])
    if now_time - cand_time < window_time then
        break
    end
    redis.call("ZREM", key_time, cand_val)
    redis.call("ZREM", key_wager, cand_val)
end

redis.call("ZADD", key_time, new_el_time, new_el_value)
redis.call("ZADD", key_wager, new_el_wager, new_el_value)

if redis.call("ZCARD", key_time) > high_size then
    local lowest_high = redis.call("ZREVRANGE", key_wager, high_size - 1, high_size - 1, "WITHSCORES")

    local threshold_value = lowest_high[1]
    local threshold_score = tonumber(lowest_high[2])

    local pivot_time = redis.call("ZSCORE", key_time, threshold_value)
    local els_to_rm = redis.call("ZRANGEBYSCORE", key_time, "-inf", pivot_time - 1)
    for _, value in ipairs(els_to_rm) do
        redis.call("ZREM", key_time, value)
        redis.call("ZREM", key_wager, value)
    end
end

local rank = redis.call("ZREVRANK", key_wager, new_el_value)
return tonumber(rank) < high_size
