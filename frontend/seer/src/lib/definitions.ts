import { z } from "zod";


const usernameSchema = z
  .string()
  .min(3, { message: "Username must be at least 3 characters" })
  .max(15, { message: "Username must be at most 15 characters" })
  .regex(/^[a-z0-9]+$/i, { message: "Username must be alphanumeric (A–Z, a-z, 0–9) only" });

const emailSchema = z.string().min(1, "Email is required").email({ message: "Email is invalid" });

const statusSchema = z.enum(['pending_email_verification', 'activated']);

const passwordSchema = z
  .string()
  .min(8, "Password must be at least 8 characters")
  .max(49, "Password must be at most 49 characters");


export const UserSchema = z.object({
  id: z.string().uuid(),
  email: emailSchema,
  username: usernameSchema,
  profileImageUrl: z.url().optional(),
  status: statusSchema,
  balance: z.number().min(0),
});

export type User = z.infer<typeof UserSchema>;

export const RegisterSchema = z.object({
  username: usernameSchema,
  email: emailSchema,
  password: passwordSchema,
});


export const LoginSchema = z.object({
  login: z.union([emailSchema, usernameSchema]),
  password: z.string().min(1, "Password is required"),
});

export const ProfileCompletionSchema = z.object({
  username: usernameSchema,
});

export type RegisterFormValues = z.infer<typeof RegisterSchema>;
export type LoginFormValues = z.infer<typeof LoginSchema>;
export type ProfileCompletionFormValues = z.infer<typeof ProfileCompletionSchema>;