import Input from "@/ui/Input"
import { Eye, EyeClosed, EyeOff } from "lucide-react"
import { useState } from "react"
import FormField from "./FormField"

export default function Password({...props}: any) {

    const [passwordShown, setPasswordShown] = useState(false)

    const eyeIcon = passwordShown
        ? <Eye className="text-white h-full w-5.5 cursor-pointer" strokeWidth={1.4} onClick={() => setPasswordShown(false)} />
        : <EyeOff className="text-white h-full w-5.5 cursor-pointer" strokeWidth={1.4} onClick={() => setPasswordShown(true)} />

    return (
        <FormField {...props} rightEl={eyeIcon} type={passwordShown ? "text" : "password"} autoComplete="on"></FormField>
    )
}