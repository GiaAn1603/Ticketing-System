package scripts

import _ "embed"

//go:embed lua/rate_limit.lua
var RateLimitScript string

//go:embed lua/buy_ticket.lua
var BuyTicketScript string
