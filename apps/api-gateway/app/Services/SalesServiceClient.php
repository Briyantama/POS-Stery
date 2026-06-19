<?php

namespace App\Services;

class SalesServiceClient extends BaseGrpcGatewayClient
{
    public function __construct()
    {
        parent::__construct('sales_service_url');
    }

    public function createSale(array $data): array
    {
        return $this->post('/v1/sales', $data);
    }

    public function report(string $tenantId, string $storeId, string $date): array
    {
        return $this->get('/v1/sales/report', compact('tenantId', 'storeId', 'date'));
    }
}
