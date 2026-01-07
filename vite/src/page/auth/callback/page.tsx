import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth, getCurrentUser } from '@/hooks/use-auth'

export default function AuthCallbackPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { setAuth, setLoading } = useAuth()

  useEffect(() => {
    const handleCallback = async () => {
      const token = searchParams.get('token')
      const redirect = searchParams.get('redirect') || '/dashboard'
      const error = searchParams.get('error')

      if (error) {
        console.error('Auth error:', error)
        navigate('/login?error=' + error)
        return
      }

      if (!token) {
        console.error('No token received')
        navigate('/login?error=no_token')
        return
      }

      try {
        setLoading(true)
        const user = await getCurrentUser(token)
        setAuth(token, user)
        navigate(redirect)
      } catch (err) {
        console.error('Failed to get user:', err)
        navigate('/login?error=auth_failed')
      }
    }

    handleCallback()
  }, [searchParams, setAuth, setLoading, navigate])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto"></div>
        <p className="mt-4 text-muted-foreground">Signing you in...</p>
      </div>
    </div>
  )
}
