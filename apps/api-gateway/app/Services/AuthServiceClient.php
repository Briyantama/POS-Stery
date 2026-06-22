<?php

namespace App\Services;

class AuthServiceClient extends BaseGrpcGatewayClient
{
    public function __construct()
    {
        parent::__construct('auth_service_url');
    }

    public function login(string $email, string $password, string $storeId = ''): array
    {
        return $this->post('/v1/auth/login', compact('email', 'password', 'storeId'));
    }

    public function validate(string $token): array
    {
        return $this->post('/v1/auth/validate', compact('token'));
    }

    public function logout(string $accessToken, string $refreshToken = ''): array
    {
        // auth.proto LogoutRequest expects access_token / refresh_token.
        return $this->post('/v1/auth/logout', [
            'access_token'  => $accessToken,
            'refresh_token' => $refreshToken,
        ]);
    }
}
