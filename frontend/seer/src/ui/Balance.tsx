import { Currency } from "@/lib/definitions";
import { useBalanceQuery } from "@/lib/queries/useBalanceQuery";
import NumberFlow from "@number-flow/react";
import { Wallet } from "lucide-react";
import Image from "next/image";
import Button from "./Button";

export default function Balance({ currency, ...rest }: { currency: Currency } & React.HTMLAttributes<HTMLDivElement>) {
    const { data: balance } = useBalanceQuery(currency);
    if (!balance) return null;

    return (
        <div className="flex items-center gap-3" {...rest}>
            <Button bg="bg-black" width="small" className="border border-gray-700">
                <div className="flex items-center gap-2.5">
                    <Image src={"/icons/dollar.svg"} alt="Dollar" width={16} height={16} />
                    <span className="font-medium text-white">
                        <NumberFlow locales={"en-US"} value={balance.balance.toNumber()} format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }} />
                    </span>
                </div>
            </Button>

            <Button bg="bg-neon-blue" width="small">
                <div className="flex items-center lg:gap-2">
                    <Wallet strokeWidth={1.3} size={20} />
                    <span className="hidden lg:block">Wallet</span>
                </div>
            </Button>

        </div>
    );
}
