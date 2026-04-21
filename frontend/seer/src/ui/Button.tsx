import { ButtonHTMLAttributes } from "react";
import Loader from "./loader/Loader";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  width: "small" | "full";
  height?: "small" | "extraSmall" | "large";
  bg: string;
  children: React.ReactNode;
  onClick?: () => void;
  className?: string;
  isLoading?: boolean;
  disabled?: boolean;
};

export default function Button({
  width,
  height = "small",
  bg,
  children,
  onClick,
  className,
  isLoading = false,
  disabled = false,
  ...rest
}: ButtonProps) {
  const widthClasses = {
    small: "px-3",
    full: "w-full",
  };

  const heightClasses = {
    extraSmall: "h-9",
    small: "h-11",
    large: "h-13",
  };

  const isDisabled = isLoading || disabled;

  return (
    <button
      className={`
        ${heightClasses[height]} 
        ${widthClasses[width]} 
        ${bg}
        select-none 
        text-sm 
        text-white 
        font-medium 
        rounded-lg
        transition-all 
        duration-100
        shrink-0

        flex
        items-center
        justify-center

        hover:brightness-115

        active:scale-95 
        active:brightness-80
        
        disabled:brightness-50
        disabled:hover:brightness-50
        disabled:active:scale-100

        enabled:cursor-pointer
        ${className || ""}
      `}
      onClick={onClick}
      disabled={isDisabled}
      {...rest}
    >
      {isLoading ? (
        <div className="flex items-center justify-center">
          <Loader />
        </div>
      ) : (
        children
      )}
    </button>
  );
}
