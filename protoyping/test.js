// ------------------------
// Golden-section line search
// ------------------------

// Factory returning a pure golden-section 1-D minimizer:
// goldenSectionSearch(obj, left, right, tolerance, maxIter)
var goldenSectionFactoryCache, goldenSectionFactoryReady;
function makeGoldenSectionSearch() {
    if (goldenSectionFactoryReady) return goldenSectionFactoryCache;
    goldenSectionFactoryReady = 1;

    // Golden ratio shrink factor
    var phi = 2 / (1 + Math.sqrt(5));

    goldenSectionFactoryCache = goldenSectionSearch;

    function goldenSectionSearch(objective, left, right, tolerance, maxIter) {
        // Internal state mirrors original variable flow
        var midPointAtReturn, objAtReturnMid, iter = 0;

        // Interior test points
        var x1 = right - phi * (right - left);
        var x2 = left + phi * (right - left);

        // Objective evaluations at bracket ends and interior points
        var f1 = objective(x1);
        var f2 = objective(x2);
        var fL = objective(left);
        var fR = objective(right);

        // Keep the original "remembered ends" semantics
        var rememberedLeft = left;
        var rememberedRight = right;

        // Main shrink loop
        while (++iter < maxIter && Math.abs(right - left) > tolerance) {
            if (f2 > f1) {
                // Shrink [left, right] -> [left, x2]
                right = x2;
                x2 = x1;
                f2 = f1;
                x1 = right - phi * (right - left);
                f1 = objective(x1);
            } else {
                // Shrink [left, right] -> [x1, right]
                left = x1;
                x1 = x2;
                f1 = f2;
                x2 = left + phi * (right - left);
                f2 = objective(x2);
            }
        }

        // Original exit rules:
        // - If maxed iterations, return arithmetic average of f1 and f2
        if (iter === maxIter) return 0.5 * (f1 + f2);

        // - If any NaN, return NaN
        if (isNaN(f2) || isNaN(f1)) return NaN;

        // - Else choose midpoint index based on comparing avg(f1,f2) to fL and fR
        midPointAtReturn = 0.5 * (right + left);
        objAtReturnMid = 0.5 * (f1 + f2);

        return fL < objAtReturnMid
            ? rememberedLeft
            : (fR < objAtReturnMid ? rememberedRight : midPointAtReturn);
    }

    return goldenSectionFactoryCache;
}


// ------------------------
// 1-D bracket expansion (unimodal bracketing)
// ------------------------

// Factory returning a bracketer that expands from a guess:
// bracketMinimum(obj, guess, step0, lowerBound, upperBound, maxIter)
var bracketFactoryCache, bracketFactoryReady;
function makeBracketMinimum() {
    if (bracketFactoryReady) return bracketFactoryCache;
    bracketFactoryReady = 1;

    bracketFactoryCache = bracketMinimum;

    function bracketMinimum(objective, guess, initialStep, lowerBound, upperBound, maxIter) {
        var fLeft, fRight, fBest, step, left, right, done;

        // Initialize expansion counters and points
        step = 1;
        left = guess;
        right = guess;

        // Evaluate at guess
        fBest = fRight = fLeft = objective(guess);

        // Expand alternatingly while step is finite
        while (!done && isFinite(initialStep) && !isNaN(initialStep)) {
            ++step;
            done = true;

            // Try expanding to the left
            if (fLeft <= fBest) {
                fBest = fLeft;
                left = Math.max(lowerBound, left - initialStep);
                fLeft = objective(left);
                done = false;
            }

            // Try expanding to the right
            if (fRight <= fBest) {
                fBest = fRight;
                right = Math.min(upperBound, right + initialStep);
                fRight = objective(right);
                done = false;
            }

            // Keep the best observed value across the three
            fBest = Math.min(fBest, fLeft, fRight);

            // Stop expanding if pinned to either boundary with no improvement
            if ((fLeft === fBest && left === lowerBound) || (fRight === fBest && right === upperBound)) {
                done = true;
            }

            // Grow expansion scale (same schedule as original)
            initialStep *= step < 4 ? 2 : Math.exp(step * 0.5);

            if (!isFinite(initialStep)) {
                return [-Infinity, Infinity];
            }
            if (step >= maxIter) break;
        }

        // Return the bounding interval
        return [left, right];
    }

    return bracketFactoryCache;
}


