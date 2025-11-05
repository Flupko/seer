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

        console.log("WsProvider mounting, subscribing to ws events");

        ws.on("markets_update", (marketUpdate: MarketUpdate) => {
            qc.setQueryData(['market', marketUpdate.marketID], (oldMarket: MarketView | null) => {
                if (!oldMarket) return oldMarket;
                if (oldMarket.version >= marketUpdate.marketVersion) {
                    return oldMarket; // Ignore outdated update
                }

                // Update each outcome
                for (const updatedOutcome of marketUpdate.outcomes) {
                    const outcome = oldMarket.outcomes.find(o => o.id === updatedOutcome.id);
                    if (outcome) {
                        outcome.quantity = updatedOutcome.quantity;
                    }
                }

                pricesForMarket(oldMarket);

                // Update charts for each outcome
                // Add new price point (replace last point if same timestamp)
                const now = new Date();
                for (const outcome of oldMarket.outcomes) {
                    if (!outcome.priceCharts) continue;
                    for (const priceChart of outcome.priceCharts) {
                        // Depending on the chart interval, we compute current timestamp bucket
                        let bucketDate = new Date();
                        switch (priceChart.timeframe) {

                            case '24h':
                                // 5 minutes buckets
                                bucketDate = new Date(Math.floor(now.getTime() / (5 * 60 * 1000)) * (5 * 60 * 1000));
                                break;

                            case '7d':
                                // 1h buckets
                                bucketDate = new Date(Math.floor(now.getTime() / (60 * 60 * 1000)) * (60 * 60 * 1000));
                                break;

                            case '30d':
                                // 4h buckets
                                bucketDate = new Date(Math.floor(now.getTime() / (4 * 60 * 60 * 1000)) * (4 * 60 * 60 * 1000));
                                break;
                            case 'all':
                            // TODO

                        }

                        const lastPoint = priceChart.prices[priceChart.prices.length - 1];
                        if (lastPoint.date.getTime() === bucketDate.getTime()) {
                            // Replace last point
                            lastPoint.price = outcome.price;
                            console.log("Replacing last price point for chart ", priceChart.timeframe, lastPoint, "outcome price:", outcome.price, "outcome", outcome);
                        } else {
                            // Add new point
                            console.log("Replacing last price point for chart ", priceChart.timeframe, lastPoint, "outcome price:", outcome.price, "outcome", outcome);
                            priceChart.prices.push({ timestamp: bucketDate.getTime(), date: bucketDate, price: outcome.price });
                        }

                    }
                }

                // Update total volume
                oldMarket.totalVolume = marketUpdate.totalVolume;

                return { ...oldMarket, version: marketUpdate.marketVersion };
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