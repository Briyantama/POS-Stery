<?php

namespace App\Services;

use RuntimeException;

class GrpcClientException extends RuntimeException
{
    public function __construct(
        string $message,
        int $code,
        public readonly array $details = []
    ) {
        parent::__construct($message, $code);
    }
}