// ------------------------
// 1-D line search driver (bracket + golden section)
// ------------------------

// Factory that returns a convenience wrapper combining bracketing then golden section:
// lineSearch1D(objective, {tolerance, initialIncrement, lowerBound, upperBound, maxIter, guess})
var lineSearch1DFactoryCache, lineSearch1DFactoryReady;
function makeLineSearch1D() {
    if (lineSearch1DFactoryReady) return lineSearch1DFactoryCache;
    lineSearch1DFactoryReady = 1;

    var goldenSectionSearch = makeGoldenSectionSearch();
    var bracketMinimum = makeBracketMinimum();

    lineSearch1DFactoryCache = function lineSearch1D(objective, opts) {
        opts = opts || {};

        var guess, bracket, tolerance = opts.tolerance === void 0 ? 1e-8 : opts.tolerance;
        var initialIncrement = opts.initialIncrement === void 0 ? 1 : opts.initialIncrement;
        var lowerBound = opts.lowerBound === void 0 ? -Infinity : opts.lowerBound;
        var upperBound = opts.upperBound === void 0 ? Infinity : opts.upperBound;
        var maxIter = opts.maxIter === void 0 ? 100 : opts.maxIter;

        // If explicit bounds given, trust them; else bracket around a guess
        if (isFinite(upperBound) && isFinite(lowerBound)) {
            bracket = [lowerBound, upperBound];
        } else {
            // Default guess logic mirrors original
            if (opts.guess === void 0) {
                if (lowerBound > -Infinity) {
                    guess = (upperBound < Infinity) ? 0.5 * (lowerBound + upperBound) : lowerBound;
                } else {
                    guess = (upperBound < Infinity) ? upperBound : 0;
                }
            } else {
                guess = opts.guess;
            }

            bracket = bracketMinimum(objective, guess, initialIncrement, lowerBound, upperBound, maxIter);
            if (isNaN(bracket[0]) || isNaN(bracket[1])) return NaN;
        }

        // Golden-section on the bracket
        return goldenSectionSearch(objective, bracket[0], bracket[1], tolerance, maxIter);
    };

    return lineSearch1DFactoryCache;
}


// ------------------------
// Powell direction-set (multi-D optimizer with 1-D line search)
// ------------------------

