import { Decimal } from 'decimal.js';
import { MarketView } from '../definitions';
import { feeFromSpend, gainProportion, softmaxB, truncatePrecision } from './math_util';


export const SCALE = 12;
export const MIN_SHARES = new Decimal(0.1);

Decimal.set({ precision: 12, rounding: Decimal.ROUND_DOWN });

export function bisect(lo: Decimal, hi: Decimal, tol: Decimal, f: (x: Decimal) => Decimal): Decimal {

    while (hi.minus(lo).greaterThan(tol)) {
        const mid = lo.plus(hi).dividedBy(2);
        if (f(mid).greaterThan(0)) {
            hi = mid;
        } else {
            lo = mid;
        }
    }

    return lo
}

export function computeB(q: Decimal[], alpha: Decimal): Decimal {
    const sum = q.reduce((a, b) => a.plus(b), new Decimal(0));
    return Decimal.mul(alpha, sum);
}

export function cost(q: Decimal[], alpha: Decimal): Decimal {
    const b = computeB(q, alpha);
    const maxQi = q.reduce((a, b) => Decimal.max(a, b), new Decimal(0));
    const maxXi = maxQi.div(alpha);
    const sumExp = q.reduce((a, qi) => a.plus(Decimal.exp(qi.div(b).minus(maxXi))), new Decimal(0));
    const logSumExp = Decimal.ln(sumExp).plus(maxXi);
    return b.mul(logSumExp);
}

export function prices(q: Decimal[], alpha: Decimal, fee: Decimal): Decimal[] {

    const softmaxes = softmaxB(q, alpha);
    const sumSi = softmaxes.reduce((a, b) => a.plus(b.mul(b.ln())), new Decimal(0));

    const entropyTerm = sumSi.neg();
    const com = entropyTerm.mul(alpha)

    const oneMinusFee = new Decimal(1).minus(fee);
    return softmaxes.map(s => s.add(com).div(oneMinusFee));
}

export function pricesForMarket(market: MarketView): void {
    const q = market.outcomes.map(o => o.quantity)
    const alpha = market.alpha
    const fee = market.fee

    const p = prices(q, alpha, fee)
    market.outcomes.forEach((o, idx) => {
        o.price = p[idx];
    });
}

export function payoutForSpend(q: Decimal[], idxOutcome: number, alpha: Decimal, spend: Decimal): Decimal {

    const baseCost = cost(q, alpha);

    const p = prices(q, alpha, new Decimal(0));

    const lo = new Decimal(0);
    const hi = spend.div(p[idxOutcome]); // large upper bound

    const tol = new Decimal(10).pow(-3);
    const qNext = q.slice();

    const f = (deltaQoi: Decimal): Decimal => {
        qNext[idxOutcome] = q[idxOutcome].plus(deltaQoi);
        const nextCosts = cost(qNext, alpha);
        return nextCosts.minus(baseCost).minus(spend);
    };

    const payout = bisect(lo, hi, tol, f);
    return payout;
}

export function maxSpendToCap(q: Decimal[], idxOutcome: number, alpha: Decimal, capPrice: Decimal): Decimal {

    const p = prices(q, alpha, new Decimal(0));
    const currentPrice = p[idxOutcome];

    if (capPrice.lessThanOrEqualTo(currentPrice)) {
        return new Decimal(0);
    }

    const lo = new Decimal(MIN_SHARES);
    let hi = new Decimal(MIN_SHARES); // large upper bound
    const qNext = q.slice();

    while (true) {
        qNext[idxOutcome] = q[idxOutcome].plus(hi);
        const newPrices = prices(qNext, alpha, new Decimal(0));
        const newPrice = newPrices[idxOutcome];
        if (newPrice.greaterThanOrEqualTo(capPrice)) {
            break;
        }
        hi = hi.mul(2);
    }

    const tol = new Decimal(10).pow(-3);

    const f = (deltaQi: Decimal): Decimal => {
        qNext[idxOutcome] = q[idxOutcome].plus(deltaQi);
        const nextPrices = prices(qNext, alpha, new Decimal(0));
        const nextPrice = nextPrices[idxOutcome];
        return nextPrice.minus(capPrice);
    }

    const maxDeltaShares = bisect(lo, hi, tol, f);

    const baseCost = cost(q, alpha)

    const nextQ = q.slice();
    nextQ[idxOutcome] = nextQ[idxOutcome].plus(maxDeltaShares);
    const nextCost = cost(nextQ, alpha)

    return nextCost.minus(baseCost)
}


export function possiblePayoutPropForSpend(market: MarketView, outcomeId: number, spend: Decimal): [boolean, Decimal, Decimal] {

    const q = market.outcomes.map(o => o.quantity)
    const idxOutcome = market.outcomes.findIndex(o => o.id === outcomeId)
    if (idxOutcome == -1) {
        throw new Error("failed to find provided outcome id")
    }

    const alpha = market.alpha
    const fee = market.fee

    const feePaid = feeFromSpend(spend, fee)
    const availSpend = spend.minus(feePaid)

    const maxSpend = maxSpendToCap(q, idxOutcome, alpha, market.capPrice)
    if (spend.greaterThanOrEqualTo(maxSpend)) {
        return [false, new Decimal(0), new Decimal(0)]
    }

    const payout = payoutForSpend(q, idxOutcome, alpha, availSpend)
    const payoutRounded = truncatePrecision(payout, 2, Decimal.ROUND_DOWN)

    const proportion = gainProportion(spend, payout)
    const proportionRounded = truncatePrecision(proportion, SCALE, Decimal.ROUND_UP)

    return [true, payoutRounded, proportionRounded]



}

// export function possiblePayoutForCashout(market: MarketView, bet: Bet): [boolean, Decimal] {

// }
