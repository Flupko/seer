import { MarketView } from "./definitions";


export function isMarketActive(market: MarketView): boolean {
    return market.status === 'active' && (!market.closeTime || market.closeTime.getTime() > new Date().getTime());
}

export function isMarketPending(market: MarketView): boolean {
    return market.status === 'pending' || (market.closeTime ? market.closeTime.getTime() <= new Date().getTime() : false);
}