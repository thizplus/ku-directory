import { Link } from "react-router-dom";
import { FolderSearch } from "lucide-react";
import { LoginForm } from "@/components/page/login/login-form";
import { LoginAnimation } from "@/components/page/login/login-animation";

export default function LoginPage() {
  return (
    <div className="grid min-h-svh lg:grid-cols-2">
      {/* Left side - Login form */}
      <div className="flex flex-col gap-4 p-6 md:p-10">
        <div className="flex justify-between items-center">
          <Link to="/" className="flex items-center gap-2 font-medium">
            <div className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-md">
              <FolderSearch className="size-4" />
            </div>
            KU DIRECTORY
          </Link>
        </div>
        <div className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-xs">
            <LoginForm />
          </div>
        </div>
      </div>

      {/* Right side - Animation */}
      <div className="relative hidden lg:block overflow-hidden">
        <LoginAnimation />
      </div>
    </div>
  );
}
