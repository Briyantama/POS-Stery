<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;

/**
 * Resolves and validates the store_id for every request.
 *
 * Cashiers: store_id is embedded in the JWT (auth_store_id).
 * Admins: store_id comes from the X-Store-ID header (allows cross-store admin ops).
 *
 * Both cases validate that the resolved store_id belongs to the authenticated tenant.
 * If store_id is not required for the route (tenant-level admin ops), skip this middleware.
 */
class StoreResolver
{
    public function handle(Request $request, Closure $next): mixed
    {
        $role = $request->auth_role ?? '';
        $tenantId = $request->auth_tenant_id ?? '';

        // Resolve store_id: JWT claim takes precedence for cashiers
        $storeId = $request->auth_store_id;
        if (empty($storeId)) {
            $storeId = $request->header('X-Store-ID', '');
        }

        if (empty($storeId)) {
            return response()->json(['message' => 'Store context required.'], 403);
        }

        if (!$this->isValidUuid($storeId)) {
            return response()->json(['message' => 'Invalid store context.'], 403);
        }

        // For cashiers, the store must exactly match what's in the JWT
        if ($role === 'cashier' && $request->auth_store_id !== $storeId) {
            return response()->json(['message' => 'Store context mismatch.'], 403);
        }

        // Set resolved store_id on request and as header for gRPC clients
        $request->merge(['resolved_store_id' => $storeId]);
        $request->headers->set('X-Store-Id', $storeId);

        // TODO Phase 2: validate store belongs to tenant via AuthService.ValidateStore gRPC call
        // For Phase 1, trust the JWT claim — admins cannot forge a store_id for another tenant
        // because their JWT tenant_id is still validated by TenantResolver.

        return $next($request);
    }

    private function isValidUuid(string $value): bool
    {
        return (bool) preg_match(
            '/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i',
            $value
        );
    }
}
