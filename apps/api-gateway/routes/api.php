<?php

use App\Http\Controllers\Api\V1\AuthController;
use App\Http\Controllers\Api\V1\CustomerController;
use App\Http\Controllers\Api\V1\InventoryController;
use App\Http\Controllers\Api\V1\ProductController;
use App\Http\Controllers\Api\V1\SalesController;
use App\Http\Controllers\Api\V1\SupplierController;
use Illuminate\Support\Facades\Route;

// Auth endpoints — public
Route::post('/login', [AuthController::class, 'login']);

// All other endpoints require a valid JWT
Route::middleware(['jwt'])->group(function () {
    Route::post('/logout', [AuthController::class, 'logout']);

    // Product catalog — cashiers read, admins write
    Route::get('/products', [ProductController::class, 'index'])
        ->middleware('role:admin,cashier');
    Route::post('/products', [ProductController::class, 'store'])
        ->middleware('role:admin');
    Route::put('/products/{id}', [ProductController::class, 'update'])
        ->middleware('role:admin');

    // Inventory — stock-related roles
    Route::get('/inventory', [InventoryController::class, 'index'])
        ->middleware('role:admin,stock_manager');
    Route::post('/inventory', [InventoryController::class, 'store'])
        ->middleware('role:admin,stock_manager');

    // Sales — cashiers create, admins view reports
    Route::post('/sales', [SalesController::class, 'store'])
        ->middleware('role:cashier');
    Route::get('/sales/report', [SalesController::class, 'report'])
        ->middleware('role:admin');

    // Suppliers and purchase orders
    Route::get('/suppliers', [SupplierController::class, 'index'])
        ->middleware('role:admin,stock_manager');
    Route::post('/suppliers', [SupplierController::class, 'store'])
        ->middleware('role:admin');
    Route::post('/purchase-orders', [SupplierController::class, 'order'])
        ->middleware('role:admin,stock_manager');

    // Customers
    Route::get('/customers', [CustomerController::class, 'index'])
        ->middleware('role:admin,cashier');
    Route::post('/customers', [CustomerController::class, 'store'])
        ->middleware('role:admin,cashier');
});
