import { motion } from "motion/react";
import Loader from "./loader/Loader";

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  width: "small" | "full";
  height?: "small" | "large";
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
    small: "px-3.5",
    full: "w-full",
  };

  const heightClasses = {
    small: "h-12",
    large: "h-13.5",
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
        font-semibold 
        rounded-md
        transition-all 
        duration-200

        hover:brightness-115

        active:scale-95 
        active:brightness-80
        
        disabled:opacity-50
        disabled:hover:brightness-100
        disabled:active:scale-100

        ${!isDisabled ? "cursor-pointer" : ""}
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
