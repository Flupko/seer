package market

import (
	"errors"
	"fmt"

	"github.com/ericlagergren/decimal"
)

var ctx = decimal.Context{
	Precision:     30,
	RoundingMode:  decimal.ToNegativeInf,
	OperatingMode: decimal.GDA,
	Traps:         ^(decimal.Inexact | decimal.Subnormal),
}

const capPPM = 950_000 // 0.95 maximum price a share can attain, before being capped
var capMarket = decimal.New(capPPM, 6)

// README
// README
// README
// README
// README
// README
// README
// ------- 1 SHARE = 1 CENT (NOT 1 USDT) --------------------
// README
// README
// README
// README
// README
// README

func ComputeBDec(q []int64, alphaPPM int64) (*decimal.Big, error) {

	if len(q) == 0 {
		return nil, errors.New("empty q vector")
	}

	if alphaPPM <= 0 {
		return nil, errors.New("alpha must be > 0")
	}

	sum := decimal.New(0, 0)

	for _, qi := range q {

		if qi < 0 {
			return nil, errors.New("number of shares can't be negative")
		}

		ctx.Add(sum, sum, decimal.New(qi, 0))

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to cumulate sum qi: %w", err)
		}
	}

	// to prevent division by zero, if sum(q) is zero, b = 0
	if sum.Sign() == 0 {
		return nil, errors.New("sum(q) is equal to zero")
	}

	// b = alpha * sum, with alpha = alphaPPM * 10^-6
	alpha := decimal.New(alphaPPM, 6)
	var b decimal.Big
	ctx.Mul(&b, alpha, sum)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to compute b: %w", err)
	}

	// Edge cases
	if b.Sign() <= 0 {
		return nil, fmt.Errorf("computed b is <= 0")
	}

	return &b, nil

}

func Cost(q []int64, alphaPPM int64) (int64, error) {

	if len(q) == 0 {
		return 0, errors.New("empty q vector")
	}

	if alphaPPM <= 0 {
		return 0, errors.New("alpha must be > 0")
	}

	var maxqiI int64 = -1

	for _, qi := range q {

		if qi < 0 {
			return 0, errors.New("number of shares can't be negative")
		}

		if qi > maxqiI {
			maxqiI = qi
		}

	}

	b, err := ComputeBDec(q, alphaPPM)
	if err != nil {
		return 0, fmt.Errorf("ComputeBDec failed: %w", err)
	}

	// xi = qi/b
	// Compute max(xi)
	maxXi := decimal.New(maxqiI, 0)
	ctx.Quo(maxXi, maxXi, b)
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to compute maxXi: %w", err)
	}

	sumExp := decimal.New(0, 0)
	for _, qi := range q {

		// xi = qi / b
		var xi decimal.Big
		ctx.Quo(&xi, decimal.New(qi, 0), b)

		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("failed to compute xi quotient: %w", err)
		}

		// d = xi - max(xi)
		var d decimal.Big

		// no overflow on exp
		ctx.Sub(&d, &xi, maxXi)
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("failed to compute d = xi - max(xi): %w", err)
		}

		// e = exp(d)
		var e decimal.Big
		ctx.Exp(&e, &d)

		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("failed to compute exp(d): %w", err)
		}

		ctx.Add(sumExp, sumExp, &e)
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("failed to compute sum(exp(d)): %w", err)
		}

	}

	// sumExp must be > 0
	if sumExp.Sign() <= 0 {
		return 0, errors.New("sum of exps is <= 0")
	}

	var lnSum decimal.Big
	ctx.Log(&lnSum, sumExp)

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to compute ln(sumExp): %w", err)
	}

	var tmp decimal.Big
	ctx.Add(&tmp, maxXi, &lnSum)
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to compute maxXi + ln(sumExp): %w", err)
	}

	var costD decimal.Big
	ctx.Mul(&costD, b, &tmp) // costD = bD * (maxXiD + ln(sumExpD))
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to compute cost: %w", err)
	}

	var flooredCost decimal.Big
	ctx.Floor(&flooredCost, &costD)

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to floor cost: %w", err)
	}

	costI, ok := flooredCost.Int64()
	if !ok {
		return 0, fmt.Errorf("error converting cost to int64")
	}

	return costI, nil

}

const (
	maxHi = 1 << 50
)

