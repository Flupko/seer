import { Decimal } from "decimal.js";
import { gcd } from "./lslmsr/math_util";

export type OddsFormat = 'decimal' | 'american' | 'fractional' | 'percent'

export function formatOdds(prob: Decimal, format: OddsFormat): string {

    switch (format) {
        case 'decimal':
            return (new Decimal(1).div(prob)).toDecimalPlaces(2, Decimal.ROUND_DOWN).toFixed(2);
        case 'american':
            if (prob.greaterThanOrEqualTo(0.5)) {
                return `-${prob.div(new Decimal(1).minus(prob)).mul(100).toFixed(0)}`;
            }
            return `+${new Decimal(1).minus(prob).div(prob).mul(100).toFixed(0)}`;

        case 'fractional':

            const maxDen = 100
            const f = new Decimal(1).div(prob).minus(1);

            const num = f.mul(maxDen).round();
            const den = new Decimal(maxDen);

            const g = gcd(num.abs(), den).abs();

            return `${num.div(g).toNumber()}/${den.div(g).toNumber()}`

        case 'percent':
            // Return percentage with no decimal places
            return `${prob.mul(100).toDecimalPlaces(0, Decimal.ROUND_UP).toFixed(0)}%`;
    }
}

export function splitNumberLikeParts(formattedOdd: string): { part: string, isNumber: boolean }[] {

    const parts: { part: string, isNumber: boolean }[] = [];
    let currentPart = "";
    let isCurrentPartNumberLike = false;
    for (let i = 0; i < formattedOdd.length; i++) {
        const char = formattedOdd[i];
        const isCharDigitLike = (char >= '0' && char <= '9') || char === '.' || char === ',' || char === '-' || char === '+';
        if (currentPart === "") {
            currentPart += char;
            isCurrentPartNumberLike = isCharDigitLike;
            continue;
        }

        if (isCharDigitLike === isCurrentPartNumberLike) {
            currentPart += char;
            continue;
        }

        parts.push({ part: currentPart, isNumber: isCurrentPartNumberLike });
        currentPart = char;
        isCurrentPartNumberLike = isCharDigitLike;

    }

    if (currentPart !== "") {
        parts.push({ part: currentPart, isNumber: isCurrentPartNumberLike });
    }

    return parts;
}