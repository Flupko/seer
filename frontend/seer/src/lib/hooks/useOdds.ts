// hooks/useOdds.ts
import { formatOdds } from '@/lib/odds';
import { usePrefs } from '@/lib/stores/prefs';
import { Decimal } from "decimal.js";

export const useOdds = () => {
    const oddsFormat = usePrefs((state) => state.oddsFormat);

    return {
        format: (prob: Decimal) => formatOdds(prob, oddsFormat),
        oddsFormat,
    };
};