// Quote returns the maximum number of shares that can be bought for an outcome with a certain budget
func Quote(q []int64, alphaPPM int64, budgetCents int64, idx int) (int64, error) {

	if budgetCents <= 0 {
		return 0, errors.New("budget must be > 0")
	}

	if idx < 0 || idx >= len(q) {
		return 0, errors.New("idx is out of range")
	}

	baseCost, err := Cost(q, alphaPPM)
	if err != nil {
		return 0, err
	}

	var lo int64 = 0
	var hi int64 = 1

	// Create qnext
	qNext := make([]int64, len(q))
	copy(qNext, q)

	for {
		qNext[idx] = q[idx] + hi
		nextCost, err := Cost(qNext, alphaPPM)
		if err != nil {
			return 0, fmt.Errorf("failed to compute cost: %w", err)
		}

		delta := nextCost - baseCost

		if delta >= budgetCents {
			break
		}

		hi *= 2

		if hi > maxHi {
			return 0, fmt.Errorf("overflow: hi > maxHi")
		}

	}

	for lo < hi {
		mid := (lo + hi + 1) / 2
		qNext[idx] = mid + q[idx]

		nextCost, err := Cost(qNext, alphaPPM)
		if err != nil {
			return 0, fmt.Errorf("failed to compute cost: %w", err)
		}

		delta := nextCost - baseCost

		if delta > budgetCents {
			hi = mid - 1
		} else {
			lo = mid
		}

	}

	// Verification to be sure that quoted amount of shares fits in the budget
	for k := lo; k >= 0; k-- {
		qNext := make([]int64, len(q))
		copy(qNext, q)
		qNext[idx] += k

		nextCost, err := Cost(qNext, alphaPPM)
		if err != nil {
			return 0, fmt.Errorf("failed to compute cost: %w", err)
		}

		delta := nextCost - baseCost
		if delta <= budgetCents {
			return k, nil
		}
	}

	return 0, nil

}

// MaxSharesToPriceCap returns the maximum number of shares buyable for an outcome such after buying those shares
// the outcome's price is < (STRICTLY LESS) than capD
func MaxSharesToPriceCap(q []int64, alphaPPM int64, idx int, cap *decimal.Big) (int64, error) {

	if len(q) == 0 {
		return 0, errors.New("empty q vector")
	}

	if idx < 0 || idx >= len(q) {
		return 0, errors.New("idx is out of range")
	}

	if alphaPPM <= 0 {
		return 0, errors.New("alpha must be > 0")
	}

	pricesInitDec, err := PricesDec(q, alphaPPM)

	if err != nil {
		return 0, fmt.Errorf("PricesDec failed: %w", err)
	}

	// If price is already greater than or equal to capD return 0
	if pricesInitDec[idx].Cmp(cap) >= 0 {
		return 0, nil
	}

	var lo int64 = 0
	var hi int64 = 1

	for {
		qNext := make([]int64, len(q))
		copy(qNext, q)
		qNext[idx] += hi

		nextPrices, err := PricesDec(qNext, alphaPPM)

		if err != nil {
			return 0, fmt.Errorf("PricesDec failed: %w", err)
		}

		if nextPrices[idx].Cmp(cap) >= 0 {
			break
		}

		hi *= 2

		if hi > maxHi {
			return 0, fmt.Errorf("overflow: hi > maxHi")
		}
	}

	for lo < hi {
		mid := (lo + hi + 1) / 2

		qNext := make([]int64, len(q))
		copy(qNext, q)
		qNext[idx] += mid

		nextPrices, err := PricesDec(qNext, alphaPPM)

		if err != nil {
			return 0, fmt.Errorf("PricesDec failed: %w", err)
		}

		if nextPrices[idx].Cmp(cap) >= 0 {
			hi = mid - 1
		} else {
			lo = mid
		}

	}

	return lo, nil

}

func PricesDec(q []int64, alphaPPM int64) ([]*decimal.Big, error) {

	if len(q) == 0 {
		return nil, errors.New("empty q vector")
	}

	if alphaPPM <= 0 {
		return nil, errors.New("alpha must be > 0")
	}

	n := len(q)

	b, err := ComputeBDec(q, alphaPPM)
	if err != nil {
		return nil, fmt.Errorf("ComputeBDec failed: %w", err)
	}

	alphaD := decimal.New(alphaPPM, 6)

	s, err := SoftmaxB(q, b)
	if err != nil {
		return nil, fmt.Errorf("ComputeBDec failed: %w", err)
	}

	if len(s) != len(q) {
		return nil, errors.New("q and sD of different length")
	}

	// Compute sum( si log(si) )
	sumSi := decimal.New(0, 0)
	for _, si := range s {

		if si == nil || si.Sign() <= 0 || si.Cmp(decimal.New(1, 0)) > 0 {
			return nil, fmt.Errorf("softmax not valid")
		}

		var lnSi decimal.Big
		ctx.Log(&lnSi, si)

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute ln(si): %w", err)
		}

		// si ln(si)
		var x decimal.Big
		ctx.Mul(&x, si, &lnSi)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute product si * ln(si): %w", err)
		}

		ctx.Add(sumSi, sumSi, &x)

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to cumulate sum[si log(si)]: %w", err)
		}

	}

	// Negative, give H(s) entropy term
	ctx.Neg(sumSi, sumSi)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to negate sum[si log(si)]: %w", err)
	}

	// Multiply alpha
	// alpha * H(s)
	var com decimal.Big
	ctx.Mul(&com, alphaD, sumSi)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("alpha * H(s): %w", err)
	}

	// Vector of prices pi
	p := make([]*decimal.Big, n)

	// Compute price for each i
	for i := range n {

		var pi decimal.Big
		ctx.Add(&pi, s[i], &com)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to si * com: %w", err)
		}

		p[i] = &pi
	}

	return p, nil

}

