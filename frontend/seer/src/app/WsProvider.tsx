"use client";

import { Balance, MarketView } from "@/lib/definitions";
import { pricesForMarket } from "@/lib/lslmsr/lslmsr";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useBetStore } from "@/lib/stores/bets";
import { useChatStore } from "@/lib/stores/chats";
import { useOnlineStore } from "@/lib/stores/online";
import { BalanceUpdate, MarketUpdate } from "@/socket/messages";
import { getWSClient, WSClient } from "@/socket/socket";
import { useQueryClient } from "@tanstack/react-query";
import { createContext, useContext, useEffect, useState } from "react";


interface WsContext {
    wsClient: WSClient | null;
}


const WsContext = createContext<WsContext>({
    wsClient: null,
});

// Create custom hook
export function useWebSocket() {
    const context = useContext(WsContext);
    return context.wsClient;
}

export function WsProvider({ children }: { children: React.ReactNode }) {

    const [wsClient, setWsClient] = useState<WSClient | null>(null);
    const qc = useQueryClient();

    const betsLatest = useBetStore((state) => state.latestBets);
    const addLatestBet = useBetStore((state) => state.addLatestBet);
    const betsHigh = useBetStore((state) => state.highBets);
    const addHighBet = useBetStore((state) => state.addHighBet);

    const addChatMessage = useChatStore((state) => state.addChatMessage)

    const updateOnlineCount = useOnlineStore((state) => state.updateOnlineCount);

    const { data: user } = useUserQuery();

    useEffect(() => {
        const ws = getWSClient() // only runs in browser

        ws.on("markets_update", (marketUpdate: MarketUpdate) => {
            qc.setQueryData(['market', marketUpdate.marketID], (oldMarket: MarketView | null) => {
                if (!oldMarket) return oldMarket;
                if (oldMarket.version >= marketUpdate.marketVersion) {
                    return oldMarket; // Ignore outdated update
                }

                // Update each outcome immutably
                const updatedOutcomes = oldMarket.outcomes.map(outcome => {
                    const update = marketUpdate.outcomes.find(u => u.id === outcome.id);
                    if (!update) return outcome;
                    return { ...outcome, quantity: update.quantity };
                });

                const updatedMarket = { ...oldMarket, outcomes: updatedOutcomes, totalVolume: marketUpdate.totalVolume };
                pricesForMarket(updatedMarket);

                // Update charts for each outcome immutably
                const now = new Date();
                const outcomesWithCharts = updatedMarket.outcomes.map(outcome => {
                    if (!outcome.priceCharts) return outcome;

                    const updatedCharts = outcome.priceCharts.map(priceChart => {
                        let bucketDate = new Date();
                        switch (priceChart.timeframe) {
                            case '24h':
                                bucketDate = new Date(Math.floor(now.getTime() / (5 * 60 * 1000)) * (5 * 60 * 1000));
                                break;
                            case '7d':
                                bucketDate = new Date(Math.floor(now.getTime() / (60 * 60 * 1000)) * (60 * 60 * 1000));
                                break;
                            case '30d':
                                bucketDate = new Date(Math.floor(now.getTime() / (4 * 60 * 60 * 1000)) * (4 * 60 * 60 * 1000));
                                break;
                            case 'all':
                                bucketDate = new Date(Math.floor(now.getTime() / (24 * 60 * 60 * 1000)) * (24 * 60 * 60 * 1000));
                                break;
                        }

                        const prices = [...priceChart.prices];
                        const lastPoint = prices[prices.length - 1];
                        if (lastPoint && lastPoint.date.getTime() === bucketDate.getTime()) {
                            prices[prices.length - 1] = { ...lastPoint, price: outcome.priceYesNormalized };
                        } else {
                            prices.push({ timestamp: bucketDate.getTime(), date: bucketDate, price: outcome.priceYesNormalized });
                        }
                        return { ...priceChart, prices };
                    });

                    return { ...outcome, priceCharts: updatedCharts };
                });

                return { ...updatedMarket, outcomes: outcomesWithCharts, version: marketUpdate.marketVersion };
            });
        });

        ws.on("balance", (balanceUpdate: BalanceUpdate) => {
            qc.setQueryData(['balance', balanceUpdate.currency], (oldBalance: Balance | null) => {
                if (!oldBalance) return oldBalance;
                if (oldBalance.version >= balanceUpdate.version) {
                    return oldBalance; // Ignore outdated update
                }
                return balanceUpdate;
            });
        })

        ws.on("bets:latest", (betUpdate) => {
            addLatestBet(betUpdate);
        });

        ws.on("bets:high", (betUpdate) => {
            addHighBet(betUpdate);
        });

        ws.on("chat", (chatMessage) => {
            addChatMessage(chatMessage)
        });

        ws.on("online", (onlineUpdate) => {
            updateOnlineCount(onlineUpdate);
        });

        ws.onConnect(() => {
            ws.emit({ type: "subscribe:markets_update" });
            ws.emit({ type: "subscribe:online" });
            ws.emit({ type: "subscribe:bets" });
            ws.emit({ type: "subscribe:chat", payload: { chatSlug: "global" } });
        });

        setWsClient(ws)
    }, [])

    return (
        <WsContext.Provider value={{ wsClient: wsClient }}>
            {children}
        </WsContext.Provider>
    )
}