"use client"

import { getUserPreferences, updateUserPreferences } from "@/lib/api";
import { UpdateUserPreferences } from "@/lib/definitions";
import { OddsFormat } from "@/lib/odds";
import { usePrefs } from "@/lib/stores/prefs";
import Loader from "@/ui/loader/Loader";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import Switch from "@/ui/switch/Switch";
import { toastStyled } from "@/ui/Toast";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Container from "../Container";
import PreferenceContainer from "./PreferenceContainer";

export default function SettingsPage() {
    const queryClient = useQueryClient()

    const { data: preferences, isPending } = useQuery({
        queryKey: ['user', 'preferences'],
        queryFn: getUserPreferences,
    })

    const mutation = useMutation({
        mutationFn: updateUserPreferences,
        onMutate: async (newPrefs: UpdateUserPreferences) => {
            await queryClient.cancelQueries({ queryKey: ['user', 'preferences'] });

            const previousPrefs = queryClient.getQueryData(['user', 'preferences']);
            queryClient.setQueryData(['user', 'preferences'], (old: any) => ({
                ...old,
                ...newPrefs
            }));

            return { previousPrefs };
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: ['user', 'preferences'] });
        },
        onSuccess: () => {
            toastStyled("Preferences updated", { type: "success", autoClose: 1500 });
            queryClient.invalidateQueries({ queryKey: ['user', 'preferences'] });
        },
        onError: () => {
            toastStyled("Failed to update preferences", { type: "error" });
        },
    });


    const oddsFormat = usePrefs((s) => s.oddsFormat)
    const setOddsFormat = usePrefs((s) => s.setOddsFormat)

    return (
        <Container title="Preferences">

            {isPending || !preferences ? (
                <div className="w-full h-20 flex items-center justify-center">
                    <Loader size={0.8} />
                </div>

            ) : <>
                <PreferenceContainer>

                    <div className="flex flex-col sm:flex-row justify-between sm:items-center gap-6 w-full">

                        <div className="flex flex-col gap-1.5">
                            <span className="font-bold text-sm">Odds Format</span>
                            <span className="text-gray-400 text-xs">Odds will be displayed using this format.</span>
                        </div>

                        <div className="w-full sm:w-40">
                            <MenuVertical leftPart={""} value={oddsFormat} onChange={(v) => setOddsFormat(v as OddsFormat)}
                                choices={[
                                    { value: "decimal", element: "Decimal" },
                                    { value: "american", element: "American" },
                                    { value: "fractional", element: "Fractional" },
                                    { value: "percent", element: "Percent" }
                                ]} />
                        </div>



                    </div>


                </PreferenceContainer>

                <PreferenceContainer>

                    <div className="flex items-center gap-6">

                        <Switch isOn={preferences.hidden} toggle={() => {
                            mutation.mutate({ hidden: !preferences.hidden })
                        }} disableToggle={mutation.isPending} />

                        <div className="flex flex-col gap-1.5">
                            <span className="font-bold text-sm">Hidden Mode</span>
                            <span className="text-gray-400 text-xs">Other users won't be able to see your wagers.</span>
                        </div>

                    </div>


                </PreferenceContainer>

                <PreferenceContainer>

                    <div className="flex items-center gap-6">

                        <Switch isOn={preferences.receiveMarketingEmails} toggle={() => {
                            mutation.mutate({ receiveMarketingEmails: !preferences.receiveMarketingEmails })
                        }} disableToggle={mutation.isPending} />

                        <div className="flex flex-col gap-1.5">
                            <span className="font-bold text-sm">Receive Marketting Emails</span>
                            <span className="text-gray-400 text-xs">Receive emails for promotions and updates.</span>
                        </div>

                    </div>


                </PreferenceContainer>
            </>}

        </Container>
    )
}