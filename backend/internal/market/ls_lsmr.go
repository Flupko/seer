package market

import (
	"errors"
	"fmt"
	"seer/internal/numeric"

	"github.com/ericlagergren/decimal"
)

var ctx = decimal.Context{
	Precision:     30,
	RoundingMode:  decimal.ToZero,
	OperatingMode: decimal.GDA,
	Traps:         ^(decimal.Inexact | decimal.Subnormal),
}

var minShares = *decimal.New(1, 2) // 0.1 share minimum to buy
var zeroDec = *decimal.New(0, 0)

// README
// README
// README
// README
// README
// README
// README
// ------- 1 SHARE = 1 USDT --------------------
// README
// README
// README
// README
// README
// README

func ComputeBDec(q []decimal.Big, alpha decimal.Big) (decimal.Big, error) {

	if len(q) == 0 {
		return zeroDec, errors.New("empty q vector")
	}

	if alpha.Sign() <= 0 {
		return zeroDec, errors.New("alpha must be > 0")
	}

	sum := decimal.New(0, 0)

	for _, qi := range q {

		if qi.Sign() < 0 {
			return zeroDec, errors.New("number of shares can't be negative")
		}

		ctx.Add(sum, sum, &qi)

		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to cumulate sum qi: %w", err)
		}
	}

	// to prevent division by zero, if sum(q) is zero, b = 0
	if sum.Sign() == 0 {
		return zeroDec, errors.New("sum(q) is equal to zero")
	}

	var b decimal.Big
	ctx.Mul(&b, &alpha, sum)
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute b: %w", err)
	}

	// Edge cases
	if b.Sign() <= 0 {
		return zeroDec, fmt.Errorf("computed b is <= 0")
	}

	return b, nil

}

func Cost(q []decimal.Big, alpha decimal.Big) (decimal.Big, error) {

	if len(q) == 0 {
		return zeroDec, errors.New("empty q vector")
	}

	if alpha.Sign() <= 0 {
		return zeroDec, errors.New("alpha must be > 0")
	}

	maxQi := decimal.New(0, 0)
	for _, qi := range q {

		if qi.Sign() < 0 {
			return zeroDec, errors.New("number of shares can't be negative")
		}

		if qi.Cmp(maxQi) > 0 {
			maxQi.Set(&qi)
		}
	}

	// Compute b
	b, err := ComputeBDec(q, alpha)
	if err != nil {
		return zeroDec, fmt.Errorf("ComputeBDec failed: %w", err)
	}

	// xi = qi/b
	// Compute max(xi)
	var maxXi decimal.Big
	ctx.Quo(&maxXi, maxQi, &b)
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute maxXi: %w", err)
	}

	sumExp := decimal.New(0, 0)
	for _, qi := range q {

		// xi = qi / b
		var xi decimal.Big
		ctx.Quo(&xi, &qi, &b)

		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute xi quotient: %w", err)
		}

		// d = xi - max(xi)
		var d decimal.Big

		// no overflow on exp
		ctx.Sub(&d, &xi, &maxXi)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute d = xi - max(xi): %w", err)
		}

		// e = exp(d)
		var e decimal.Big
		ctx.Exp(&e, &d)

		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute exp(d): %w", err)
		}

		ctx.Add(sumExp, sumExp, &e)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute sum(exp(d)): %w", err)
		}

	}

	// sumExp must be > 0
	if sumExp.Sign() <= 0 {
		return zeroDec, errors.New("sum of exps is <= 0")
	}

	var lnSum decimal.Big
	ctx.Log(&lnSum, sumExp)

	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute ln(sumExp): %w", err)
	}

	var tmp decimal.Big
	ctx.Add(&tmp, &maxXi, &lnSum)
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute maxXi + ln(sumExp): %w", err)
	}

	var cost decimal.Big
	ctx.Mul(&cost, &b, &tmp) // costD = b * (maxXi + ln(sumExp))
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute cost: %w", err)
	}

	return cost, nil

}

