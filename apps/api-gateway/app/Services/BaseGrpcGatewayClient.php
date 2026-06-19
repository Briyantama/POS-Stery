<?php

namespace App\Services;

use GuzzleHttp\Client;
use GuzzleHttp\Exception\ClientException;
use GuzzleHttp\Exception\ServerException;
use Illuminate\Support\Facades\Request;
use RuntimeException;

abstract class BaseGrpcGatewayClient
{
    protected Client $http;
    protected string $baseUrl;

    public function __construct(string $configKey)
    {
        $this->baseUrl = rtrim(config("services.{$configKey}"), '/');
        $this->http = new Client([
            'timeout' => 10,
            'connect_timeout' => 3,
        ]);
    }

    protected function post(string $path, array $body): array
    {
        return $this->request('POST', $path, ['json' => $body]);
    }

    protected function put(string $path, array $body): array
    {
        return $this->request('PUT', $path, ['json' => $body]);
    }

    protected function patch(string $path, array $body): array
    {
        return $this->request('PATCH', $path, ['json' => $body]);
    }

    protected function get(string $path, array $query = []): array
    {
        return $this->request('GET', $path, ['query' => array_filter($query)]);
    }

    private function request(string $method, string $path, array $options): array
    {
        $options['headers'] = array_merge(
            $options['headers'] ?? [],
            [
                'X-Service-Token' => $this->serviceToken(),
                'X-Tenant-Id'     => Request::header('X-Tenant-Id', ''),
                'X-Store-Id'      => Request::header('X-Store-Id', ''),
                'Accept'          => 'application/json',
            ]
        );

        try {
            $response = $this->http->request($method, $this->baseUrl . $path, $options);
            return json_decode($response->getBody()->getContents(), true) ?? [];
        } catch (ClientException $e) {
            $body = json_decode($e->getResponse()->getBody()->getContents(), true) ?? [];
            throw new GrpcClientException(
                $body['message'] ?? 'Service error',
                $e->getCode(),
                $body
            );
        } catch (ServerException $e) {
            throw new RuntimeException('Upstream service error: ' . $e->getMessage(), 502);
        }
    }

    private function serviceToken(): string
    {
        $secret = config('services.service_token_secret');
        if ($secret === 'changeme') {
            return '';
        }
        return hash_hmac('sha256', (string) time(), $secret);
    }
}
