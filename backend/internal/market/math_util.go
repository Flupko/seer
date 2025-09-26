package market

import (
	"errors"
	"fmt"

	"github.com/ericlagergren/decimal"
)

// FeeFromBudget returns ceil( (budget * feePPM) / 1e6)
func FeeFromBudget(budget, feePPM int64) (int64, error) {

	if budget < 0 {
		return 0, errors.New("budget must be >= 0")
	}

	if feePPM < 0 || feePPM >= 1_000_000 {
		return 0, errors.New("feePPM must be between 0 and 1_000_000")
	}

	b := decimal.New(budget, 0)
	fee := decimal.New(feePPM, 6)

	var prod, ceil decimal.Big
	ctx.Mul(&prod, b, fee)
	ctx.Ceil(&ceil, &prod)

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to ceil prod: %w", err)
	}

	ceilFeeInt, ok := ceil.Int64()

	if !ok {
		return 0, fmt.Errorf("error converting charged fee to int64")
	}

	return ceilFeeInt, nil

}

// Softmax returns a slice containing the softmax of (q/b)
// It computes s_i = exp((q_i / b) - max_j(q_j / b)) / sum_k exp((q_k / b) - max_j(q_j / b))
func SoftmaxB(q []int64, b *decimal.Big) ([]*decimal.Big, error) {

	if len(q) == 0 {
		return nil, errors.New("empty q vector")
	}

	if b == nil {
		return nil, errors.New("nil b")
	}

	if b.Sign() <= 0 {
		return nil, errors.New("b must be > 0")
	}

	n := len(q)

	var maxqiI int64 = -1
	for _, qi := range q {

		if qi < 0 {
			return nil, errors.New("number of shares can't be negative")
		}

		if qi > maxqiI {
			maxqiI = qi
		}

	}

	maxQi := decimal.New(maxqiI, 0)
	var maxXi decimal.Big
	ctx.Quo(&maxXi, maxQi, b)

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to compute maxXiD: %w", err)
	}

	// sum(exp(xi - max(xi)))
	sumExp := decimal.New(0, 0)
	// xi
	exps := make([]*decimal.Big, n)

	for i, qi := range q {

		// xi = qi / b
		var xi decimal.Big
		ctx.Quo(&xi, decimal.New(qi, 0), b)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute xi quotient: %w", err)
		}

		// d = xi - max(xi)
		var d decimal.Big
		ctx.Sub(&d, &xi, &maxXi)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute d = xi - max(xi): %w", err)
		}

		var e decimal.Big
		ctx.Exp(&e, &d)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute exp(d): %w", err)
		}

		ctx.Add(sumExp, sumExp, &e)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute sum(exp(d)): %w", err)
		}

		exps[i] = &e

	}

	s := make([]*decimal.Big, n)

	for i, ei := range exps {

		var si decimal.Big
		ctx.Quo(&si, ei, sumExp)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute quotient in softmax(qi/b): %w", err)
		}

		s[i] = &si
	}

	return s, nil

}

func ComputePricePPM(betAmountCents int64, gainCents int64) (int64, error) {

	if betAmountCents <= 0 {
		return 0, fmt.Errorf("negative bet amount")
	}

	if gainCents <= 0 {
		return 0, fmt.Errorf("negative payout")
	}

	// price = (betAmount / gain), then PPM = floor(price * 10^6)
	num := decimal.New(betAmountCents, 0)
	den := decimal.New(gainCents, 0)

	var p decimal.Big
	ctx.Quo(&p, num, den)
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to compute gainCents/betAmountCents : %w", err)
	}

	// Multiply by 10^6 to keep in PPM
	var scaledP decimal.Big
	ctx.Mul(&scaledP, &p, decimal.New(1, -6))
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to scale*10^6 price: %w", err)
	}

	var ceiledScaledP decimal.Big
	ctx.Ceil(&ceiledScaledP, &scaledP)
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("failed to ceil scaled price: %w", err)
	}

	p_PPH_I, ok := ceiledScaledP.Int64()
	if !ok {
		return 0, fmt.Errorf("error converting scaled price to int64")
	}

	return p_PPH_I, nil

}
