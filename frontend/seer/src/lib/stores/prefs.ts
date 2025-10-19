// lib/prefs.ts
import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { OddsFormat } from '../odds'


type PrefsState = {
    oddsFormat: OddsFormat
    setOddsFormat: (fmt: OddsFormat) => void
}

export const usePrefs = create<PrefsState>()(persist(
    (set) => ({
        oddsFormat: 'decimal',
        setOddsFormat: (oddsFormat) => set({ oddsFormat }),
    }),
    {
        name: 'prefs', // localStorage key
        storage: createJSONStorage(() => localStorage),
        // partialize: (s) => ({ oddsFormat: s.oddsFormat, locale: s.locale }),
    },
),
)
