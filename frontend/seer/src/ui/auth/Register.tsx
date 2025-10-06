"use client"

import Button from "@/ui/Button";
import Input from "@/ui/Input";
import Providers from "./Providers";
import Password from "./Password";
import Link from "next/link";
import { useForm, SubmitHandler } from "react-hook-form"

import { RegisterSchema, RegisterFormValues } from "@/lib/definitions";
import { zodResolver } from "@hookform/resolvers/zod";
import FormField from "./FormField";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import * as api from "@/lib/api";
import { redirect, useRouter } from "next/navigation";
import { useModal } from "../modal/Modal";
import Terms from "./Terms";


export default function Register() {

    const router = useRouter();

    const {
        register,
        handleSubmit,
        setError,
        formState: { errors },
    } = useForm<RegisterFormValues>({
        resolver: zodResolver(RegisterSchema), // Apply the zodResolver
        mode: "onBlur", // Validate on blur
    });

    const queryClient = useQueryClient();
    const {closeModal} = useModal();


    const mutation = useMutation({
        mutationFn: api.register,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['user'] });
            router.push("/");
            closeModal();
        },
        onError: (error: api.APIError) => {
            if (error.errors) {
                error.errors.forEach(({ field, message }) => {
                    if (field in RegisterSchema.shape) {
                        setError(field as keyof RegisterFormValues, { message });
                    }
                });
            }

            if (error.message) {
                setError("root", { message: error.message });
            }
        },
    });

    const onSubmit: SubmitHandler<RegisterFormValues> = data => {
        mutation.mutate(data)
    };

    return (
        <div className="flex flex-col gap-8">

            <form className="flex flex-col gap-10" onSubmit={handleSubmit(onSubmit)}>
                <div className="flex flex-col gap-5">

                    <FormField
                        name="email"
                        label="Email*"
                        register={register}
                        error={errors.email}
                        placeholder="Enter Email"
                        type="text"
                    />

                    <FormField
                        name="username"
                        label="Username*"
                        register={register}
                        error={errors.username}
                        type="text"
                        placeholder="Enter Username"
                    />



                    <Password
                        name="password"
                        label="Password*"
                        register={register}
                        error={errors.password}
                        placeholder="Enter Password"
                    />

                    <Terms />
                </div>


                <Button bg="bg-neon-blue" width="full" height="large" type="submit" isLoading={mutation.isPending}>
                    Register
                </Button>
            </form>


            <Providers />

        </div>)
}

