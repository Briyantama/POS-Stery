<?php

namespace App\Services;

class CustomerServiceClient extends BaseGrpcGatewayClient
{
    public function __construct()
    {
        parent::__construct('customer_service_url');
    }

    public function list(string $tenantId, string $query = '', int $limit = 20, int $offset = 0): array
    {
        return $this->get('/v1/customers', compact('tenantId', 'query', 'limit', 'offset'));
    }

    public function create(array $data): array
    {
        return $this->post('/v1/customers', $data);
    }

    public function get(string $customerId): array
    {
        return $this->get("/v1/customers/{$customerId}");
    }
}
