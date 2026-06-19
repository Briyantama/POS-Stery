<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Services\ProductServiceClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class ProductController extends Controller
{
    public function __construct(private ProductServiceClient $products) {}

    public function index(Request $request): JsonResponse
    {
        $data = $this->products->search(
            tenantId: $request->auth_tenant_id,
            query:    $request->input('q', ''),
            category: $request->input('category', ''),
            limit:    (int) $request->input('limit', 20),
            offset:   (int) $request->input('offset', 0),
        );
        return response()->json($data);
    }

    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'name'        => ['required', 'string', 'max:255'],
            'sku'         => ['required', 'string', 'max:100'],
            'barcode'     => ['nullable', 'string'],
            'category_id' => ['nullable', 'uuid'],
            'base_price'  => ['required', 'numeric', 'min:0'],
            'description' => ['nullable', 'string'],
            'unit'        => ['nullable', 'string'],
        ]);

        $validated['tenant_id'] = $request->auth_tenant_id;
        $data = $this->products->create($validated);
        return response()->json($data, 201);
    }

    public function update(Request $request, string $id): JsonResponse
    {
        $validated = $request->validate([
            'name'        => ['nullable', 'string', 'max:255'],
            'base_price'  => ['nullable', 'numeric', 'min:0'],
            'sale_price'  => ['nullable', 'numeric', 'min:0'],
            'is_active'   => ['nullable', 'boolean'],
        ]);

        $validated['tenant_id'] = $request->auth_tenant_id;
        $data = $this->products->update($id, $validated);
        return response()->json($data);
    }
}