var maxHi = decimal.New(1<<59, 0)

func Bisect(lo, hi, tol decimal.Big, f func(x decimal.Big) (decimal.Big, error)) (decimal.Big, error) {

	if lo.Cmp(&hi) >= 0 {
		return zeroDec, errors.New("lo must be < hi")
	}

	if tol.Sign() <= 0 {
		return zeroDec, errors.New("tolerance must be > 0")
	}

	for {

		var delta decimal.Big
		ctx.Sub(&delta, &hi, &lo)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute delta: %w", err)
		}

		// Check if hi - lo <= tol
		if delta.Cmp(&tol) <= 0 {
			break
		}

		var mid decimal.Big
		ctx.Add(&mid, &lo, &hi)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute sum of lo + hi: %w", err)
		}

		ctx.Quo(&mid, &mid, decimal.New(2, 0))
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute mid/2: %w", err)
		}

		fMid, err := f(mid)
		if err != nil {
			return zeroDec, fmt.Errorf("failed to compute f(mid): %w", err)
		}

		if fMid.Sign() > 0 {
			hi = mid
		} else {
			lo = mid
		}
	}

	return lo, nil

}

// GetMaxSharesCanBuy returns the maximum number of shares that can be bought for an outcome with a certain budget
func GetMaxSharesCanBuy(q []decimal.Big, idx int, alpha, budget decimal.Big) (decimal.Big, error) {

	if budget.Sign() <= 0 {
		return zeroDec, errors.New("budget must be > 0")
	}

	if idx < 0 || idx >= len(q) {
		return zeroDec, errors.New("idx is out of range")
	}

	if alpha.Sign() <= 0 {
		return zeroDec, errors.New("alpha must be > 0")
	}

	baseCost, err := Cost(q, alpha)
	if err != nil {
		return zeroDec, err
	}

	// Compute the initial hi as budget / price_i
	prices, err := PricesDec(q, alpha)
	if err != nil {
		return zeroDec, fmt.Errorf("PricesDec failed: %w", err)
	}

	lo := zeroDec
	var hi decimal.Big
	ctx.Quo(&hi, &budget, &prices[idx])
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute initial lo: %w", err)
	}

	// Create qnext
	qNext := make([]decimal.Big, len(q))
	copy(qNext, q)

	// Prepare bisection
	f := func(deltaShares decimal.Big) (decimal.Big, error) {

		ctx.Add(&qNext[idx], &q[idx], &deltaShares)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute qNext[idx]: %w", err)
		}

		nextCost, err := Cost(qNext, alpha)
		if err != nil {
			return zeroDec, fmt.Errorf("failed to compute nextCost: %w", err)
		}

		var res decimal.Big
		ctx.Sub(&res, &nextCost, &baseCost)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute nextCost - baseCost: %w", err)
		}

		ctx.Sub(&res, &res, &budget)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute nextCost - baseCost - budget: %w", err)
		}

		return res, nil

	}

	nbSharesCanBuy, err := Bisect(lo, hi, *decimal.New(1, numeric.Scale), f)
	if err != nil {
		return zeroDec, fmt.Errorf("Bisect failed: %w", err)
	}

	// Round the result
	nbSharesCanBuy, err = TruncatePrecision(nbSharesCanBuy, numeric.Scale, decimal.ToZero)
	if err != nil {
		return zeroDec, fmt.Errorf("failed to truncate lo: %w", err)
	}

	return nbSharesCanBuy, nil
}

