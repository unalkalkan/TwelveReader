import { IconShieldExclamation } from '@tabler/icons-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export function ForbiddenPage() {
  const navigate = useNavigate();
  const { logout } = useAuth();

  return (
    <div className="page">
      <div className="container-center container container-tight py-4">
        <div className="card card-md text-center" style={{ maxWidth: '480px', margin: '3rem auto' }}>
          <div className="card-body">
            <div className="mb-3">
              <IconShieldExclamation size={48} className="text-danger" />
            </div>
            <h2 className="card-title mb-2">Access Denied</h2>
            <p className="text-secondary mb-3">
              This dashboard requires admin privileges. Your current account does not have the necessary permissions to view debug and administration tools.
            </p>
            <div className="d-flex gap-2 justify-content-center">
              <button className="btn btn-link" onClick={() => navigate('/login')}>
                Sign in with a different account
              </button>
              <button className="btn btn-secondary btn-sm" onClick={logout}>
                Sign out
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