// Factory returning a Powell optimizer that uses the lineSearch1D above:
// powellMinimize(f, x0, options, trace?)
var powellFactoryCache, powellFactoryReady;
function makePowellMinimizer() {
    if (powellFactoryReady) return powellFactoryCache;
    powellFactoryReady = 1;

    var lineSearch1D = makeLineSearch1D();

    powellFactoryCache = powellMinimize;

    function powellMinimize(f, x0, options, trace) {
        // Renamed variables for clarity, semantics preserved
        var i, j, direction, step, stepSize, lineObj, newPoint, dirNormSq, progressVec, progressNorm, totalStep, dx, boundsSpan;

        options = options || {};
        var maxIter = options.maxIter === void 0 ? 20 : options.maxIter;
        var tol = options.tolerance === void 0 ? 1e-8 : options.tolerance;
        var lineTol = options.lineTolerance === void 0 ? tol : options.lineTolerance;
        var bounds = options.bounds === void 0 ? [] : options.bounds;
        var verbose = options.verbose === void 0 ? false : options.verbose;

        if (trace) trace.points = [];

        var dim = x0.length;
        var x = x0.slice(0);

        // Initialize direction set to identity
        var directions = [];
        var lastX = [];

        for (i = 0; i < dim; i++) {
            directions[i] = [];
            for (j = 0; j < dim; j++) directions[i][j] = (i === j ? 1 : 0);
        }

        // Project x into bounds (if any)
        function projectIntoBounds(vec) {
            for (var k = 0; k < bounds.length; k++) {
                var b = bounds[k];
                if (b) {
                    if (isFinite(b[0])) vec[k] = Math.max(b[0], vec[k]);
                    if (isFinite(b[1])) vec[k] = Math.min(b[1], vec[k]);
                }
            }
        }

        projectIntoBounds(x);
        if (trace) trace.points.push(x.slice());

        // Compute feasible line parameter range [tMin, tMax] for a given direction
        var lineBounds = options.bounds
            ? function (xCur, dir) {
                var tMax = Infinity, tMin = -Infinity;
                for (var d = 0; d < dim; d++) {
                    var b = bounds[d];
                    if (b && dir[d] !== 0) {
                        if (b[0] !== void 0 && isFinite(b[0])) {
                            tMin = (dir[d] > 0 ? Math.max : Math.min)(tMin, (b[0] - xCur[d]) / dir[d]);
                        }
                        if (b[1] !== void 0 && isFinite(b[1])) {
                            tMax = (dir[d] > 0 ? Math.min : Math.max)(tMax, (b[1] - xCur[d]) / dir[d]);
                        }
                    }
                }
                return [tMin, tMax];
            }
            : function () { return [-Infinity, Infinity]; };

        // Reusable work arrays and closures (match original structure)
        var trial = [];
        var lineFn = function (t) {
            for (var k = 0; k < dim; k++) trial[k] = x[k] + direction[k] * t;
            return f(trial);
        };

        var iter = 0;
        var lastStepNorm = 0;

        while (++iter < maxIter) {
            // Periodically reset directions to identity
            if (iter % dim === 0) {
                for (i = 0; i < dim; i++) {
                    for (j = 0; j < dim; j++) directions[i][j] = (i === j ? 1 : 0);
                }
            }

            // Snapshot point before sweeping directions
            for (j = 0, lastX = []; j < dim; j++) lastX[j] = x[j];

            // Sweep all directions with line searches
            for (i = 0; i < dim; i++) {
                direction = directions[i];

                boundsSpan = lineBounds(x, direction);
                stepSize = 0.1; // same seed as original

                step = lineSearch1D(lineFn, {
                    lowerBound: boundsSpan[0],
                    upperBound: boundsSpan[1],
                    initialIncrement: stepSize,
                    tolerance: stepSize * lineTol
                });

                if (step === 0) return x; // converged along this direction

                // Move along the chosen direction
                for (j = 0; j < dim; j++) x[j] += step * direction[j];

                projectIntoBounds(x);
                if (trace) trace.points.push(x.slice());
            }

            // Build "progress direction" = current - last
            directions.shift();
            progressVec = [];
            var progressLenSq = 0;
            for (j = 0; j < dim; j++) {
                progressVec[j] = x[j] - lastX[j];
                progressLenSq += progressVec[j] * progressVec[j];
            }

            progressNorm = Math.sqrt(progressLenSq);
            if (progressNorm > 0) {
                for (j = 0; j < dim; j++) progressVec[j] /= progressNorm;
            } else {
                return x; // no progress
            }

            // Append progress direction and take one more line search step on it
            directions.push(progressVec);
            direction = progressVec;

            boundsSpan = lineBounds(x, direction);
            stepSize = 0.1;

            step = lineSearch1D(lineFn, {
                lowerBound: boundsSpan[0],
                upperBound: boundsSpan[1],
                initialIncrement: stepSize,
                tolerance: stepSize * lineTol
            });

            if (step === 0) return x;

            // Apply this final step
            totalStep = 0;
            for (j = 0; j < dim; j++) {
                dx = step * direction[j];
                totalStep += dx * dx;
                x[j] += dx;
            }

            projectIntoBounds(x);
            if (trace) trace.points.push(x.slice());

            totalStep = Math.sqrt(totalStep);

            if (verbose) {
                console.log(
                    "Iteration " + iter + ": " + (totalStep / lastStepNorm) + " f(" + x + ") = " + f(x)
                );
            }

            // Global stopping criterion
            if (totalStep / lastStepNorm < tol) return x;
            lastStepNorm = totalStep;
        }

        return x; // Return best-known point
    }

    return powellFactoryCache;
}

// Build optimizer and whatever wrapper the environment provides
var powell = makePowellMinimizer();


// ------------------------
// Numerics helpers
// ------------------------

// Stable log-sum-exp in linear-time with max-shift
function logSumExp(...xs) {
    const m = Math.max(...xs);
    return m + Math.log(xs.map(x => Math.exp(x - m)).reduce((acc, v) => acc + v, 0));
}


