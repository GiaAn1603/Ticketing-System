local SUCCESS = 1
local ALREADY_PROCESSED = 2

local ERR_INVALID_INPUT = -1
local ERR_LIMIT_EXCEEDED = -2
local ERR_OUT_OF_STOCK = -3
local ERR_EVENT_NOT_FOUND = -4

local stock_key = KEYS[1]
local limit_key = KEYS[2]
local user_history_key = KEYS[3]
local request_key = KEYS[4]

local buy_quantity = tonumber(ARGV[1] or "0")
local ttl_seconds = tonumber(ARGV[2] or "86400")

if buy_quantity <= 0 then
  return ERR_INVALID_INPUT
end

if redis.call("EXISTS", request_key) == 1 then
  return ALREADY_PROCESSED
end

local stock = redis.call("GET", stock_key)
stock = tonumber(stock)

if not stock then
  return ERR_EVENT_NOT_FOUND
end

if stock < buy_quantity then
  return ERR_OUT_OF_STOCK
end

local max_limit = tonumber(redis.call("GET", limit_key) or "1")
local current_bought = tonumber(redis.call("GET", user_history_key) or "0")

if current_bought + buy_quantity > max_limit then
  return ERR_LIMIT_EXCEEDED
end

redis.call("DECRBY", stock_key, buy_quantity)
redis.call("INCRBY", user_history_key, buy_quantity)

if redis.call("TTL", user_history_key) < 0 then
  redis.call("EXPIRE", user_history_key, ttl_seconds)
end

redis.call("SET", request_key, 1, "EX", ttl_seconds)

return SUCCESS
