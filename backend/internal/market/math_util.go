package market

import (
	"errors"
	"fmt"
	"seer/internal/numeric"

	"github.com/ericlagergren/decimal"
)

// FeeFromBudget returns ceil(budget * fee)
func FeeFromBudget(budget, fee decimal.Big) (decimal.Big, error) {

	if budget.Sign() < 0 {
		return zeroDec, errors.New("budget must be >= 0")
	}

	if fee.Cmp(decimal.New(0, 0)) < 0 || fee.Cmp(decimal.New(1, 0)) >= 0 {
		fmt.Println(fee.String())
		return zeroDec, errors.New("fee must be between 0 and 1 (exclusive)")
	}

	var feePaid decimal.Big
	ctx.Mul(&feePaid, &budget, &fee)
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute budget*fee: %w", err)
	}

	feePaidRoundedUp, err := TruncatePrecision(feePaid, numeric.Scale, decimal.AwayFromZero)
	if err != nil {
		return zeroDec, err
	}

	return feePaidRoundedUp, nil

}

// Softmax returns a slice containing the softmax of (q/b)
// It computes s_i = exp((q_i / b) - max_j(q_j / b)) / sum_k exp((q_k / b) - max_j(q_j / b))
func SoftmaxB(q []decimal.Big, b decimal.Big) ([]decimal.Big, error) {

	if len(q) == 0 {
		return nil, errors.New("empty q vector")
	}

	if b.Sign() <= 0 {
		return nil, errors.New("b must be > 0")
	}

	n := len(q)

	maxQi := decimal.New(0, 0)
	for _, qi := range q {

		if qi.Sign() < 0 {
			return nil, errors.New("number of shares can't be negative")
		}

		if qi.Cmp(maxQi) > 0 {
			maxQi.Set(&qi)
		}
	}

	var maxXi decimal.Big
	ctx.Quo(&maxXi, maxQi, &b)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to compute maxXi: %w", err)
	}

	// sum(exp(xi - max(xi)))
	sumExp := decimal.New(0, 0)
	// xi
	exps := make([]decimal.Big, n)

	for i, qi := range q {

		// xi = qi / b
		var xi decimal.Big
		ctx.Quo(&xi, &qi, &b)
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

		exps[i] = e

	}

	s := make([]decimal.Big, n)

	for i, ei := range exps {

		var si decimal.Big
		ctx.Quo(&si, &ei, sumExp)
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("failed to compute quotient in softmax(qi/b): %w", err)
		}

		s[i] = si
	}

	return s, nil

}

// TruncatePrecision truncates a decimal.Big to a certain number of decimals
func TruncatePrecision(d decimal.Big, scale int, roundingMode decimal.RoundingMode) (decimal.Big, error) {
	d.Context.RoundingMode = roundingMode
	d.Context.Precision = 30
	d.Quantize(scale)

	if err := d.Context.Err(); err != nil {
		fmt.Println("Error in TruncatePrecision:", err)
		return zeroDec, fmt.Errorf("failed to truncate precision: %w", err)
	}
	return d, nil
}

func computeAvgPrice(betAmount decimal.Big, gain decimal.Big) (decimal.Big, error) {

	if betAmount.Sign() <= 0 {
		return zeroDec, fmt.Errorf("negative bet amount")
	}

	if gain.Sign() <= 0 {
		return zeroDec, fmt.Errorf("negative payout")
	}

	// price = (betAmount / gain)
	var p decimal.Big
	ctx.Quo(&p, &betAmount, &gain)
	if err := ctx.Err(); err != nil {
		return zeroDec, fmt.Errorf("failed to compute gainCents/betAmountCents : %w", err)
	}

	return p, nil

}
