"use client";

import { Balance, MarketView } from "@/lib/definitions";
import { pricesForMarket } from "@/lib/lslmsr/lslmsr";
import { useBetStore } from "@/lib/stores/bets";
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

    useEffect(() => {
        const ws = getWSClient() // only runs in browser

        console.log("WsProvider mounting, subscribing to ws events");
        ws.onConnect(() => {
            ws.emit({ type: "subscribe:markets_update" });
            ws.emit({ type: "subscribe:online_count" });
            ws.emit({ type: "subscribe:bets" });
        });

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
            console.log("Received latest bet update:", betUpdate);
            addLatestBet(betUpdate);
        });

        ws.on("bets:high", (betUpdate) => {
            addHighBet(betUpdate);
        });



        setWsClient(ws)
    }, [])

    return (
        <WsContext.Provider value={{ wsClient: wsClient }}>
            {children}
        </WsContext.Provider>
    )
}