import math
from matplotlib import patheffects
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.colors import BoundaryNorm, ListedColormap, TwoSlopeNorm
import matplotlib.patches as mpatches

ALPHA = 0.043
SEEDING = 1666
FEE = 0


def compute_b(q, alpha):
    return alpha * np.sum(q)


def softmax_b(q, b):
    max_q = np.max(q)
    max_x = max_q / b
    x = q / b
    exps = np.exp(x - max_x)
    sum_exps = np.sum(exps)
    return np.exp(x - max_x) / sum_exps


def ls_lmsr_cost_vec(q, alpha):

    b = compute_b(q, alpha)
    max_q = np.max(q)
    max_x = max_q / b
    x = q / b
    sum_exps = np.sum(np.exp(x - max_x))
    ln_sum_exps = math.log(sum_exps)
    return b * (max_x + ln_sum_exps)


def ls_lsmr_price(q, alpha):
    b = compute_b(q, alpha)
    s = softmax_b(q, b)
    sum_s = np.sum(s * np.log(s))
    h_s = -sum_s
    com = alpha * h_s
    return s + com


def profit_two_outcomes(fee, alpha, seeding):

    base_cost = ls_lmsr_cost_vec(np.array([seeding, seeding]), alpha)

    N = 500
    x = np.logspace(0, 8, N)
    y = np.logspace(0, 8, N)

    Z = np.zeros((N, N))
    Z_ABSOLUTE = np.zeros((N, N))
    E = np.zeros((N, N))

    for i in range(N):
        for j in range(N):
            q = np.array([x[i] + seeding, y[j] + seeding])

            cost = ls_lmsr_cost_vec(q, alpha)
            price = ls_lsmr_price(q, alpha)

            # closeness of either price to 0.9
            E[j, i] = np.min(np.abs(price - 0.9))  # scalar field

            # worst-case payout and platform profit
            max_q = np.max(np.array([x[i], y[j]]))
            total_paid = (cost - base_cost) / (1 - fee)
            profit = total_paid - max_q
            profit_percent = profit / total_paid
            Z[j, i] = profit_percent
            Z_ABSOLUTE[j, i] = profit

    tol = 0.01
    # Fig size
    fig, ax = plt.subplots(figsize=(10, 7))
    ax.set_title("Profit 0–10% (discrete), black: price≈0.94, blue: loss ≥ seeding/2")
    ax.set_xscale("log")
    ax.set_yscale("log")
    ax.set_xlabel("Shares of outcome 1")
    ax.set_ylabel("Shares of outcome 2")

    plt.contourf(x, y, Z, cmap="RdYlGn", levels=np.linspace(0, 0.08, 9))
    plt.colorbar(label="Profit %", ax=ax)

    plt.contour(x, y, E, levels=[tol], colors="black", linewidths=1.25)
    # The more blue, the more we lose
    plt.contourf(x, y, -Z_ABSOLUTE, cmap="Blues", levels=np.linspace(0, 50, 6))
    plt.colorbar(label="Loss $", ax=ax)

    plt.show()


profit_two_outcomes(FEE, ALPHA, SEEDING)


# print(
#     (
#         ls_lmsr_cost_vec(np.array([SEEDING + 40, SEEDING]), ALPHA)
#         - ls_lmsr_cost_vec(np.array([SEEDING, SEEDING]), ALPHA)
#     )
#     / (1 - FEE)
# )

# 2825
# 3435
