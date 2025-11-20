import { toast, ToastOptions } from "react-toastify";


export function toastStyled(message: string, options?: ToastOptions, autoClose = 2500) {
  toast.dismiss();
  toast(message, {
    className: "!bg-gray-800 text-sm !w-2xs !mt-5 lg:!mt-18",
    progressClassName: ` ${options?.type === "error" ? "!bg-error !text-error" : "!bg-success !text-success"} `,
    autoClose,
    delay: 0,
    ...options,
  });
}

