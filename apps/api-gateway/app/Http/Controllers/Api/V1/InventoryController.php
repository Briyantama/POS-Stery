<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\InventoryServiceClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class InventoryController extends Controller
{
    public function __construct(private InventoryServiceClient $inventory) {}

    public function index(Request $request): JsonResponse
    {
        $data = $this->inventory->list(
            tenantId:     $request->auth_tenant_id,
            storeId:      $request->resolved_store_id ?? $request->auth_store_id,
            lowStockOnly: (bool) $request->input('low_stock_only', false),
            limit:        (int) $request->input('limit', 20),
            offset:       (int) $request->input('offset', 0),
        );
        return response()->json($data);
    }

    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'product_id'   => ['required', 'uuid'],
            'quantity'     => ['required', 'integer', 'min:0'],
            'min_quantity' => ['nullable', 'integer', 'min:0'],
        ]);

        $validated['tenant_id'] = $request->auth_tenant_id;
        $validated['store_id']  = $request->resolved_store_id;
        $data = $this->inventory->addItem($validated);
        return response()->json($data, 201);
    }
}
