import { Request, Response } from 'express';

interface CreateOrderRequest {
    productId: string;
    quantity: number;
    notes?: string;
}

interface OrderItem {
    productId: string;
    productName: string;
    quantity: number;
    price: number;
}

interface OrderResponse {
    id: string;
    items: OrderItem[];
    total: number;
    status: string;
}

type OrderStatus = 'pending' | 'confirmed' | 'shipped' | 'delivered';

interface OrderDetailResponse extends OrderResponse {
    status: OrderStatus;
    createdAt: string;
}

interface PaginatedResponse<T> {
    items: T[];
    total: number;
    page: number;
    pageSize: number;
}

/**
 * createOrder creates a new order.
 */
app.post('/orders', (req: Request<{}, {}, CreateOrderRequest>, res: Response<OrderResponse>) => {
    const { productId, quantity } = req.body;
    res.json({ id: '1', items: [], total: 0, status: 'pending' });
});

/**
 * getOrder returns an order by ID.
 */
app.get('/orders/:id', (req: Request<{ id: string }>, res: Response<OrderDetailResponse>) => {
    res.json({ id: '1', items: [], total: 0, status: 'pending', createdAt: '2024-01-01' });
});

/**
 * listOrders returns all orders.
 */
app.get('/orders', (req: Request, res: Response<PaginatedResponse<OrderResponse>>) => {
    res.json({ items: [], total: 0, page: 1, pageSize: 10 });
});
