<?php

return [

    /*
    |--------------------------------------------------------------------------
    | gRPC service base URLs (HTTP/JSON transcoding via grpc-gateway)
    |--------------------------------------------------------------------------
    */

    'auth_service_url'      => env('AUTH_SERVICE_URL',      'http://localhost:8081'),
    'product_service_url'   => env('PRODUCT_SERVICE_URL',   'http://localhost:8082'),
    'inventory_service_url' => env('INVENTORY_SERVICE_URL', 'http://localhost:8083'),
    'sales_service_url'     => env('SALES_SERVICE_URL',     'http://localhost:8084'),
    'supplier_service_url'  => env('SUPPLIER_SERVICE_URL',  'http://localhost:8085'),
    'customer_service_url'  => env('CUSTOMER_SERVICE_URL',  'http://localhost:8086'),

    'service_token_secret' => env('SERVICE_TOKEN_SECRET', 'changeme'),

    'jwt_public_key_path' => env('JWT_RS256_PUBLIC_KEY_PATH', storage_path('keys/public.pem')),

];
