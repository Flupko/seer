import Decimal from "decimal.js";
import { computeB, SCALE } from "./lslmsr";


export function truncatePrecision(num: Decimal, precision: number, rounding: Decimal.Rounding): Decimal {
    return num.toDecimalPlaces(precision, rounding);
}

export function gainProportion(spend: Decimal, payout: Decimal): Decimal {
    return spend.div(payout);
}

export function softmaxB(q: Decimal[], alpha: Decimal): Decimal[] {

    const b = computeB(q, alpha);

    const maxQi = q.reduce((a, b) => Decimal.max(a, b), new Decimal(0));
    const maxXi = maxQi.div(alpha);

    const expValues = q.map(qi => Decimal.exp(qi.div(b).minus(maxXi)));
    const sumExp = expValues.reduce((a, b) => a.plus(b), new Decimal(0));

    return expValues.map(expVal => expVal.div(sumExp));
}

export function feeFromSpend(spend: Decimal, fee: Decimal): Decimal {

    const feePaid = spend.mul(fee)
    const feePaidRoundedUp = truncatePrecision(feePaid, SCALE, Decimal.ROUND_UP)
    return feePaidRoundedUp
}