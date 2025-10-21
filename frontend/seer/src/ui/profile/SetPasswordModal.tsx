"use client"

import Button from "@/ui/Button";
import { SubmitHandler, useForm } from "react-hook-form";

import * as api from "@/lib/api";
import { SetPasswordFormValues, SetPasswordPayloadSchema, SetPasswordSchema } from "@/lib/definitions";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { motion } from "motion/react";
import { useRouter } from "next/navigation";
import Password from "../form/Password";


export default function SetPasswordModal() {

    const router = useRouter();

    const {
        register,
        handleSubmit,
        formState: { errors },
        setError,
    } = useForm<SetPasswordFormValues>({
        resolver: zodResolver(SetPasswordSchema), // Apply the zodResolver
        mode: "onBlur", // Validate on blur
    });

    const mutation = useMutation({
        mutationFn: api.setPassword,
        onSuccess: () => {
            window.location.replace('/');
        },
        onError: (error: api.APIError) => {
            if (error.errors) {
                error.errors.forEach(({ field, message }) => {
                    if (field in SetPasswordSchema.shape) {
                        setError(field as keyof SetPasswordFormValues, { message });
                    }
                });
            }

            if (error.message) {
                setError("root", { message: error.message });
            }
        },
    });

    const onSubmit: SubmitHandler<SetPasswordFormValues> = data => {
        const payload = SetPasswordPayloadSchema.parse(data);
        mutation.mutate(payload)
    };


    return (

        <div className="flex flex-col gap-12 px-7.5 lg:px-9.5 pb-8 md:pt-11 pt-6 w-full">

            <div className="flex flex-col gap-5">
                <h1 className="text-2xl font-extrabold">Set Password</h1>
                <p className="text-sm text-white mt-1">Setting your password will log you out from all your sessions.</p>
            </div>


            <form className="flex flex-col gap-15" onSubmit={handleSubmit(onSubmit)}>
                <div className="flex flex-col gap-5">


                    <Password name="password"
                        label="Password*"
                        register={register}
                        error={errors.password}
                        placeholder="Enter Password"
                    />

                    <Password name="confirmPassword"
                        label="Confirm Password*"
                        register={register}
                        error={errors.confirmPassword}
                        placeholder="Confirm Password"
                    />

                </div>

                {errors.root &&
                    <motion.div className="text-sm font-semibold text-error block text-center" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, ease: "easeIn" }}>
                        {errors.root.message}
                    </motion.div>}


                <Button bg="bg-neon-blue" width="full" height="large" type="submit" isLoading={mutation.isPending}>
                    Set Password
                </Button>
            </form>


        </div>)
}