// Returns (odds, active) in Parts per Hundreds rounded down, ie 2.529 OUTPUTS => 252
func OddsDecPPH(q []int64, alphaPPM, feePPM int64) ([]int64, []bool, error) {

	if len(q) == 0 {
		return nil, nil, errors.New("empty q vector")
	}

	if alphaPPM <= 0 {
		return nil, nil, errors.New("alpha must be > 0")
	}

	if feePPM < 0 || feePPM >= 1_000_000 {
		return nil, nil, errors.New("feePPM must be in between 0 and 1_000_000")
	}

	// Prices vector in decimal
	p, err := PricesDec(q, alphaPPM)

	if err != nil {
		return nil, nil, fmt.Errorf("PricesDec failed: %w", err)
	}

	if len(q) != len(p) {
		return nil, nil, errors.New("q and pD of different length")
	}

	// Odds = inverse of prices

	// Compute fee applied
	fee := decimal.New(feePPM, 6)

	// For each outcome we have the following :
	// odd = (1 / pi) * (1 - fee)
	var oneMinFee decimal.Big
	ctx.Sub(&oneMinFee, decimal.New(1, 0), fee)
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to compute (1 - fee): %w", err)
	}

	n := len(q)
	oddsPPH_I := make([]int64, n)
	active := make([]bool, n)

	for i, pi := range p {

		if pi.Sign() <= 0 {
			return nil, nil, errors.New("non positive price")
		}

		// Active q only if price < cap price
		active[i] = pi.Cmp(capMarket) == -1

		// invPiD =  (1 / piD)
		var invPi decimal.Big
		ctx.Quo(&invPi, decimal.New(1, 0), pi)
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("failed to compute inv of pi: %w", err)
		}

		// oddD = invPiD * oneMinFee = (1/pi) * (1-fee)
		var odd decimal.Big
		ctx.Mul(&odd, &invPi, &oneMinFee)
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("failed to multiply baseD * oneMinFeeD: %w", err)
		}

		// Multiply by 100 to keep in PPH
		var scaledOdd decimal.Big
		ctx.Mul(&scaledOdd, &odd, decimal.New(100, 0))
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("failed to scaled odd: %w", err)
		}

		// Floor, explicit RoundDown

		var floorScaledOdd decimal.Big

		ctx.Floor(&floorScaledOdd, &scaledOdd)
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("failed to floor scaled odd: %w", err)
		}

		oddPPH_I, ok := floorScaledOdd.Int64()
		if !ok {
			return nil, nil, fmt.Errorf("error converting scaled odd to int64")
		}

		oddsPPH_I[i] = oddPPH_I

	}

	return oddsPPH_I, active, nil

}

// OddAndGainFromBudget applies fees to the budget, buys shares, and returns (gainCents, feeCents, oddPPH)
// budgetCents - gainCents gives the market maker's fee
// oddPPH is floor( ((budget-fee)/gain) * 100 ) -> Purely "indicative", DO NOT USE for business logic
func OddAndGainFromBudget(q []int64, alphaPPM int64, feePPM int64, budgetCents int64, idx int) (int64, int64, int64, error) {

	if len(q) == 0 {
		return 0, 0, 0, errors.New("empty q vector")
	}

	if alphaPPM <= 0 {
		return 0, 0, 0, errors.New("alpha must be > 0")
	}

	if feePPM < 0 || feePPM >= 1_000_000 {
		return 0, 0, 0, errors.New("feePPM must be between 0 and 1_000_000")
	}

	if budgetCents <= 0 {
		return 0, 0, 0, errors.New("budget must be positive")
	}

	feeCents, err := FeeFromBudget(budgetCents, feePPM)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to compute fee: %w", err)
	}

	availBudgetCents := budgetCents - feeCents

	// maxShares = gainCents
	gainFromBudgetCents, err := Quote(q, alphaPPM, availBudgetCents, idx)

	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to Quote: %w", err)
	}

	if gainFromBudgetCents <= 0 {
		return 0, 0, 0, errors.New("quoted gain is not positive")
	}

	capCents, err := MaxSharesToPriceCap(q, alphaPPM, idx, capMarket)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to compute MaxSharesToPriceCap: %w", err)
	}

	gainCents := min(capCents, gainFromBudgetCents)

	oddPPH_I, err := ComputeOddDecPPH(budgetCents, gainCents)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error ComputeOddDecPPH: %w", err)
	}

	return gainCents, feeCents, oddPPH_I, nil

}
