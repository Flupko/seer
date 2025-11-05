import { Decimal } from 'decimal.js';
import { Bet, MarketView } from '../definitions';
import { feeFromSpend, gainProportion, softmaxB, truncatePrecision } from './math_util';


export const SCALE = 12;
export const MIN_SHARES = new Decimal(0.1);

Decimal.set({ precision: 12, rounding: Decimal.ROUND_DOWN });

export function bisect(lo: number, hi: number, tol: number, f: (x: number) => number): number {

    let nbIterations = 0;

    while (hi - lo > tol) {
        const mid = (lo + hi) / 2;
        nbIterations += 1;
        if (f(mid) > 0) {
            hi = mid;
        } else {
            lo = mid;
        }
    }

    console.log("bisect iterations:", nbIterations);

    return lo
}

export function computeB(q: number[], alpha: number): number {
    const sum = q.reduce((a, b) => a + b, 0);
    return alpha * sum;
}

export function cost(q: number[], alpha: number): number {
    const b = computeB(q, alpha);

    // LogSumExp trick to improve numerical stability
    // const maxQi = q.reduce((a, b) => Math.max(a, b), 0);
    // const maxXi = maxQi / b;
    // const sumExp = q.reduce((a, qi) => a + Math.exp((qi / b) - maxXi), 0);
    // const logSumExp = Math.log(sumExp) + maxXi;

    // Without the trick
    const sumExp = q.reduce((a, qi) => a + Math.exp(qi / b), 0);
    const logSumExp = Math.log(sumExp);
    return b * logSumExp;
}

export function prices(q: number[], alpha: number, fee: number): number[] {
    const softmaxes = softmaxB(q, alpha);
    const sumSi = softmaxes.reduce((a, b) => a + (b * Math.log(b)), 0);
    const entropyTerm = -sumSi;
    const com = entropyTerm * alpha;
    const oneMinusFee = 1 - fee;

    return softmaxes.map(s => (s + com) / oneMinusFee);
}

export function pricesForMarket(market: MarketView): void {
    const q = market.outcomes.map(o => o.quantity)
    const alpha = market.alpha
    const fee = market.fee

    const p = prices(q.map(q => q.toNumber()), alpha.toNumber(), fee.toNumber())
    market.outcomes.forEach((o, idx) => {
        o.price = new Decimal(p[idx]);
    });
}

export function payoutForSpend(q: number[], idxOutcome: number, alpha: number, spend: number): number {

    const baseCost = cost(q, alpha);

    const p = prices(q, alpha, 0);

    const lo = spend;
    const hi = spend / p[idxOutcome]; // large upper bound

    const tol = 10 ** -3;
    const qNext = q.slice();

    const f = (deltaQi: number): number => {
        qNext[idxOutcome] = q[idxOutcome] + deltaQi;
        const nextCosts = cost(qNext, alpha);
        return nextCosts - baseCost - spend;
    };

    const payout = bisect(lo, hi, tol, f);
    return payout;
}


export function possiblePayoutProbForSpend(market: MarketView, outcomeId: number, spend: Decimal): [boolean, Decimal, Decimal] {

    const q = market.outcomes.map(o => o.quantity)
    const idxOutcome = market.outcomes.findIndex(o => o.id === outcomeId)
    if (idxOutcome == -1) {
        throw new Error("failed to find provided outcome id")
    }

    const alpha = market.alpha
    const fee = market.fee

    const feePaid = feeFromSpend(spend, fee)
    const availSpend = spend.minus(feePaid)

    const qNum = q.map(qi => qi.toNumber())
    const alphaNum = alpha.toNumber()
    const capPriceNum = market.capPrice.toNumber()
    const availSpendNum = availSpend.toNumber()

    const payoutNum = payoutForSpend(qNum, idxOutcome, alphaNum, availSpendNum)

    // Check if price after buying payout shares is less than can
    const qAfterNum = qNum.slice()
    qAfterNum[idxOutcome] += payoutNum
    if (prices(qAfterNum, alphaNum, 0)[idxOutcome] > capPriceNum) return [false, new Decimal(0), new Decimal(0)]

    const payout = new Decimal(payoutNum)
    const payoutRounded = truncatePrecision(payout, 2, Decimal.ROUND_DOWN)

    const proportion = gainProportion(spend, payoutRounded)
    const proportionRounded = truncatePrecision(proportion, SCALE, Decimal.ROUND_UP)

    return [true, payoutRounded, proportionRounded]



}

