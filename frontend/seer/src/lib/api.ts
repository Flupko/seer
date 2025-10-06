import { LoginFormValues, ProfileCompletionFormValues, RegisterFormValues, UserSchema } from "@/lib/definitions";
import { User } from "@/lib/definitions";
import { toast } from "react-toastify";
import z from "zod";

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || ""

export interface APIError {
  message: string;
  errors?: Array<{ field: string; message: string }>;
}

export const register = async (formData: RegisterFormValues) => {
  const response = await fetch(`${API_BASE_URL}/auth/register`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();

  if (!response.ok) {

    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;
};

export const login = async (formData: LoginFormValues) => {

  const response = await fetch(`${API_BASE_URL}/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();

  if (!response.ok) {
    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;

}

const UserResponseSchema = z.object({
  user: z.union([UserSchema, z.null()]),
});

export const getUser = async (): Promise<User | null> => {
  const response = await fetch(`${API_BASE_URL}/user/me`, {
    credentials: "include",
  });

  const data = await response.json();

  if (!response.ok) {
    if (response.status === 401) {
      return null;
    }

    const error: APIError = {
      message: data.message || "Failed to fetch user",
    };
    throw error;
  }

  // Validate response
  const result = UserResponseSchema.safeParse(data);
  
  if (!result.success) {
    throw new Error(`Invalid user data`);
  }

  return result.data.user;
};

export const completeProfile = async (formData: ProfileCompletionFormValues) => {

  const response = await fetch(`${API_BASE_URL}/auth/complete-profile`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(formData),
  });

  const data = await response.json();
  
  if (!response.ok) {
    const error: APIError = {
      message: data.message,
      errors: data.errors,
    };
    throw error;
  }

  return null;
}

export const logout = async () => {
    const response = await fetch(`${API_BASE_URL}/auth/logout`, {
        method: "POST",
        credentials: "include", // Important: sends cookies
    });

    if (!response.ok) {
        const data = await response.json();
        const error: APIError = {
            message: data.message || "Logout failed",
        };
        throw error;
    }

    return response.json();
};