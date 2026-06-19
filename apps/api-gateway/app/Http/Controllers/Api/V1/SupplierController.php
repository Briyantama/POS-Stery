<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\SupplierServiceClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SupplierController extends Controller
{
    public function __construct(private SupplierServiceClient $suppliers) {}

    public function index(Request $request): JsonResponse
    {
        $data = $this->suppliers->list(
            tenantId: $request->auth_tenant_id,
            limit:    (int) $request->input('limit', 20),
            offset:   (int) $request->input('offset', 0),
        );
        return response()->json($data);
    }

    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'name'    => ['required', 'string', 'max:255'],
            'contact' => ['nullable', 'string'],
            'phone'   => ['nullable', 'string'],
            'email'   => ['nullable', 'email'],
            'address' => ['nullable', 'string'],
        ]);

        $validated['tenant_id'] = $request->auth_tenant_id;
        $data = $this->suppliers->create($validated);
        return response()->json($data, 201);
    }

    public function order(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'supplier_id'             => ['required', 'uuid'],
            'items'                   => ['required', 'array', 'min:1'],
            'items.*.product_id'      => ['required', 'uuid'],
            'items.*.quantity_ordered'=> ['required', 'integer', 'min:1'],
            'items.*.unit_cost'       => ['required', 'numeric', 'min:0'],
            'notes'                   => ['nullable', 'string'],
        ]);

        $validated['tenant_id'] = $request->auth_tenant_id;
        $validated['store_id']  = $request->resolved_store_id;
        $data = $this->suppliers->createPurchaseOrder($validated);
        return response()->json($data, 201);
    }
}
