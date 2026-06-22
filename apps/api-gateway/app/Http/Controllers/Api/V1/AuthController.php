<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\AuthServiceClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AuthController extends Controller
{
    public function __construct(private AuthServiceClient $auth) {}

    public function login(Request $request): JsonResponse
    {
        $request->validate([
            'email'    => ['required', 'email'],
            'password' => ['required', 'string'],
            'store_id' => ['nullable', 'uuid'],
        ]);

        $result = $this->auth->login(
            $request->input('email'),
            $request->input('password'),
            $request->input('store_id', '')
        );

        return response()->json($result);
    }

    public function logout(Request $request): JsonResponse
    {
        $this->auth->logout(
            $request->auth_token ?? '',
            $request->input('refresh_token', '')
        );
        return response()->json(['success' => true]);
    }
}