// MaxSharesToPriceCap returns the maximum number of shares buyable for an outcome such after buying those shares
// the outcome's price is < (STRICTLY LESS) than capD
func MaxSpendToPriceCap(q []decimal.Big, idx int, alpha, cap decimal.Big) (decimal.Big, error) {

	if len(q) == 0 {
		return zeroDec, errors.New("empty q vector")
	}

	if idx < 0 || idx >= len(q) {
		return zeroDec, errors.New("idx is out of range")
	}

	if alpha.Sign() <= 0 {
		return zeroDec, errors.New("alpha must be > 0")
	}

	pricesInit, err := PricesDec(q, alpha)

	if err != nil {
		return zeroDec, fmt.Errorf("PricesDec failed: %w", err)
	}

	// If price is already greater than or equal to capD return 0
	if pricesInit[idx].Cmp(&cap) >= 0 {
		out := decimal.New(0, 0)
		return *out, nil
	}

	var lo, hi decimal.Big

	lo.Set(&minShares)
	hi.Set(&minShares)

	qNext := make([]decimal.Big, len(q))
	copy(qNext, q)

	for {

		ctx.Add(&qNext[idx], &q[idx], &hi)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute qNext[idx]: %w", err)
		}

		nextPrices, err := PricesDec(qNext, alpha)
		if err != nil {
			return zeroDec, fmt.Errorf("failed to compute nextPirces: %w", err)
		}

		if nextPrices[idx].Cmp(&cap) >= 0 {
			break
		}

		hi.Mul(&hi, decimal.New(2, 0))
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to double hi: %w", err)
		}

		if hi.Cmp(maxHi) > 0 {
			return zeroDec, fmt.Errorf("overflow: hi > maxHi")
		}
	}

	// If lo == hi, return 0 because buying anything >= minShares would put us over the cap
	if lo.Cmp(&hi) == 0 {
		return zeroDec, nil
	}

	f := func(deltaShares decimal.Big) (decimal.Big, error) {
		ctx.Add(&qNext[idx], &q[idx], &deltaShares)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute qNext[idx]: %w", err)
		}

		nextPrices, err := PricesDec(qNext, alpha)
		if err != nil {
			return zeroDec, fmt.Errorf("failed to compute nextPirces: %w", err)
		}

		var res decimal.Big
		ctx.Sub(&res, &nextPrices[idx], &cap)
		if err := ctx.Err(); err != nil {
			return zeroDec, fmt.Errorf("failed to compute nextPrices[idx] - cap: %w", err)
		}

		return res, nil
	}

	deltaSharesCap, err := Bisect(lo, hi, *decimal.New(1, numeric.Scale), f)
	if err != nil {
		return zeroDec, fmt.Errorf("Bisect failed: %w", err)
	}

	// Compute corresponding spend
	baseCost, err := Cost(q, alpha)
	if err != nil {
		return zeroDec, err
	}

	qNext = make([]decimal.Big, len(q))
	copy(qNext, q)

	ctx.Add(&qNext[idx], &q[idx], &deltaSharesCap)
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute q[idx] + deltaSharesCap: %w", err)
	}

	nextCost, err := Cost(qNext, alpha)
	if err != nil {
		return zeroDec, err
	}

	var maxSpend decimal.Big
	ctx.Sub(&maxSpend, &nextCost, &baseCost)
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute nextCost - baseCost: %w", err)
	}

	return maxSpend, nil
}

func PricesDec(q []decimal.Big, alpha decimal.Big) ([]decimal.Big, error) {

	if len(q) == 0 {
		return nil, errors.New("empty q vector")
	}

	if alpha.Sign() <= 0 {
		return nil, errors.New("alpha must be > 0")
	}

	n := len(q)

	b, err := ComputeBDec(q, alpha)
	if err != nil {
		return nil, fmt.Errorf("ComputeBDec failed: %w", err)
	}

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

		if si.Sign() <= 0 || si.Cmp(decimal.New(1, 0)) > 0 {
			return nil, fmt.Errorf("softmax not valid")
		}

		var lnSi decimal.Big
		ctx.Log(&lnSi, &si)

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute ln(si): %w", err)
		}

		// si ln(si)
		var x decimal.Big
		ctx.Mul(&x, &si, &lnSi)
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
	ctx.Mul(&com, &alpha, sumSi)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("alpha * H(s): %w", err)
	}

	// Vector of prices pi
	p := make([]decimal.Big, n)

	// Compute price for each i
	for i := range n {

		var pi decimal.Big
		ctx.Add(&pi, &s[i], &com)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to si * com: %w", err)
		}

		p[i] = pi
	}

	return p, nil

}

