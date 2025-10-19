import { toast, ToastOptions } from "react-toastify";


export function toastStyled(message: string, options?: ToastOptions) {
  // toast.dismiss();
  toast(message, {
    className: "!bg-gray-800 text-sm !w-2xs !mt-5 lg:!mt-18",
    progressClassName: ` ${options?.type === "error" ? "!bg-red-500" : ""} `,
    autoClose: 2500,
    ...options,
  });
}

