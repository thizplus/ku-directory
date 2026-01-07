// src/layouts/root-layout.tsx
import React from 'react';
import { Outlet } from 'react-router-dom';
import { Toaster } from "@/components/ui/sonner"


/**
 * Root Layout ที่ครอบคลุมทุกหน้าในแอปพลิเคชัน
 */
const RootLayout: React.FC = () => {
 

  return (
    <div className="min-h-svh bg-background text-foreground antialiased">
      <Outlet />
      <Toaster style={{
        fontFamily: 'Roboto, "Noto Sans Thai", sans-serif',
      }} position="top-right" richColors />
     
    </div>
  );
};

export default RootLayout;