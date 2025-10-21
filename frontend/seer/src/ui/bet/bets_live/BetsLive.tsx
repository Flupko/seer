"use client";

import { useBetStore } from "@/lib/stores/bets";
import MenuHorizontal from "@/ui/menu_horizontal/MenuHorizontal";
import MenuHorizontalItem from "@/ui/menu_horizontal/MenuHorizontalItem";
import { useState } from "react";
import BetsLiveTable from "./BetsLiveTable";

export default function BetsLive() {


    const [selectedBetTab, setSelectedBetTab] = useState<"latest" | "high">("latest");

    const latestBets = useBetStore((s) => s.latestBets);
    const latestParity = useBetStore((s) => s.latestParity);


    const highBets = useBetStore((s) => s.highBets);
    const highParity = useBetStore((s) => s.highParity);


    return (
        <div className="flex flex-col gap-4">

            <MenuHorizontal>
                <MenuHorizontalItem active={selectedBetTab === "latest"} onClick={() => setSelectedBetTab("latest")}>Latest Bets</MenuHorizontalItem>
                <MenuHorizontalItem active={selectedBetTab === "high"} onClick={() => setSelectedBetTab("high")}>High Wagers</MenuHorizontalItem>
            </MenuHorizontal>

            {selectedBetTab === "latest" && <BetsLiveTable bets={latestBets} parity={latestParity} />}
            {selectedBetTab === "high" && <BetsLiveTable bets={highBets} parity={highParity} />}

        </div>

    );
}
