local SUCCESS = 1

local ERR_INVALID_INPUT = -1
local ERR_LIMIT_EXCEEDED = -2

local rate_key = KEYS[1]

local capacity = tonumber(ARGV[1] or "0")
local rate = tonumber(ARGV[2] or "0")
local requested = tonumber(ARGV[3] or "1")

if capacity <= 0 or rate <= 0 or requested <= 0 then
  return { ERR_INVALID_INPUT, 0 }
end

local redis_time = redis.call("TIME")
local now = tonumber(redis_time[1])

local info = redis.call("HMGET", rate_key, "tokens", "last_refreshed")
local tokens = tonumber(info[1])
local last_refreshed = tonumber(info[2])

if not tokens or not last_refreshed then
  tokens = capacity
  last_refreshed = now
end

local time_passed = math.max(0, now - last_refreshed)
tokens = math.min(capacity, tokens + (time_passed * rate))

if tokens >= requested then
  tokens = tokens - requested
  redis.call("HSET", rate_key, "tokens", tokens, "last_refreshed", now)

  local ttl_seconds = math.ceil(capacity / rate) + 2
  redis.call("EXPIRE", rate_key, ttl_seconds)

  return { SUCCESS, tokens }
end

return { ERR_LIMIT_EXCEEDED, tokens }
