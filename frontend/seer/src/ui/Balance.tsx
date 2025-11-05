import { Currency } from "@/lib/definitions";
import { useBalanceQuery } from "@/lib/queries/useBalanceQuery";
import NumberFlow from "@number-flow/react";
import { Wallet } from "lucide-react";
import DollarIcon from "../../public/icons/dollar.svg";
import Button from "./Button";

export default function Balance({ currency, ...rest }: { currency: Currency } & React.HTMLAttributes<HTMLDivElement>) {
    const { data: balance } = useBalanceQuery(currency);
    if (!balance) return null;

    return (
        <div className="flex items-center gap-3" {...rest}>
            <Button bg="bg-black" width="small" className="border border-gray-700">
                <div className="flex items-center gap-2">
                    <DollarIcon className="w-4 h-4" />
                    <span className="font-medium text-white">
                        <NumberFlow locales={"en-US"}
                            value={balance.balance.toNumber()}
                            format={{ style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }} />
                    </span>
                </div>
            </Button>

            <div className="flex justify-center lg:w-23.5 w-12 shrink-0">
                <Button bg="bg-neon-blue" width="full">
                    <div className="flex items-center lg:gap-1.5">
                        <Wallet strokeWidth={1.3} size={19} />
                        <span className="hidden lg:block">Wallet</span>
                    </div>
                </Button>
            </div>

        </div>
    );
}