// ------------------------
// LMSR (constant b) helpers
// ------------------------

// Trade cost in LMSR: amount to add `deltaShares` on `targetOutcome`
function lmsrTradeCost(market, targetOutcome, deltaShares, lmsrIndex) {
    const b = market.scoring_rule_metadata.lmsr[lmsrIndex].liquidity_param;

    const outcomes = market.outcomes.filter(o => !o.disabled);
    if (outcomes.length === 0 || typeof outcomes[0] !== 'object') {
        throw "Error while calculating lmsr price: Not valid outcomes for question";
    }

    // Current scaled q/b and hypothetical (q+Δ)/b
    const scaledNow = [];
    const scaledNext = [];

    for (const o of outcomes) {
        const qOverB = o.shares[lmsrIndex] / b;
        scaledNow.push(qOverB);
        scaledNext.push(o.id === targetOutcome.id ? (o.shares[lmsrIndex] + deltaShares) / b : qOverB);
    }

    const costAfter = b * logSumExp(...scaledNext);
    const costBefore = b * logSumExp(...scaledNow);
    return costAfter - costBefore;
}

// Closed-form inverse for LMSR: shares purchasable for `spend`
function lmsrSharesForSpend(market, targetOutcome, spend, indexLike) {
    const idx = CD(indexLike); // External mapping helper as in original
    const b = market.scoring_rule_metadata.lmsr[idx].liquidity_param;

    const outcomes = market.outcomes.filter(o => !o.active);
    if (outcomes.length === 0 || typeof outcomes[0] !== 'object') {
        throw "Error while calculating lmsr shares: Not valid outcomes for question";
    }

    // S = Σ exp(q_j/b), T = Σ_{j≠i} exp(q_j/b), F = exp(spend/b)
    const sumAll = outcomes.reduce((acc, o) => acc + Math.exp(o.shares[idx] / b), 0);
    const sumOthers = outcomes
        .filter(o => o.id !== targetOutcome.id)
        .reduce((acc, o) => acc + Math.exp(o.shares[idx] / b), 0);
    const spendFactor = Math.exp(spend / b);

    // x = b * log(F*S - T) - q_i
    return b * Math.log(spendFactor * sumAll - sumOthers) - targetOutcome.shares[idx];
}


// ------------------------
// LS-LMSR (variable b) helpers
// ------------------------

// α = funding / (n * log n)
const alphaFromFunding = (funding, quantities) => {
    const n = quantities.length;
    return funding / (n * Math.log(n));
};

// b(q) = α * Σ q_j
const liquidityScale = (funding, quantities) => {
    const alpha = alphaFromFunding(funding, quantities);
    const total = quantities.reduce((acc, q) => acc + q, 0);
    return alpha * total;
};

// Pre-evaluate a 1-D cost delta along a path of length `amount` at index `k`
// side 'l': add to index k; side 's': add to all indices except k
const lsLmsrCostDeltaForMove = (quantities, k, amount, funding, side) =>
    (side === "l"
        ? (u => {
            const next = [...quantities];
            next[k] += u;
            return lsLmsrCost(funding, next) - lsLmsrCost(funding, quantities);
        })
        : (u => {
            let next = [...quantities];
            next = next.map((q, j) => (j === k ? q : q + u));
            return lsLmsrCost(funding, next) - lsLmsrCost(funding, quantities);
        }))(amount);

// LS-LMSR cost: C(q) = b(q) * log( Σ exp(q_j / b(q)) )
const lsLmsrCost = (funding, quantities) => {
    const bq = liquidityScale(funding, quantities);
    const sumExp = quantities.reduce((acc, q) => acc + Math.exp(q / bq), 0);
    return bq * Math.log(sumExp);
};

