import { FieldError, FieldValue, FieldValues, Path, RegisterOptions, UseFormRegister } from "react-hook-form";
import Input, { InputProps } from "../Input";
import { motion } from "motion/react";


type FormFieldProps<T extends FieldValues> = {
    name: Path<T>;
    label: string;
    register: UseFormRegister<T>;
    error?: FieldError;
} & Omit<InputProps, 'hasError'>;


export default function FormField<T extends FieldValues>({ name,
    label,
    register,
    error,
    ...inputProps }: FormFieldProps<T>) {
    return (
        <div>
            {label && <label className="text-xs font-semibold text-white mb-1 block">{label}</label>}
            <Input id={name} hasError={error?.message != undefined} {...register(name)} {...inputProps} />
            {error &&
                <motion.label className="text-xs font-semibold text-red-500 mt-2 block" initial={{ opacity: 0}} animate={{ opacity: 1}} transition={{ duration: 0.3, ease: "easeIn" }}>
                    {error.message}
                </motion.label>}
        </div>
    );
}

