package market

import "errors"

var (
	ErrMarketNotOpen       = errors.New("market is not opened")
	ErrInvalidQuotedGain   = errors.New("invalid quoted gain")
	ErrInvalidBetAmount    = errors.New("invalid bet amount")
	ErrMarketNotFound      = errors.New("market not found")
	ErrNoBetBinaryMarket   = errors.New("cannot place no bet on binary market")
	ErrOutcomeNotFound     = errors.New("failed to find provided outcome")
	ErrBetAlreadyPlaced    = errors.New("bet already placed")
	ErrBetNotFound         = errors.New("bet not found")
	ErrBetAlreadyCashedOut = errors.New("bet already cashed out")
)
