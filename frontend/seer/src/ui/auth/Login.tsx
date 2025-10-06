"use client"

import Button from "@/ui/Button";
import Input from "@/ui/Input";
import Providers from "./Providers";
import Password from "./Password";
import Link from "next/link";
import { useForm, SubmitHandler } from "react-hook-form"

import { LoginFormValues, LoginSchema } from "@/lib/definitions";
import { zodResolver } from "@hookform/resolvers/zod";
import FormField from "./FormField";
import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useModal } from "@/ui/modal/Modal";
import * as api from "@/lib/api";
import { motion } from "motion/react";


export default function Login() {

    const router = useRouter();

    const {
        register,
        handleSubmit,
        formState: { errors },
        setError,
    } = useForm<LoginFormValues>({
        resolver: zodResolver(LoginSchema), // Apply the zodResolver
        mode: "onBlur", // Validate on blur
    });

    const queryClient = useQueryClient();
    const { closeModal } = useModal();


    const mutation = useMutation({
        mutationFn: api.login,
        onSuccess: () => {
            console.log("Login successful");
            queryClient.invalidateQueries({ queryKey: ['user'] });
            router.push("/");
            closeModal();
        },
        onError: (error: api.APIError) => {
            if (error.errors) {
                error.errors.forEach(({ field, message }) => {
                    if (field in LoginSchema.shape) {
                        setError(field as keyof LoginFormValues, { message });
                    }
                });
            }

            if (error.message) {
                setError("root", { message: error.message });
            }
        },
    });

    const onSubmit: SubmitHandler<LoginFormValues> = data => {
        mutation.mutate(data)
    };


    return (
        <div className="flex flex-col gap-8">

            <form className="flex flex-col gap-10" onSubmit={handleSubmit(onSubmit)}>
                <div className="flex flex-col gap-5">

                    <FormField
                        name="login"
                        label="Email or Username*"
                        register={register}
                        error={errors.login}
                        placeholder="Enter Email or Username"
                        type="text"
                    />

                    <Password
                        name="password"
                        label="Password*"
                        register={register}
                        error={errors.password}
                        placeholder="Enter Password"
                    />

                    
                </div>

                {errors.root &&
                        <motion.div className="text-sm font-semibold text-red-500 block text-center" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, ease: "easeIn" }}>
                            {errors.root.message}
                        </motion.div>}


                <Button bg="bg-neon-blue" width="full" height="large" type="submit" isLoading={mutation.isPending}>
                    Login
                </Button>
            </form>


            <Providers />

        </div>)
}

