<?php

namespace App\Services;

class SupplierServiceClient extends BaseGrpcGatewayClient
{
    public function __construct()
    {
        parent::__construct('supplier_service_url');
    }

    public function list(string $tenantId, int $limit = 20, int $offset = 0): array
    {
        return $this->get('/v1/suppliers', compact('tenantId', 'limit', 'offset'));
    }

    public function create(array $data): array
    {
        return $this->post('/v1/suppliers', $data);
    }

    public function createPurchaseOrder(array $data): array
    {
        return $this->post('/v1/purchase-orders', $data);
    }

    public function receiveStock(string $purchaseOrderId, array $data): array
    {
        return $this->post("/v1/purchase-orders/{$purchaseOrderId}/receive", $data);
    }
}
