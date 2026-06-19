<?php

namespace App\Http\Middleware;

use Closure;
use Firebase\JWT\JWT;
use Firebase\JWT\Key;
use Firebase\JWT\ExpiredException;
use Firebase\JWT\SignatureInvalidException;
use Illuminate\Http\Request;
use Illuminate\Http\JsonResponse;
use RuntimeException;

class JwtValidation
{
    public function handle(Request $request, Closure $next): mixed
    {
        $token = $this->extractToken($request);

        if ($token === null) {
            return response()->json(['message' => 'Unauthenticated.'], 401);
        }

        try {
            $publicKeyPath = config('services.jwt_public_key_path');
            $publicKey = file_get_contents($publicKeyPath);

            if ($publicKey === false) {
                throw new RuntimeException('Cannot read JWT public key');
            }

            $decoded = JWT::decode($token, new Key($publicKey, 'RS256'));

            $request->merge([
                'auth_user_id'   => $decoded->sub ?? '',
                'auth_tenant_id' => $decoded->tid ?? '',
                'auth_store_id'  => $decoded->sid ?? '',
                'auth_role'      => $decoded->role ?? '',
                'auth_email'     => $decoded->email ?? '',
                'auth_token'     => $token,
            ]);

            // Propagate as headers so downstream gRPC clients can read them
            $request->headers->set('X-Tenant-Id', $decoded->tid ?? '');
            $request->headers->set('X-Store-Id', $decoded->sid ?? '');

        } catch (ExpiredException) {
            return response()->json(['message' => 'Token expired.'], 401);
        } catch (SignatureInvalidException) {
            return response()->json(['message' => 'Invalid token signature.'], 401);
        } catch (\Exception $e) {
            return response()->json(['message' => 'Unauthenticated.'], 401);
        }

        return $next($request);
    }

    private function extractToken(Request $request): ?string
    {
        $bearer = $request->bearerToken();
        if ($bearer !== null) {
            return $bearer;
        }

        return $request->cookie('pos_token');
    }
}
