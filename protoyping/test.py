# I'll re-run the profit-region plot but *fix* the payout error:
# - The earlier script subtracted only the newly bought shares (X or Y) when computing the worst-case payout.
# - The correct worst-case payout is the total outstanding shares for the winning outcome:
#     payout = seed + purchased_shares
#   i.e., the final q_i (not just the increment).
#
# I will:
# 1) compute the LS-LMSR cost vectorized (same formula),
# 2) compute delta_C = C(q) - C(q0),
# 3) compute gross_budgets = delta_C / (1 - FEE),
# 4) compute payout_worst = max(q_final) (this *includes* the seed),
# 5) compute profit_worst = gross_budgets - payout_worst,
# 6) mark red where profit_worst < 0, green where profit_worst/gross_budgets >= 0.04.
#
# I will display the corrected plot.
import math
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.colors import ListedColormap
import matplotlib.patches as mpatches

ALPHA = 0.05
SEEDING = 100.0   # keep your original seed to show the effect
FEE = 0.06

def ls_lmsr_cost_vec(q0, q1, alpha=ALPHA):
    q0 = np.asarray(q0, dtype=np.float64)
    q1 = np.asarray(q1, dtype=np.float64)
    # per-cell max and sum (elementwise)
    max_q = np.maximum(q0, q1)
    s = q0 + q1
    b = alpha * s
    tiny = np.finfo(float).tiny
    # avoid division by zero: where b tiny, cost ~ max_q
    with np.errstate(divide='ignore', over='ignore', invalid='ignore'):
        ex0 = np.exp((q0 - max_q) / np.where(b <= tiny, 1.0, b))
        ex1 = np.exp((q1 - max_q) / np.where(b <= tiny, 1.0, b))
        logsum = np.log(ex0 + ex1)
    # when b tiny, we want cost = max_q (as in scalar implementation)
    cost = max_q + b * logsum
    # fix cells where b is tiny
    cost = np.where(b <= tiny, max_q, cost)
    return cost

# base cost via same function for numerical consistency
base_q0 = SEEDING
base_q1 = SEEDING
base_cost = ls_lmsr_cost_vec(base_q0, base_q1, ALPHA)  # scalar

# grid
N = 250
x_vals = np.unique(np.round(np.logspace(0, 6, N)).astype(int))
y_vals = np.unique(np.round(np.logspace(0, 6, N)).astype(int))
X, Y = np.meshgrid(x_vals, y_vals)

# final obligations q = seed + purchased
q0_final = SEEDING + X.astype(np.float64)
q1_final = SEEDING + Y.astype(np.float64)

cost = ls_lmsr_cost_vec(q0_final, q1_final, ALPHA)
delta_C = cost - base_cost  # net money flowing into cost function (what funds obligations)

# gross money traders pay (includes fee)
gross_budgets = delta_C / (1.0 - FEE)

# correct worst-case payout: total outstanding shares (include seed)
payout_worst = np.maximum(q0_final, q1_final).astype(np.float64)

# worst-case profit (what the market maker pockets after resolution, including collected fees)
profit_worst = gross_budgets - payout_worst

# margin relative to what's bet (gross budgets)
# guard divide-by-zero where gross_budgets == 0
with np.errstate(divide='ignore', invalid='ignore'):
    margin = np.where(gross_budgets > 0, profit_worst / gross_budgets, -np.inf)

red_mask = profit_worst < -50.0
green_mask = margin >= 0.04

# Build log edges
def log_edges(vals):
    logs = np.log(vals.astype(np.float64))
    mids = (logs[:-1] + logs[1:]) / 2.0
    first = logs[0] - (mids[0] - logs[0])
    last = logs[-1] + (logs[-1] - mids[-1])
    edges = np.concatenate([[first], mids, [last]])
    return np.exp(edges)

x_edges = log_edges(x_vals)
y_edges = log_edges(y_vals)

fig, ax = plt.subplots(figsize=(9, 7))

C_red = red_mask.astype(float)
cmap_red = ListedColormap(["white", "red"])
ax.pcolormesh(x_edges, y_edges, C_red, shading="auto", cmap=cmap_red, alpha=0.75)

C_green = green_mask.astype(float)
cmap_green = ListedColormap(["white", "green"])
ax.pcolormesh(x_edges, y_edges, C_green, shading="auto", cmap=cmap_green, alpha=0.5)

ax.set_xscale("log")
ax.set_yscale("log")
ax.set_xlabel("Shares bought for Outcome 1 (x)")
ax.set_ylabel("Shares bought for Outcome 2 (y)")
ax.set_title("Corrected LS-LMSR Worst-Case Profit Regions (seed included in payout)\nRed: worst-case loss | Green: worst-case win ≥ 4% of gross bet")

red_patch = mpatches.Patch(color="red", label="Loss (worst case)")
green_patch = mpatches.Patch(color="green", label="Win ≥ 4% (worst case)")
ax.legend(handles=[red_patch, green_patch], loc="lower right")

plt.tight_layout()
plt.show()

# Also produce a small diagnostic numeric summary
total_cells = X.size
num_loss = int(np.sum(red_mask))
num_good = int(np.sum(green_mask))
print(f"Grid cells: {total_cells}, losses: {num_loss} ({num_loss/total_cells:.1%}), wins≥4%: {num_good} ({num_good/total_cells:.1%})")
