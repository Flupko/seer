from mpmath import mp

mp.dps = 120  # 120 decimal precisions

def softmax_scaled(q, b, scale=10**20):
    r = [mp.mpf(qi)/b for qi in q]
    m = max(r)
    exps = [mp.e**(ri - m) for ri in r]
    Z = mp.fsum(exps)
    s = [ei/Z for ei in exps]
    for si in s:
        print(int(mp.floor(si*scale)), si * scale)
    return [int(mp.floor(si*scale)) for si in s]

# Example:
alpha = mp.mpf('0.123456')
q = [37_518_378_724 * 100, 1_234_311_111 * 100, 100 * 100, 123_123_123_444_322 * 100, 5, 1]
b = alpha * mp.fsum(q)
print(softmax_scaled(q, b))