export function possiblePayoutDeltaForCashout(market: MarketView, bet: Bet): [boolean, Decimal, Decimal] {

    const q = market.outcomes.map(o => o.quantity)
    const idxBoughtOutcome = market.outcomes.findIndex(o => o.id === bet.outcomeId)

    const alpha = market.alpha
    const qNum = q.map(qi => qi.toNumber())
    const alphaNum = alpha.toNumber()
    const capPriceNum = market.capPrice.toNumber()
    const spend = bet.pricePaid

    const sharesBoughtNum = bet.payout.toNumber()

    const qHedgeNum = qNum.slice()
    for (let i = 0; i < qHedgeNum.length; i++) {
        if (i === idxBoughtOutcome) {
            continue;
        }
        qHedgeNum[i] += sharesBoughtNum;
    }

    // Check if price after buying hedge shares is less than cap
    const pricesAfterHedge = prices(qHedgeNum, alphaNum, 0);

    for (let i = 0; i < pricesAfterHedge.length; i++) {
        if (i === idxBoughtOutcome) {
            continue;
        }
        if (pricesAfterHedge[i] > capPriceNum) {
            return [false, new Decimal(0), new Decimal(0)];
        }
    }

    const baseCost = cost(qNum, alphaNum)
    const hedgeCost = cost(qHedgeNum, alphaNum)

    const deltaHedgeCost = hedgeCost - baseCost
    const gain = new Decimal(sharesBoughtNum - deltaHedgeCost)
    const gainRounded = truncatePrecision(gain, 2, Decimal.ROUND_DOWN)

    const deltaProp = gainRounded.div(spend).minus(1);
    const deltaPropRounded = truncatePrecision(deltaProp, SCALE, Decimal.ROUND_UP)

    return [true, deltaPropRounded, gainRounded]

}


// export function possiblePayoutForCashout(market: MarketView, bet: Bet): [boolean, Decimal] {

// }




// import { Decimal } from 'decimal.js';
// import { MarketView } from '../definitions';
// import { feeFromSpend, gainProportion, softmaxB, truncatePrecision } from './math_util';


// export const SCALE = 12;
// export const MIN_SHARES = new Decimal(0.1);

// Decimal.set({ precision: 12, rounding: Decimal.ROUND_DOWN });

// export function bisect(lo: Decimal, hi: Decimal, tol: Decimal, f: (x: Decimal) => Decimal): Decimal {

//     while (hi.minus(lo).greaterThan(tol)) {
//         const mid = lo.plus(hi).dividedBy(2);
//         if (f(mid).greaterThan(0)) {
//             hi = mid;
//         } else {
//             lo = mid;
//         }
//     }

//     return lo
// }

// export function computeB(q: Decimal[], alpha: Decimal): Decimal {
//     const sum = q.reduce((a, b) => a.plus(b), new Decimal(0));
//     return Decimal.mul(alpha, sum);
// }

// export function cost(q: Decimal[], alpha: Decimal): Decimal {
//     const b = computeB(q, alpha);
//     const maxQi = q.reduce((a, b) => Decimal.max(a, b), new Decimal(0));
//     const maxXi = maxQi.div(alpha);
//     const sumExp = q.reduce((a, qi) => a.plus(Decimal.exp(qi.div(b).minus(maxXi))), new Decimal(0));
//     const logSumExp = Decimal.ln(sumExp).plus(maxXi);
//     return b.mul(logSumExp);
// }

// export function prices(q: Decimal[], alpha: Decimal, fee: Decimal): Decimal[] {

//     const softmaxes = softmaxB(q, alpha);
//     const sumSi = softmaxes.reduce((a, b) => a.plus(b.mul(b.ln())), new Decimal(0));

//     const entropyTerm = sumSi.neg();
//     const com = entropyTerm.mul(alpha)

