<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\CustomerServiceClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class CustomerController extends Controller
{
    public function __construct(private CustomerServiceClient $customers) {}

    public function index(Request $request): JsonResponse
    {
        $data = $this->customers->list(
            tenantId: $request->auth_tenant_id,
            query:    $request->input('q', ''),
            limit:    (int) $request->input('limit', 20),
            offset:   (int) $request->input('offset', 0),
        );
        return response()->json($data);
    }

    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'name'  => ['required', 'string', 'max:255'],
            'phone' => ['nullable', 'string'],
            'email' => ['nullable', 'email'],
        ]);

        $validated['tenant_id'] = $request->auth_tenant_id;
        $data = $this->customers->create($validated);
        return response()->json($data, 201);
    }
}
