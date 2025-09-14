INIT_SEED_HANDLE = 10
HOUSE_EDGE = 0.04

liability = [0] * 2
handle = INIT_SEED_HANDLE


def smoothing_function(stake):
    return stake + 2


while True:

    user_input = input("Choose outcome 0/1 and a bet amount: ").split(" ")
    if len(user_input) != 2:
        continue

    side_str, stake_str = user_input
    side, stake = -1, -1

    try:
        side = int(side_str)
        stake = int(stake_str)
    except ValueError:
        continue

    max_odd = (1-HOUSE_EDGE) + (handle * (1 - HOUSE_EDGE) - liability[side]) / (2 * smoothing_function(stake))
    print("MAX_ODD =", max_odd)
    if input("confirm ? (y/n): ") == "y":
        handle += stake
        liability[side] += max_odd * stake
