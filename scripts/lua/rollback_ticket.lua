local SUCCESS = 1
local SKIPPED = 0

local stock_key = KEYS[1]
local user_history_key = KEYS[2]
local request_key = KEYS[3]

local rollback_quantity = tonumber(ARGV[1] or "0")

if redis.call("EXISTS", request_key) == 0 then
  return SKIPPED
end

redis.call("INCRBY", stock_key, rollback_quantity)
redis.call("DECRBY", user_history_key, rollback_quantity)
redis.call("DEL", request_key)

return SUCCESS
