FROM php:8.2-fpm-alpine

RUN apk add --no-cache \
    git \
    unzip \
    zip \
    libzip-dev \
    oniguruma-dev \
    postgresql-dev \
    libxml2-dev \
    && docker-php-ext-install pdo pdo_pgsql mbstring xml bcmath

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /var/www/html

# Copy only Laravel app files (not entire monorepo)
COPY apps/api-gateway/composer.json apps/api-gateway/composer.lock* ./

RUN composer install --no-dev --optimize-autoloader --no-interaction --prefer-dist

# Copy application code
COPY apps/api-gateway/ ./

EXPOSE 8000

CMD ["php", "artisan", "serve", "--host=0.0.0.0", "--port=8000"]