// Returns prices in BigDecimal
func PricesBD(qBD []*numeric.BigDecimal, alphaBD, feeBD *numeric.BigDecimal) ([]*numeric.BigDecimal, error) {

	if len(qBD) == 0 {
		return nil, errors.New("empty q vector")
	}

	if alphaBD.Sign() <= 0 {
		return nil, errors.New("alpha must be > 0")
	}

	alpha := alphaBD.Big
	fee := feeBD.Big
	q := make([]decimal.Big, 0, len(qBD))
	for _, qiBD := range qBD {
		q = append(q, qiBD.Big)
	}

	// Prices vector in eric../decimal
	p, err := PricesDec(q, alpha)

	if err != nil {
		return nil, fmt.Errorf("PricesDec failed: %w", err)
	}

	if len(q) != len(p) {
		return nil, errors.New("q and pD of different length")
	}

	// For each outcome we have the following :
	// price_final = pi * (1 - fee)
	var oneMinFee decimal.Big
	ctx.Sub(&oneMinFee, decimal.New(1, 0), &fee)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to compute (1 - fee): %w", err)
	}

	n := len(q)
	prices := make([]*numeric.BigDecimal, n)

	for i, pi := range p {

		if pi.Sign() <= 0 {
			return nil, errors.New("non positive price")
		}

		// price_final = (pi * oneMinFee) * 10^6
		var pf decimal.Big
		ctx.Mul(&pf, &pi, &oneMinFee)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to multiply pf * oneMinFee: %w", err)
		}

		pfRounded, err := TruncatePrecision(pf, numeric.Scale, decimal.AwayFromZero)
		if err != nil {
			return nil, fmt.Errorf("failed to truncate pf: %w", err)
		}

		var pfBD numeric.BigDecimal
		pfBD.Big = pfRounded
		prices[i] = &pfBD

	}

	return prices, nil

}

// PriceAndGainFromBudget applies fees to the budget, buys shares, and returns (gain, feePaid, price)
// price is ceil of the average price shares are bought = (gain / budget)
// = Purely "indicative", DO NOT USE for business logic
func PossibleGainFeePriceForBuy(qBD []*numeric.BigDecimal, idx int, alphaBD, feeBD, budgetBD, capBD *numeric.BigDecimal) (bool, *numeric.BigDecimal, *numeric.BigDecimal, *numeric.BigDecimal, error) {

	if len(qBD) == 0 {
		return false, nil, nil, nil, errors.New("empty q vector")
	}

	if alphaBD.Sign() <= 0 {
		return false, nil, nil, nil, errors.New("alpha must be > 0")
	}

	if budgetBD.Sign() <= 0 {
		return false, nil, nil, nil, errors.New("budget must be positive")
	}

	budget := budgetBD.Big
	fee := feeBD.Big
	alpha := alphaBD.Big
	capPrice := capBD.Big
	q := make([]decimal.Big, 0, len(qBD))
	for _, qiBD := range qBD {
		q = append(q, qiBD.Big)
	}

	feePaid, err := FeeFromBudget(budget, fee)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("failed to compute fee: %w", err)
	}

	var availBudget decimal.Big
	ctx.Sub(&availBudget, &budget, &feePaid)
	if err := ctx.Err(); err != nil {
		return false, nil, nil, nil, fmt.Errorf("failed to compute availBudget: %w", err)
	}

	maxSpendCap, err := MaxSpendToPriceCap(q, idx, alpha, capPrice)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("failed to compute MaxSharesToPriceCap: %w", err)
	}

	if maxSpendCap.Cmp(&availBudget) <= 0 {
		return false, nil, nil, nil, nil
	}

	// maxShares = gain
	gainFromBudget, err := GetMaxSharesCanBuy(q, idx, alpha, availBudget)

	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("failed to Quote: %w", err)
	}

	if gainFromBudget.Sign() <= 0 {
		return false, nil, nil, nil, errors.New("quoted gain is not positive")
	}

	// avg price
	avgPrice, err := computeAvgPrice(budget, gainFromBudget)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("failed to compute average price: %w", err)
	}

	// Round down to 12 decimal places
	avgPriceRounded, err := TruncatePrecision(avgPrice, numeric.Scale, decimal.AwayFromZero)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("failed to truncate avgPrice: %w", err)
	}

	gainRounded, err := TruncatePrecision(gainFromBudget, numeric.Scale, decimal.ToZero)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("failed to truncate gain: %w", err)
	}

	return true, &numeric.BigDecimal{Big: gainRounded}, &numeric.BigDecimal{Big: feePaid}, &numeric.BigDecimal{Big: avgPriceRounded}, nil

}

