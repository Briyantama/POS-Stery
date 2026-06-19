<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;

class TenantResolver
{
    public function handle(Request $request, Closure $next): mixed
    {
        $tenantId = $request->auth_tenant_id ?? '';

        if (empty($tenantId)) {
            return response()->json(['message' => 'Tenant context missing.'], 403);
        }

        // Validate UUID format
        if (!$this->isValidUuid($tenantId)) {
            return response()->json(['message' => 'Invalid tenant context.'], 403);
        }

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