// Instantaneous price-like gradient component at index `i`
function lsLmsrInstantaneousPrice(quantities, i = 1, funding) {
    const bq = liquidityScale(funding, quantities);

    const expQi = Math.exp(quantities[i] / bq);
    const expAll = quantities.map(q => Math.exp(q / bq));
    const sumExp = expAll.reduce((acc, x) => acc + x, 0);

    const sumQ = quantities.reduce((acc, x) => acc + x, 0);

    // α * log(Σ exp) + chain-rule correction
    const alphaLog = alphaFromFunding(funding, quantities) * Math.log(sumExp);
    const correctionNumer = expQi * sumQ - quantities.reduce((acc, q, j) => acc + q * expAll[j], 0);
    const correction = correctionNumer / (sumQ * sumExp);

    return alphaLog + correction;
}

// Vector of "prices"; optionally normalized to sum to 1 (display convenience)
function lsLmsrPriceVector(quantities, normalize = true, funding) {
    let prices = quantities.map((_, i) => lsLmsrInstantaneousPrice(quantities, i, funding));
    if (normalize) {
        const total = prices.reduce((acc, x) => acc + x, 0);
        prices = prices.map(x => x / total);
    }
    return prices;
}

// Compute trade size u for target spend under LS-LMSR by 1-D search
function lsLmsrSharesForSpend(quantities, targetSpend, outcomeIndex, funding, side = "l") {

    const n = quantities.length;
    const spend = parseFloat(targetSpend);

    let seed, objective;

    if (side === "l") {
        seed = spend / lsLmsrPriceVector(quantities, true, funding)[outcomeIndex];

        objective = t => {
            const next = [...quantities];
            next[outcomeIndex] += t[0];
            const delta = lsLmsrCost(funding, next) - lsLmsrCost(funding, quantities);
            return Math.abs(delta - spend) * n;
        };

    } else if (side === "s") {
        seed = spend / (1 - lsLmsrPriceVector(quantities, true, funding)[outcomeIndex]);

        objective = t => {
            const next = [...quantities];
            for (let j = 0; j < next.length; j++) if (j !== outcomeIndex) next[j] += t[0];
            const delta = lsLmsrCost(funding, next) - lsLmsrCost(funding, quantities);
            return Math.abs(delta - spend) * n;
        };
    } else {
        throw new Error("Invalid position. Must be 'l' (long) or 's' (short).");
    }

    // External root/line solver wrapper; seed as [u0]
    return rootOrLineSolver(objective, [seed]);
}

// Maker PnL (negative of trader surplus) under path policy
const makerPnl = (
    quantities,
    outcomeIndex,
    cash,
    funding,
    side = "l",
    mode = "ALWAYS_MOVING_FORWARD"
) => {
    let next, pnl;

    if (mode === "ALWAYS_MOVING_FORWARD") {
        next = quantities.map((q, j) =>
            (side === "l" && j !== outcomeIndex) || (side === "s" && j === outcomeIndex) ? q + cash : q
        );
        pnl = -1 * (lsLmsrCost(funding, next) - lsLmsrCost(funding, quantities) - cash);
    } else {
        next = quantities.map((q, j) =>
            (side === "s" && j !== outcomeIndex) || (side === "l" && j === outcomeIndex) ? q - cash : q
        );
        pnl = -1 * (lsLmsrCost(funding, next) - lsLmsrCost(funding, quantities));
    }

    if (!["l", "s"].includes(side)) {
        throw new Error("Invalid position. Must be 'l' (long) or 's' (short).");
    }
    return pnl;
};

function rootOrLineSolver(objectiveVecArg, seedVec) {
    const lineSearch1D = makeLineSearch1D();
    const guess = (seedVec && seedVec.length) ? seedVec[0] : 0;
    const scalarObjective = (u) => objectiveVecArg([u]);
    const u = lineSearch1D(scalarObjective, {
        guess,
        initialIncrement: Math.max(1e-6, Math.abs(guess) || 1),
        tolerance: 1e-8,
        maxIter: 200
    });
    return u;
}
// Example inputs
const q = [1000, 1000];                 // current shares per outcome
const n = q.length;                   // number of outcomes                 // desired α
const spend = 20;                    // budget (e.g., $1)
const outcomeIndex = 1;               // buy outcome 0

const funding = 0.06
console.log(alphaFromFunding(funding, q));

// Compute shares purchasable with 'spend' on outcomeIndex under LS-LMSR
const shares = lsLmsrSharesForSpend(q, spend, outcomeIndex, funding, 'l');