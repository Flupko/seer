'use client';
import { OddsFormat } from '@/lib/odds';
import NumberFlow from '@number-flow/react';
import { Decimal } from 'decimal.js';
import { AnimatePresence, motion } from 'motion/react';

// function clamp01(x: number) {
//     return Number.isFinite(x) ? Math.min(1, Math.max(0, x)) : 0;
// }


// 1 / p, or 0 if p ≤ 0
function toDecimalOdds(p: Decimal): number {
    if (p.lessThanOrEqualTo(0)) return 0;
    return new Decimal(1).div(p).toDecimalPlaces(2, Decimal.ROUND_DOWN).toNumber();
}

function toPercent(p: Decimal): number {
    if (p.lessThanOrEqualTo(0)) return 0;
    return p.toDecimalPlaces(2, Decimal.ROUND_UP).toNumber();
}

// American odds: + for underdog (D ≥ 2), – for favorite
function toAmerican(p: Decimal) {
    if (p.lessThanOrEqualTo(0)) {
        return { sign: '+', value: 0 };
    }
    const d = new Decimal(1).div(p);
    if (d.greaterThanOrEqualTo(2)) {
        return { sign: '+', value: d.minus(1).mul(100).toDecimalPlaces(0, Decimal.ROUND_DOWN).toNumber() };
    }
    return { sign: '-', value: new Decimal(100).div(d.minus(1)).toDecimalPlaces(0, Decimal.ROUND_DOWN).toNumber() };
}

// Fractional odds F = (1/p – 1) as n/d reduced by GCD
function toFraction(p: Decimal, maxDen = 100) {
    if (p.lte(0)) {
        return { n: 0, d: 1 };
    }
    const f = new Decimal(1).div(p).minus(1);
    const num = f.mul(maxDen).round();
    const den = new Decimal(maxDen);

    // recursive GCD for Decimal
    const gcd = (a: Decimal, b: Decimal): Decimal =>
        b.equals(0) ? a : gcd(b, a.mod(b));

    const g = gcd(num.abs(), den).abs();
    return {
        n: num.div(g).toNumber(),
        d: den.div(g).toNumber(),
    };
}

export function AnimatedOdds({
    prob,    // Decimal
    format,  // 'decimal'|'american'|'percent'|'fractional'
}: {
    prob: Decimal;
    format: OddsFormat;
}) {
    const variants = {
        initial: { opacity: 0, y: 8 },
        animate: { opacity: 1, y: 0 },
        exit: { opacity: 0, y: -8 },
        transition: { duration: 0.18 },
    };


    return (

        <AnimatePresence mode="wait" initial={false}>
            {format === 'decimal' && (
                <motion.span key="decimal" {...variants}>
                    <NumberFlow
                        value={toDecimalOdds(prob)}
                        locales="en-US"
                        format={{ style: 'decimal', minimumFractionDigits: 2, maximumFractionDigits: 2, useGrouping: false }}
                    />
                </motion.span>
            )}

            {format === 'percent' && (
                <motion.span key="percent" {...variants}>
                    <NumberFlow
                        value={toPercent(prob)}
                        locales="en-US"
                        format={{ style: 'percent', minimumFractionDigits: 0, maximumFractionDigits: 0, useGrouping: false }}
                    />
                </motion.span>
            )}

            {format === 'american' && (() => {
                const { sign, value } = toAmerican(prob);
                return (
                    <motion.span key="american" {...variants} >
                        <span>
                            <span>{sign}</span>
                            <NumberFlow
                                value={value}
                                locales="en-US"
                                format={{ style: 'decimal', minimumFractionDigits: 0, maximumFractionDigits: 0, useGrouping: false }}
                            />
                        </span>
                    </motion.span>
                );
            })()}

            {format === 'fractional' && (() => {
                const { n, d } = toFraction(prob, 100);
                return (
                    <motion.span key="fractional" {...variants}>
                        <span>
                            <NumberFlow
                                value={n}
                                locales="en-US"
                                format={{ style: 'decimal', minimumFractionDigits: 0, maximumFractionDigits: 0, useGrouping: false }}
                            />
                            <span>/</span>
                            <NumberFlow
                                value={d}
                                locales="en-US"
                                format={{ style: 'decimal', minimumFractionDigits: 0, maximumFractionDigits: 0, useGrouping: false }}
                            />
                        </span>
                    </motion.span>
                );
            })()}
        </AnimatePresence>

    );
} 