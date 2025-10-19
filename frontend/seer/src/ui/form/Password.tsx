import { Eye, EyeOff } from "lucide-react"
import { useState } from "react"
import FormField from "../form/FormField"

export default function Password({ ...props }: any) {

    const [passwordShown, setPasswordShown] = useState(false)

    const eyeIcon = passwordShown
        ? <Eye className="text-white h-full w-4.5 cursor-pointer" strokeWidth={1.4} onClick={() => setPasswordShown(false)} />
        : <EyeOff className="text-white h-full w-4.5 cursor-pointer" strokeWidth={1.4} onClick={() => setPasswordShown(true)} />

    return (
        <FormField {...props} rightEl={eyeIcon} type={passwordShown ? "text" : "password"} autoComplete="on"></FormField>
    )
}