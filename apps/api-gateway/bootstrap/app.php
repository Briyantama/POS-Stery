<?php

use App\Http\Middleware\CheckRole;
use App\Http\Middleware\JwtValidation;
use App\Http\Middleware\StoreResolver;
use App\Http\Middleware\TenantResolver;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        api: __DIR__.'/../routes/api.php',
        apiPrefix: 'api',
    )
    ->withMiddleware(function (Middleware $middleware) {
        $middleware->api(append: [
            TenantResolver::class,
            StoreResolver::class,
        ]);

        $middleware->alias([
            'jwt'  => JwtValidation::class,
            'role' => CheckRole::class,
        ]);
    })
    ->withExceptions(function (Exceptions $exceptions) {
        $exceptions->render(function (\App\Services\GrpcClientException $e) {
            return response()->json([
                'message' => $e->getMessage(),
                'details' => $e->details,
            ], $e->getCode() >= 400 && $e->getCode() < 600 ? $e->getCode() : 502);
        });

        $exceptions->render(function (\Illuminate\Validation\ValidationException $e) {
            return response()->json([
                'message' => 'Validation failed.',
                'errors'  => $e->errors(),
            ], 422);
        });

        $exceptions->render(function (\Throwable $e) {
            if (app()->environment('production')) {
                return response()->json(['message' => 'Internal Server Error.'], 500);
            }
        });
    })->create();
