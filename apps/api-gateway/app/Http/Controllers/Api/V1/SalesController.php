<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\SalesServiceClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SalesController extends Controller
{
    public function __construct(private SalesServiceClient $sales) {}

    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'items'               => ['required', 'array', 'min:1'],
            'items.*.product_id'  => ['required', 'uuid'],
            'items.*.quantity'    => ['required', 'integer', 'min:1'],
            'items.*.unit_price'  => ['required', 'numeric', 'min:0'],
            'customer_id'         => ['nullable', 'uuid'],
            'discount_amount'     => ['nullable', 'numeric', 'min:0'],
            'notes'               => ['nullable', 'string'],
        ]);

        $validated['tenant_id']  = $request->auth_tenant_id;
        $validated['store_id']   = $request->resolved_store_id;
        $validated['cashier_id'] = $request->auth_user_id;

        $data = $this->sales->createSale($validated);
        return response()->json($data, 201);
    }

    public function report(Request $request): JsonResponse
    {
        $request->validate([
            'date'     => ['required', 'date_format:Y-m-d'],
            'store_id' => ['nullable', 'uuid'],
        ]);

        $data = $this->sales->report(
            tenantId: $request->auth_tenant_id,
            storeId:  $request->input('store_id', $request->resolved_store_id ?? ''),
            date:     $request->input('date'),
        );
        return response()->json($data);
    }
}
