import { useState } from 'preact/hooks';
import { authService } from '../services/authService';
import { Button } from './Button.js';
import { Input } from './Input.js';
import { Card } from './Card.js';

interface LoginFormProps {
  onLogin: () => void;
  isFirstUser: boolean;
}

export function LoginForm({ onLogin, isFirstUser }: LoginFormProps) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      if (isFirstUser) {
        await authService.createFirstUser({ email, password });
      } else {
        await authService.login({ email, password });
      }
      onLogin();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        {/* Header */}
        <div className="text-center">
          <div className="mx-auto h-12 w-12 bg-blue-600 rounded-xl flex items-center justify-center shadow-lg">
            <svg className="h-8 w-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
          </div>
          <h2 className="mt-6 text-3xl font-bold text-gray-900 dark:text-white">
            {isFirstUser ? 'Welcome to Outway' : 'Welcome back'}
          </h2>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            {isFirstUser 
              ? 'Create your administrator account to get started'
              : 'Sign in to your account to continue'
            }
          </p>
        </div>

        {/* Form */}
        <Card className="shadow-2xl">
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="space-y-4">
              <Input
                label="Email"
                type="email"
                value={email}
                onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
                placeholder="Enter your email"
                required
                disabled={isLoading}
                {...(error && { error })}
              />

              <Input
                label="Password"
                type="password"
                value={password}
                onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
                placeholder="Enter your password"
                required
                disabled={isLoading}
              />
            </div>

            {error && (
              <div className="rounded-md bg-red-50 dark:bg-red-900/20 p-4 border border-red-200 dark:border-red-800">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <svg className="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                    </svg>
                  </div>
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                      {error}
                    </h3>
                  </div>
                </div>
              </div>
            )}

            <Button
              type="submit"
              variant="primary"
              size="lg"
              fullWidth
              loading={isLoading}
              disabled={!email || !password}
            >
              {isLoading ? 'Please wait...' : (isFirstUser ? 'Create Account' : 'Sign In')}
            </Button>
          </form>
        </Card>

        {/* Footer */}
        <div className="text-center">
          <p className="text-xs text-gray-500 dark:text-gray-400">
            {isFirstUser 
              ? 'This will create the first administrator account'
              : 'Secure access to your Outway dashboard'
            }
          </p>
        </div>
      </div>
    </div>
  );
}