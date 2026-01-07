// src/routes/index.tsx
import { Routes, Route, Navigate } from 'react-router-dom';
import { lazy, Suspense } from 'react';
import { ProtectedRoute } from '@/components/auth/protected-route';

// Shared Pages
const LoginPage = lazy(() => import('@/page/globals/login/page'));
const AuthCallbackPage = lazy(() => import('@/page/auth/callback/page'));

// Main Pages
const DashboardPage = lazy(() => import('@/page/admin/dashboard/page'));
const GalleryPage = lazy(() => import('@/page/gallery/page'));
const FaceSearchPage = lazy(() => import('@/page/face-search/page'));
const NewsWriterPage = lazy(() => import('@/page/news-writer/page'));
const SettingsPage = lazy(() => import('@/page/settings/page'));

// Layouts
import RootLayout from '@/layouts/root-layout';
import PageLayout from '@/layouts/page';

// Loading component
const LoadingFallback = () => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="text-center">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      <p className="mt-2">Loading...</p>
    </div>
  </div>
);

const AppRoutes = () => {
  return (
    <Suspense fallback={<LoadingFallback />}>
      <Routes>
      <Route element={<RootLayout />}>
        {/* Auth routes */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/auth/callback" element={<AuthCallbackPage />} />

        {/* Root redirect */}
        <Route path="/" element={<Navigate to="/dashboard" replace />} />

        {/* Protected routes with layout */}
        <Route element={<ProtectedRoute><PageLayout /></ProtectedRoute>}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/gallery" element={<GalleryPage />} />
          <Route path="/face-search" element={<FaceSearchPage />} />
          <Route path="/news-writer" element={<NewsWriterPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>

        {/* 404 route */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Route>
      </Routes>
    </Suspense>
  );
};

export default AppRoutes;