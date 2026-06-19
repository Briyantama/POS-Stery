<?php

namespace App\Services;

class InventoryServiceClient extends BaseGrpcGatewayClient
{
    public function __construct()
    {
        parent::__construct('inventory_service_url');
    }

    public function list(string $tenantId, string $storeId, bool $lowStockOnly = false, int $limit = 20, int $offset = 0): array
    {
        return $this->get('/v1/inventory', compact('tenantId', 'storeId', 'lowStockOnly', 'limit', 'offset'));
    }

    public function addItem(array $data): array
    {
        return $this->post('/v1/inventory', $data);
    }

    public function setThreshold(string $productId, array $data): array
    {
        return $this->put("/v1/inventory/{$productId}/threshold", $data);
    }
}
