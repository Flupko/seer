import { Wallet } from "lucide-react";
import { useState } from "react";
import Button from "./Button";
import { div } from "motion/react-client";

export default function Balance({...rest}: React.HTMLAttributes<HTMLDivElement>) {
    const [balance, setBalance] = useState(51.28);

    return (
        <div className="flex items-center gap-3" {...rest}>
            <Button bg="bg-gray-900" width="small" className="border border-gray-600">
                <span className="font-numbers font-medium text-md">
                    <span className="text-sm text-white font-light mr-1">$</span>
                    {balance.toFixed(2)}
                </span>
            </Button>

            <Button bg="bg-neon-blue" width="small">
                <div className="flex items-center lg:gap-2">
                    <Wallet strokeWidth={0.9} size={20}/>
                    <span className="hidden lg:block">Wallet</span>
                </div>
            </Button>

        </div>
    );
}