//     const oneMinusFee = new Decimal(1).minus(fee);
//     return softmaxes.map(s => s.add(com).div(oneMinusFee));
// }

// export function pricesForMarket(market: MarketView): void {
//     const q = market.outcomes.map(o => o.quantity)
//     const alpha = market.alpha
//     const fee = market.fee

//     const p = prices(q, alpha, fee)
//     market.outcomes.forEach((o, idx) => {
//         o.price = p[idx];
//     });
// }

// export function payoutForSpend(q: Decimal[], idxOutcome: number, alpha: Decimal, spend: Decimal): Decimal {

//     const baseCost = cost(q, alpha);

//     const p = prices(q, alpha, new Decimal(0));

//     const lo = new Decimal(0);
//     const hi = spend.div(p[idxOutcome]); // large upper bound

//     const tol = new Decimal(10).pow(-3);
//     const qNext = q.slice();

//     const f = (deltaQoi: Decimal): Decimal => {
//         qNext[idxOutcome] = q[idxOutcome].plus(deltaQoi);
//         const nextCosts = cost(qNext, alpha);
//         return nextCosts.minus(baseCost).minus(spend);
//     };

//     const payout = bisect(lo, hi, tol, f);
//     return payout;
// }

// export function maxSpendToCap(q: Decimal[], idxOutcome: number, alpha: Decimal, capPrice: Decimal): Decimal {

//     const p = prices(q, alpha, new Decimal(0));
//     const currentPrice = p[idxOutcome];

//     if (capPrice.lessThanOrEqualTo(currentPrice)) {
//         return new Decimal(0);
//     }

//     const lo = new Decimal(MIN_SHARES);
//     let hi = new Decimal(MIN_SHARES); // large upper bound
//     const qNext = q.slice();

//     while (true) {
//         qNext[idxOutcome] = q[idxOutcome].plus(hi);
//         const newPrices = prices(qNext, alpha, new Decimal(0));
//         const newPrice = newPrices[idxOutcome];
//         if (newPrice.greaterThanOrEqualTo(capPrice)) {
//             break;
//         }
//         hi = hi.mul(2);
//     }

//     const tol = new Decimal(10).pow(-3);

//     const f = (deltaQi: Decimal): Decimal => {
//         qNext[idxOutcome] = q[idxOutcome].plus(deltaQi);
//         const nextPrices = prices(qNext, alpha, new Decimal(0));
//         const nextPrice = nextPrices[idxOutcome];
//         return nextPrice.minus(capPrice);
//     }

//     const maxDeltaShares = bisect(lo, hi, tol, f);

//     const baseCost = cost(q, alpha)

//     const nextQ = q.slice();
//     nextQ[idxOutcome] = nextQ[idxOutcome].plus(maxDeltaShares);
//     const nextCost = cost(nextQ, alpha)

//     return nextCost.minus(baseCost)
// }


// export function possiblePayoutPropForSpend(market: MarketView, outcomeId: number, spend: Decimal): [boolean, Decimal, Decimal] {

//     const q = market.outcomes.map(o => o.quantity)
//     const idxOutcome = market.outcomes.findIndex(o => o.id === outcomeId)
//     if (idxOutcome == -1) {
//         throw new Error("failed to find provided outcome id")
//     }

//     const alpha = market.alpha
//     const fee = market.fee

//     const feePaid = feeFromSpend(spend, fee)
//     const availSpend = spend.minus(feePaid)

//     const maxSpend = maxSpendToCap(q, idxOutcome, alpha, market.capPrice)
//     if (spend.greaterThanOrEqualTo(maxSpend)) {
//         return [false, new Decimal(0), new Decimal(0)]
//     }

//     const payout = payoutForSpend(q, idxOutcome, alpha, availSpend)
//     const payoutRounded = truncatePrecision(payout, 2, Decimal.ROUND_DOWN)

//     const proportion = gainProportion(spend, payout)
//     const proportionRounded = truncatePrecision(proportion, SCALE, Decimal.ROUND_UP)

//     return [true, payoutRounded, proportionRounded]



// }


