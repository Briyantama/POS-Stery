<?php

namespace App\Services;

class ProductServiceClient extends BaseGrpcGatewayClient
{
    public function __construct()
    {
        parent::__construct('product_service_url');
    }

    public function search(string $tenantId, string $query = '', string $category = '', int $limit = 20, int $offset = 0): array
    {
        return $this->get('/v1/products', compact('tenantId', 'query', 'category', 'limit', 'offset'));
    }

    public function create(array $data): array
    {
        return $this->post('/v1/products', $data);
    }

    public function update(string $productId, array $data): array
    {
        return $this->put("/v1/products/{$productId}", $data);
    }
}