func PossibleGainForSell(qBD []*numeric.BigDecimal, idxBought int, alphaBD, nbSharesBoughtBD, capBD *numeric.BigDecimal) (bool, *numeric.BigDecimal, error) {

	if len(qBD) == 0 {
		return false, nil, errors.New("empty q vector")
	}

	if alphaBD.Sign() <= 0 {
		return false, nil, errors.New("alpha must be > 0")
	}

	if nbSharesBoughtBD.Sign() <= 0 {
		return false, nil, errors.New("nbSharesBought must be positive")
	}

	alpha := alphaBD.Big
	nbSharesBought := nbSharesBoughtBD.Big
	cap := capBD.Big
	q := make([]decimal.Big, 0, len(qBD))
	for _, qiBD := range qBD {
		q = append(q, qiBD.Big)
	}

	// Create qNext
	qHedge := make([]decimal.Big, len(q))
	copy(qHedge, q)

	// Buy nbSharesBought from opposite sides to "hedge" (simulate selling with LS-LMSR always moving forward logic)
	// IE with 2 outcomes, if you bought x shares (for $k) on outcome 0, we buy x shares to outcome 1 to hedge
	// Your profit is then x - delta C_hedge (how much it cost to hedge)

	for i := range len(qBD) {
		if i == idxBought {
			continue
		}
		ctx.Add(&qHedge[i], &q[i], &nbSharesBought)
	}

	// Check if after hedging, the price of the bought outcomes is >= cap
	pricesAfterHedge, err := PricesDec(qHedge, alpha)
	if err != nil {
		return false, nil, fmt.Errorf("failed to compute PricesDec after hedge: %w", err)
	}

	for i := range len(qBD) {
		if i == idxBought {
			continue
		}

		// Price goes above cap after hedge, can't sell
		if pricesAfterHedge[i].Cmp(&cap) >= 0 {
			return false, nil, nil
		}
	}

	baseCost, err := Cost(q, alpha)
	if err != nil {
		return false, nil, fmt.Errorf("failed to compute baseCost: %w", err)
	}

	hedgeCost, err := Cost(qHedge, alpha)
	if err != nil {
		return false, nil, fmt.Errorf("failed to compute hedgeCost: %w", err)
	}

	var deltaHedge decimal.Big
	ctx.Sub(&deltaHedge, &hedgeCost, &baseCost)
	if err := ctx.Err(); err != nil {
		return false, nil, fmt.Errorf("failed to compute deltaHedge: %w", err)
	}

	var gain decimal.Big
	ctx.Sub(&gain, &nbSharesBought, &deltaHedge)
	if err := ctx.Err(); err != nil {
		return false, nil, fmt.Errorf("failed to compute gain: %w", err)
	}

	gainRounded, err := TruncatePrecision(gain, numeric.Scale, decimal.ToZero)
	if err != nil {
		return false, nil, fmt.Errorf("failed to truncate gain: %w", err)
	}

	return true, &numeric.BigDecimal{Big: gainRounded}, nil

}
