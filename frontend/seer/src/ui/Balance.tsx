import { Wallet } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import Button from "./Button";

export default function Balance({ ...rest }: React.HTMLAttributes<HTMLDivElement>) {
    const [balance, setBalance] = useState(124.3);

    return (
        <div className="flex items-center gap-3" {...rest}>
            <Button bg="bg-black" width="small" className="border border-gray-700">
                <div className="flex items-center gap-2">
                    <Image src={"/icons/dollar.svg"} alt="Dollar" width={16} height={16} />
                    <span className="font-medium text-white">
                        {balance.toFixed(2)}
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
