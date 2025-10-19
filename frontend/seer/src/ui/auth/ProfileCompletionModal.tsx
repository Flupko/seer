"use client"

import Button from "@/ui/Button";
import { SubmitHandler, useForm } from "react-hook-form";

import * as api from "@/lib/api";
import { ProfileCompletionFormValues, ProfileCompletionSchema } from "@/lib/definitions";
import { useModalStore } from "@/lib/stores/modal";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "motion/react";
import { useRouter } from "next/navigation";
import FormField from "../form/FormField";
import Terms from "./Terms";


export default function ProfileCompletionModal() {

    const router = useRouter();

    const {
        register,
        handleSubmit,
        formState: { errors },
        setError,
    } = useForm<ProfileCompletionFormValues>({
        resolver: zodResolver(ProfileCompletionSchema), // Apply the zodResolver
        mode: "onBlur", // Validate on blur
    });

    const queryClient = useQueryClient();
    const closeModal = useModalStore((state) => state.closeModal);


    const mutation = useMutation({
        mutationFn: api.completeProfile,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['user'] });
            router.push("/");
            closeModal();
        },
        onError: (error: api.APIError) => {
            if (error.errors) {
                error.errors.forEach(({ field, message }) => {
                    if (field in ProfileCompletionSchema.shape) {
                        setError(field as keyof ProfileCompletionFormValues, { message });
                    }
                });
            }

            if (error.message) {
                setError("root", { message: error.message });
            }
        },
    });

    const onSubmit: SubmitHandler<ProfileCompletionFormValues> = data => {
        mutation.mutate(data)
    };


    return (

        <div className="flex flex-col gap-10 px-7.5 lg:px-9.5 pb-8 md:pt-11 pt-6 w-full">

            <div className="flex flex-col gap-5">
                <h1 className="text-2xl font-extrabold">Last step to create your account...</h1>
                <p className="text-md text-gray-300">Please choose a username. You can edit it later in settings.</p>
            </div>


            <form className="flex flex-col gap-13" onSubmit={handleSubmit(onSubmit)}>
                <div className="flex flex-col gap-5">

                    <FormField
                        name="username"
                        label="Username*"
                        register={register}
                        error={errors.username}
                        placeholder="Enter Username"
                        type="text"
                    />

                    <Terms />

                </div>

                {errors.root &&
                    <motion.div className="text-sm font-semibold text-red-500 block text-center" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, ease: "easeIn" }}>
                        {errors.root.message}
                    </motion.div>}


                <Button bg="bg-neon-blue" width="full" height="large" type="submit" isLoading={mutation.isPending}>
                    Create Account
                </Button>
            </form>


        </div>)
}

