import { Request, Response } from 'express';
const app = require('express')();

app.use(require('express').json());

// ===== Models =====

interface BaseModel {
    id: number;
    createdAt: string;
    updatedAt: string;
}

interface User extends BaseModel {
    username: string;
    email: string;
    password?: string;
    phone: string;
    avatar: string;
    status: number;
}

interface Product extends BaseModel {
    name: string;
    description: string;
    price: number;
    originalPrice: number;
    stock: number;
    categoryId: number;
    mainImage: string;
    status: number;
    salesCount: number;
}

interface Order extends BaseModel {
    orderNo: string;
    userId: number;
    totalAmount: number;
    discountAmount: number;
    finalAmount: number;
    status: number;
    paymentMethod: string;
    shippingAddress: string;
    receiverName: string;
    receiverPhone: string;
}

interface Category extends BaseModel {
    name: string;
    parentId: number;
    sort: number;
    icon: string;
    status: number;
}

interface Payment extends BaseModel {
    paymentNo: string;
    orderId: number;
    amount: number;
    method: string;
    status: number;
    paidAt: string;
}

interface Shipping extends BaseModel {
    shippingNo: string;
    orderId: number;
    carrier: string;
    trackingNo: string;
    status: number;
    shippedAt: string;
    deliveredAt: string;
}

interface Inventory extends BaseModel {
    productId: number;
    warehouseId: number;
    quantity: number;
    locked: number;
    available: number;
}

interface Review extends BaseModel {
    productId: number;
    userId: number;
    rating: number;
    content: string;
    images: string;
    status: number;
}

interface Notification extends BaseModel {
    userId: number;
    type: string;
    title: string;
    content: string;
    read: boolean;
}

// ===== User DTOs =====

interface CreateUserRequest {
    name: string;
    email: string;
    age?: number;
    password?: string;
    phone?: string;
}

interface UpdateUserRequest {
    name?: string;
    email?: string;
    phone?: string;
}

interface UserResponse {
    id: number;
    name: string;
    email: string;
    phone: string;
    avatar: string;
    status: number;
    tags: string[];
}

interface ListUsersResponse {
    users: UserResponse[];
    total: number;
}

interface LoginRequest {
    email: string;
    password: string;
}

interface LoginResponse {
    token: string;
    userId: number;
    username: string;
}

interface RegisterRequest {
    username: string;
    email: string;
    password: string;
    phone: string;
}

interface UserProfileResponse {
    id: number;
    username: string;
    email: string;
    phone: string;
    avatar: string;
    nickname: string;
    bio: string;
    gender: number;
    birthday: string;
}

interface AddressResponse {
    id: number;
    receiver: string;
    phone: string;
    province: string;
    city: string;
    district: string;
    detail: string;
    isDefault: boolean;
}

// ===== Product DTOs =====

interface CreateProductRequest {
    name: string;
    description: string;
    price: number;
    originalPrice: number;
    categoryId: number;
    stock: number;
    mainImage: string;
    unit: string;
}

interface ProductSpecVO {
    id: number;
    name: string;
    value: string;
    price: number;
    stock: number;
}

interface ProductDetailResponse {
    id: number;
    name: string;
    description: string;
    price: number;
    originalPrice: number;
    categoryId: number;
    categoryName: string;
    stock: number;
    sales: number;
    mainImage: string;
    images: string[];
    specs: ProductSpecVO[];
    averageRating: number;
    reviewCount: number;
}

interface ProductStatisticsResponse {
    productId: number;
    views: number;
    sales: number;
    favorites: number;
    averageRating: number;
    reviewCount: number;
}

// ===== Order DTOs =====

interface OrderItemRequest {
    productId: number;
    quantity: number;
    price: number;
}

interface CreateOrderRequest {
    userId: number;
    items: OrderItemRequest[];
    shippingAddress: string;
    receiverName: string;
    receiverPhone: string;
    remark: string;
}

interface OrderItemResponse {
    productId: number;
    productName: string;
    quantity: number;
    price: number;
}

interface OrderResponse {
    id: number;
    orderNo: string;
    userId: number;
    totalAmount: number;
    finalAmount: number;
    status: number;
    items: OrderItemResponse[];
}

interface TrackPoint {
    time: string;
    description: string;
}

interface OrderTrackResponse {
    orderNo: string;
    status: number;
    trackPoints: TrackPoint[];
}

interface OrderStatisticsResponse {
    totalOrders: number;
    totalAmount: number;
    pendingOrders: number;
    completedOrders: number;
    cancelledOrders: number;
}

// ===== Category DTOs =====

interface CreateCategoryRequest {
    name: string;
    parentId: number;
    sort: number;
    icon: string;
}

interface CategoryTreeResponse {
    id: number;
    name: string;
    parentId: number;
    sort: number;
    icon: string;
    children: CategoryTreeResponse[];
}

// ===== Payment DTOs =====

interface CreatePaymentRequest {
    orderId: number;
    amount: number;
    method: string;
}

interface PaymentRefundRequest {
    reason: string;
    amount: number;
}

interface PaymentRefundResponse {
    id: number;
    refundNo: string;
    amount: number;
    status: number;
    createdAt: string;
}

interface PaymentChannelResponse {
    code: string;
    name: string;
    icon: string;
    enabled: boolean;
}

// ===== Shipping DTOs =====

interface CreateShippingRequest {
    orderId: number;
    carrier: string;
    trackingNo: string;
}

interface ShippingTrackResponse {
    shippingNo: string;
    carrier: string;
    trackingNo: string;
    status: number;
    trackPoints: TrackPoint[];
}

interface CarrierResponse {
    code: string;
    name: string;
    icon: string;
    enabled: boolean;
}

// ===== Inventory DTOs =====

interface CreateInventoryRequest {
    productId: number;
    warehouseId: number;
    quantity: number;
}

interface InventoryMovementResponse {
    id: number;
    productId: number;
    quantity: number;
    type: string;
    reason: string;
    createdAt: string;
}

interface WarehouseResponse {
    id: number;
    name: string;
    address: string;
    contact: string;
    phone: string;
}

// ===== Review DTOs =====

interface CreateReviewRequest {
    productId: number;
    rating: number;
    content: string;
    images: string;
}

interface CommentResponse {
    id: number;
    reviewId: number;
    userId: number;
    username: string;
    content: string;
    createdAt: string;
}

// ===== Notification DTOs =====

interface CreateNotificationRequest {
    userId: number;
    type: string;
    title: string;
    content: string;
}

interface NotificationTemplateResponse {
    id: number;
    code: string;
    name: string;
    type: string;
    titleTemplate: string;
    contentTemplate: string;
}

// ===== Analytics DTOs =====

interface AnalyticsOverviewResponse {
    totalRevenue: number;
    totalOrders: number;
    totalUsers: number;
    totalProducts: number;
    todayRevenue: number;
    todayOrders: number;
    newUsers: number;
    conversionRate: number;
}

interface DailySales {
    date: string;
    sales: number;
    orders: number;
}

interface SalesStatisticsResponse {
    totalSales: number;
    totalRefund: number;
    totalOrders: number;
    averageOrderValue: number;
    dailySales: DailySales[];
}

interface Result<T> {
    code: number;
    message: string;
    data: T;
}

interface PageResult<T> {
    list: T[];
    total: number;
    pageNum: number;
    pageSize: number;
}

// ===== User Controllers =====

/**
 * listUsers returns all users.
 */
app.get('/users', (req: Request, res: Response<ListUsersResponse>) => {
    const { name, role = 'user' } = req.query;
    res.json({ users: [], total: 0 });
});

/**
 * createUser creates a new user.
 */
app.post('/users', (req: Request<{}, {}, CreateUserRequest>, res: Response<UserResponse>) => {
    const { name, email } = req.body;
    res.status(201).json({ id: 1, name, email, phone: '', avatar: '', status: 1, tags: [] });
});

/**
 * getUser returns a single user by ID.
 */
app.get('/users/:id', (req: Request<{ id: string }>, res: Response<UserResponse>) => {
    res.json({ id: 1, name: 'test', email: 'test@example.com', phone: '', avatar: '', status: 1, tags: [] });
});

/**
 * updateUser updates an existing user.
 */
app.put('/users/:id', (req: Request<{ id: string }, {}, UpdateUserRequest>, res: Response<UserResponse>) => {
    const { name, email } = req.body;
    res.json({ id: 1, name: name || 'unknown', email: email || '', phone: '', avatar: '', status: 1, tags: [] });
});

/**
 * deleteUser removes a user by ID.
 */
app.delete('/users/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * patchUser partially updates a user.
 */
app.patch('/users/:id', (req: Request<{ id: string }>, res: Response<UserResponse>) => {
    res.json({ id: 1, name: 'test', email: 'test@example.com', phone: '', avatar: '', status: 1, tags: [] });
});

/**
 * userLogin handles user login.
 */
app.post('/users/login', (req: Request<{}, {}, LoginRequest>, res: Response<LoginResponse>) => {
    res.json({ token: 'xxx', userId: 1, username: 'test' });
});

/**
 * userRegister handles user registration.
 */
app.post('/users/register', (req: Request<{}, {}, RegisterRequest>, res: Response<UserResponse>) => {
    res.status(201).json({ id: 1, name: 'test', email: 'test@example.com', phone: '', avatar: '', status: 1, tags: [] });
});

/**
 * getUserProfile returns a user's profile.
 */
app.get('/users/:id/profile', (req: Request<{ id: string }>, res: Response<UserProfileResponse>) => {
    res.json({ id: 1, username: 'test', email: 'test@example.com', phone: '', avatar: '', nickname: '', bio: '', gender: 0, birthday: '' });
});

/**
 * updateUserProfile updates a user's profile.
 */
app.put('/users/:id/profile', (req: Request<{ id: string }>, res: Response<UserProfileResponse>) => {
    res.json({ id: 1, username: 'test', email: 'test@example.com', phone: '', avatar: '', nickname: '', bio: '', gender: 0, birthday: '' });
});

/**
 * getUserAddresses returns a user's addresses.
 */
app.get('/users/:id/addresses', (req: Request<{ id: string }>, res: Response<AddressResponse[]>) => {
    res.json([]);
});

/**
 * addUserAddress adds a new address for a user.
 */
app.post('/users/:id/addresses', (req: Request<{ id: string }>, res: Response<AddressResponse>) => {
    res.status(201).json({ id: 1, receiver: '', phone: '', province: '', city: '', district: '', detail: '', isDefault: false });
});

/**
 * getUserFavorites returns a user's favorite products.
 */
app.get('/users/:id/favorites', (req: Request<{ id: string }>, res: Response<PageResult<Product>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

// ===== Product Controllers =====

/**
 * listProducts returns a paginated list of products.
 */
app.get('/api/products', (req: Request, res: Response<PageResult<Product>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * createProduct creates a new product.
 */
app.post('/api/products', (req: Request<{}, {}, CreateProductRequest>, res: Response<Product>) => {
    res.status(201).json({ id: 1, createdAt: '', updatedAt: '', name: '', description: '', price: 0, originalPrice: 0, stock: 0, categoryId: 0, mainImage: '', status: 1, salesCount: 0 });
});

/**
 * getProduct returns a single product by ID.
 */
app.get('/api/products/:id', (req: Request<{ id: string }>, res: Response<Product>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', name: '', description: '', price: 0, originalPrice: 0, stock: 0, categoryId: 0, mainImage: '', status: 1, salesCount: 0 });
});

/**
 * updateProduct updates an existing product.
 */
app.put('/api/products/:id', (req: Request<{ id: string }, {}, CreateProductRequest>, res: Response<Product>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', name: '', description: '', price: 0, originalPrice: 0, stock: 0, categoryId: 0, mainImage: '', status: 1, salesCount: 0 });
});

/**
 * deleteProduct removes a product by ID.
 */
app.delete('/api/products/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * searchProducts searches for products.
 */
app.get('/api/products/search', (req: Request, res: Response<PageResult<Product>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * getProductDetail returns detailed product information.
 */
app.get('/api/products/:id/detail', (req: Request<{ id: string }>, res: Response<ProductDetailResponse>) => {
    res.json({ id: 1, name: '', description: '', price: 0, originalPrice: 0, categoryId: 0, categoryName: '', stock: 0, sales: 0, mainImage: '', images: [], specs: [], averageRating: 0, reviewCount: 0 });
});

/**
 * getProductsByCategory returns products in a category.
 */
app.get('/api/products/category/:categoryId', (req: Request<{ categoryId: string }>, res: Response<PageResult<Product>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * getHotProducts returns hot products.
 */
app.get('/api/products/hot', (req: Request, res: Response<Product[]>) => {
    res.json([]);
});

/**
 * getNewProducts returns new products.
 */
app.get('/api/products/new', (req: Request, res: Response<Product[]>) => {
    res.json([]);
});

/**
 * addProductSpec adds a product specification.
 */
app.post('/api/products/:id/specs', (req: Request<{ id: string }>, res: Response<ProductSpecVO>) => {
    res.status(201).json({ id: 1, name: '', value: '', price: 0, stock: 0 });
});

/**
 * getProductSpecs returns product specifications.
 */
app.get('/api/products/:id/specs', (req: Request<{ id: string }>, res: Response<ProductSpecVO[]>) => {
    res.json([]);
});

/**
 * getProductStatistics returns product statistics.
 */
app.get('/api/products/:id/statistics', (req: Request<{ id: string }>, res: Response<ProductStatisticsResponse>) => {
    res.json({ productId: 1, views: 0, sales: 0, favorites: 0, averageRating: 0, reviewCount: 0 });
});

// ===== Order Controllers =====

/**
 * listOrders returns a paginated list of orders.
 */
app.get('/api/orders', (req: Request, res: Response<PageResult<OrderResponse>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * createOrder creates a new order.
 */
app.post('/api/orders', (req: Request<{}, {}, CreateOrderRequest>, res: Response<OrderResponse>) => {
    res.status(201).json({ id: 1, orderNo: '', userId: 1, totalAmount: 0, finalAmount: 0, status: 0, items: [] });
});

/**
 * getOrder returns a single order by ID.
 */
app.get('/api/orders/:id', (req: Request<{ id: string }>, res: Response<OrderResponse>) => {
    res.json({ id: 1, orderNo: '', userId: 1, totalAmount: 0, finalAmount: 0, status: 0, items: [] });
});

/**
 * updateOrder updates an existing order.
 */
app.put('/api/orders/:id', (req: Request<{ id: string }>, res: Response<OrderResponse>) => {
    res.json({ id: 1, orderNo: '', userId: 1, totalAmount: 0, finalAmount: 0, status: 0, items: [] });
});

/**
 * deleteOrder removes an order by ID.
 */
app.delete('/api/orders/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * cancelOrder cancels an order.
 */
app.post('/api/orders/:id/cancel', (req: Request<{ id: string }>, res: Response<Order>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', orderNo: '', userId: 1, totalAmount: 0, discountAmount: 0, finalAmount: 0, status: 0, paymentMethod: '', shippingAddress: '', receiverName: '', receiverPhone: '' });
});

/**
 * trackOrder returns order tracking information.
 */
app.get('/api/orders/:id/track', (req: Request<{ id: string }>, res: Response<OrderTrackResponse>) => {
    res.json({ orderNo: '', status: 0, trackPoints: [] });
});

/**
 * addOrderItem adds an item to an order.
 */
app.post('/api/orders/:id/items', (req: Request<{ id: string }, {}, OrderItemRequest>, res: Response<Order>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', orderNo: '', userId: 1, totalAmount: 0, discountAmount: 0, finalAmount: 0, status: 0, paymentMethod: '', shippingAddress: '', receiverName: '', receiverPhone: '' });
});

/**
 * refundOrder requests a refund for an order.
 */
app.post('/api/orders/:id/refund', (req: Request<{ id: string }>, res: Response<PaymentRefundResponse>) => {
    res.json({ id: 1, refundNo: '', amount: 0, status: 0, createdAt: '' });
});

/**
 * getOrderStatistics returns order statistics.
 */
app.get('/api/orders/statistics', (req: Request, res: Response<OrderStatisticsResponse>) => {
    res.json({ totalOrders: 0, totalAmount: 0, pendingOrders: 0, completedOrders: 0, cancelledOrders: 0 });
});

/**
 * getOrdersByUser returns orders for a user.
 */
app.get('/api/orders/user/:userId', (req: Request<{ userId: string }>, res: Response<PageResult<Order>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

// ===== Category Controllers =====

/**
 * listCategories returns all categories.
 */
app.get('/api/categories', (req: Request, res: Response<Category[]>) => {
    res.json([]);
});

/**
 * createCategory creates a new category.
 */
app.post('/api/categories', (req: Request<{}, {}, CreateCategoryRequest>, res: Response<Category>) => {
    res.status(201).json({ id: 1, createdAt: '', updatedAt: '', name: '', parentId: 0, sort: 0, icon: '', status: 1 });
});

/**
 * getCategory returns a single category by ID.
 */
app.get('/api/categories/:id', (req: Request<{ id: string }>, res: Response<Category>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', name: '', parentId: 0, sort: 0, icon: '', status: 1 });
});

/**
 * updateCategory updates an existing category.
 */
app.put('/api/categories/:id', (req: Request<{ id: string }, {}, CreateCategoryRequest>, res: Response<Category>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', name: '', parentId: 0, sort: 0, icon: '', status: 1 });
});

/**
 * deleteCategory removes a category by ID.
 */
app.delete('/api/categories/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * getCategoryTree returns the category tree.
 */
app.get('/api/categories/tree', (req: Request, res: Response<CategoryTreeResponse[]>) => {
    res.json([]);
});

/**
 * getCategoryProducts returns products in a category.
 */
app.get('/api/categories/:id/products', (req: Request<{ id: string }>, res: Response<PageResult<Product>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * updateCategorySort updates category sort order.
 */
app.put('/api/categories/:id/sort', (req: Request<{ id: string }>, res: Response<Category>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', name: '', parentId: 0, sort: 0, icon: '', status: 1 });
});

/**
 * getRootCategories returns root categories.
 */
app.get('/api/categories/roots', (req: Request, res: Response<Category[]>) => {
    res.json([]);
});

// ===== Payment Controllers =====

/**
 * listPayments returns a paginated list of payments.
 */
app.get('/api/payments', (req: Request, res: Response<PageResult<Payment>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * createPayment creates a new payment.
 */
app.post('/api/payments', (req: Request<{}, {}, CreatePaymentRequest>, res: Response<Payment>) => {
    res.status(201).json({ id: 1, createdAt: '', updatedAt: '', paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' });
});

/**
 * getPayment returns a single payment by ID.
 */
app.get('/api/payments/:id', (req: Request<{ id: string }>, res: Response<Payment>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' });
});

/**
 * updatePayment updates an existing payment.
 */
app.put('/api/payments/:id', (req: Request<{ id: string }>, res: Response<Payment>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' });
});

/**
 * deletePayment removes a payment by ID.
 */
app.delete('/api/payments/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * refundPayment refunds a payment.
 */
app.post('/api/payments/:id/refund', (req: Request<{ id: string }, {}, PaymentRefundRequest>, res: Response<PaymentRefundResponse>) => {
    res.json({ id: 1, refundNo: '', amount: 0, status: 0, createdAt: '' });
});

/**
 * getPaymentRecords returns payment records.
 */
app.get('/api/payments/:id/records', (req: Request<{ id: string }>, res: Response<PaymentRefundResponse[]>) => {
    res.json([]);
});

/**
 * getPaymentChannels returns available payment channels.
 */
app.get('/api/payments/channels', (req: Request, res: Response<PaymentChannelResponse[]>) => {
    res.json([]);
});

/**
 * getPaymentStatistics returns payment statistics.
 */
app.get('/api/payments/statistics', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * confirmPayment confirms a payment.
 */
app.post('/api/payments/:id/confirm', (req: Request<{ id: string }>, res: Response<Payment>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' });
});

// ===== Shipping Controllers =====

/**
 * listShipping returns a paginated list of shipping records.
 */
app.get('/api/shipping', (req: Request, res: Response<PageResult<Shipping>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * createShipping creates a new shipping record.
 */
app.post('/api/shipping', (req: Request<{}, {}, CreateShippingRequest>, res: Response<Shipping>) => {
    res.status(201).json({ id: 1, createdAt: '', updatedAt: '', shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' });
});

/**
 * getShipping returns a single shipping record by ID.
 */
app.get('/api/shipping/:id', (req: Request<{ id: string }>, res: Response<Shipping>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' });
});

/**
 * updateShipping updates an existing shipping record.
 */
app.put('/api/shipping/:id', (req: Request<{ id: string }>, res: Response<Shipping>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' });
});

/**
 * deleteShipping removes a shipping record by ID.
 */
app.delete('/api/shipping/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * trackShipping returns shipping tracking information.
 */
app.get('/api/shipping/:id/track', (req: Request<{ id: string }>, res: Response<ShippingTrackResponse>) => {
    res.json({ shippingNo: '', carrier: '', trackingNo: '', status: 0, trackPoints: [] });
});

/**
 * getCarriers returns available carriers.
 */
app.get('/api/shipping/carriers', (req: Request, res: Response<CarrierResponse[]>) => {
    res.json([]);
});

/**
 * calculateShippingRates calculates shipping rates.
 */
app.post('/api/shipping/rates', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * shipOrder ships an order.
 */
app.post('/api/shipping/:id/ship', (req: Request<{ id: string }>, res: Response<Shipping>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' });
});

/**
 * deliverOrder delivers an order.
 */
app.post('/api/shipping/:id/deliver', (req: Request<{ id: string }>, res: Response<Shipping>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' });
});

// ===== Inventory Controllers =====

/**
 * listInventory returns a paginated list of inventory.
 */
app.get('/api/inventory', (req: Request, res: Response<PageResult<Inventory>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * createInventory creates a new inventory record.
 */
app.post('/api/inventory', (req: Request<{}, {}, CreateInventoryRequest>, res: Response<Inventory>) => {
    res.status(201).json({ id: 1, createdAt: '', updatedAt: '', productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 });
});

/**
 * getInventory returns a single inventory record by ID.
 */
app.get('/api/inventory/:id', (req: Request<{ id: string }>, res: Response<Inventory>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 });
});

/**
 * updateInventory updates an existing inventory record.
 */
app.put('/api/inventory/:id', (req: Request<{ id: string }>, res: Response<Inventory>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 });
});

/**
 * deleteInventory removes an inventory record by ID.
 */
app.delete('/api/inventory/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * updateStock updates inventory stock.
 */
app.put('/api/inventory/:id/stock', (req: Request<{ id: string }>, res: Response<Inventory>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 });
});

/**
 * getInventoryMovements returns inventory movement records.
 */
app.get('/api/inventory/:id/movements', (req: Request<{ id: string }>, res: Response<InventoryMovementResponse[]>) => {
    res.json([]);
});

/**
 * getInventoryAlerts returns low stock alerts.
 */
app.get('/api/inventory/alerts', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * getWarehouses returns available warehouses.
 */
app.get('/api/inventory/warehouses', (req: Request, res: Response<WarehouseResponse[]>) => {
    res.json([]);
});

/**
 * getInventoryByProduct returns inventory for a product.
 */
app.get('/api/inventory/product/:productId', (req: Request<{ productId: string }>, res: Response<Inventory[]>) => {
    res.json([]);
});

// ===== Review Controllers =====

/**
 * listReviews returns a paginated list of reviews.
 */
app.get('/api/reviews', (req: Request, res: Response<PageResult<Review>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * createReview creates a new review.
 */
app.post('/api/reviews', (req: Request<{}, {}, CreateReviewRequest>, res: Response<Review>) => {
    res.status(201).json({ id: 1, createdAt: '', updatedAt: '', productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 });
});

/**
 * getReview returns a single review by ID.
 */
app.get('/api/reviews/:id', (req: Request<{ id: string }>, res: Response<Review>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 });
});

/**
 * updateReview updates an existing review.
 */
app.put('/api/reviews/:id', (req: Request<{ id: string }>, res: Response<Review>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 });
});

/**
 * deleteReview removes a review by ID.
 */
app.delete('/api/reviews/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * getReviewsByProduct returns reviews for a product.
 */
app.get('/api/reviews/product/:productId', (req: Request<{ productId: string }>, res: Response<PageResult<Review>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * addReviewComment adds a comment to a review.
 */
app.post('/api/reviews/:id/comments', (req: Request<{ id: string }>, res: Response<CommentResponse>) => {
    res.status(201).json({ id: 1, reviewId: 1, userId: 1, username: '', content: '', createdAt: '' });
});

/**
 * getReviewComments returns comments for a review.
 */
app.get('/api/reviews/:id/comments', (req: Request<{ id: string }>, res: Response<CommentResponse[]>) => {
    res.json([]);
});

/**
 * moderateReview moderates a review.
 */
app.put('/api/reviews/:id/moderate', (req: Request<{ id: string }>, res: Response<Review>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 });
});

/**
 * getReviewsByUser returns reviews by a user.
 */
app.get('/api/reviews/user/:userId', (req: Request<{ userId: string }>, res: Response<PageResult<Review>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

// ===== Notification Controllers =====

/**
 * listNotifications returns a paginated list of notifications.
 */
app.get('/api/notifications', (req: Request, res: Response<PageResult<Notification>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * createNotification creates a new notification.
 */
app.post('/api/notifications', (req: Request<{}, {}, CreateNotificationRequest>, res: Response<Notification>) => {
    res.status(201).json({ id: 1, createdAt: '', updatedAt: '', userId: 1, type: '', title: '', content: '', read: false });
});

/**
 * getNotification returns a single notification by ID.
 */
app.get('/api/notifications/:id', (req: Request<{ id: string }>, res: Response<Notification>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', userId: 1, type: '', title: '', content: '', read: false });
});

/**
 * updateNotification updates an existing notification.
 */
app.put('/api/notifications/:id', (req: Request<{ id: string }>, res: Response<Notification>) => {
    res.json({ id: 1, createdAt: '', updatedAt: '', userId: 1, type: '', title: '', content: '', read: false });
});

/**
 * deleteNotification removes a notification by ID.
 */
app.delete('/api/notifications/:id', (req: Request<{ id: string }>, res: Response) => {
    res.status(204).send();
});

/**
 * getNotificationsByUser returns notifications for a user.
 */
app.get('/api/notifications/user/:userId', (req: Request<{ userId: string }>, res: Response<PageResult<Notification>>) => {
    res.json({ list: [], total: 0, pageNum: 1, pageSize: 20 });
});

/**
 * markNotificationAsRead marks a notification as read.
 */
app.put('/api/notifications/:id/read', (req: Request<{ id: string }>, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * markAllNotificationsAsRead marks all notifications as read for a user.
 */
app.put('/api/notifications/user/:userId/read-all', (req: Request<{ userId: string }>, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * sendNotification sends a notification.
 */
app.post('/api/notifications/send', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * getNotificationTemplates returns notification templates.
 */
app.get('/api/notifications/templates', (req: Request, res: Response<NotificationTemplateResponse[]>) => {
    res.json([]);
});

/**
 * getUnreadNotificationCount returns unread notification count.
 */
app.get('/api/notifications/user/:userId/unread-count', (req: Request<{ userId: string }>, res: Response<number>) => {
    res.json(0);
});

// ===== Analytics Controllers =====

/**
 * getAnalyticsOverview returns analytics overview.
 */
app.get('/api/analytics/overview', (req: Request, res: Response<AnalyticsOverviewResponse>) => {
    res.json({ totalRevenue: 0, totalOrders: 0, totalUsers: 0, totalProducts: 0, todayRevenue: 0, todayOrders: 0, newUsers: 0, conversionRate: 0 });
});

/**
 * getSalesStatistics returns sales statistics.
 */
app.get('/api/analytics/sales', (req: Request, res: Response<SalesStatisticsResponse>) => {
    res.json({ totalSales: 0, totalRefund: 0, totalOrders: 0, averageOrderValue: 0, dailySales: [] });
});

/**
 * getUserAnalytics returns user analytics.
 */
app.get('/api/analytics/users', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * getProductAnalytics returns product analytics.
 */
app.get('/api/analytics/products', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * getOrderAnalytics returns order analytics.
 */
app.get('/api/analytics/orders', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * getTrendAnalytics returns trend analytics.
 */
app.get('/api/analytics/trends', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

/**
 * getDashboard returns the analytics dashboard.
 */
app.get('/api/analytics/dashboard', (req: Request, res: Response<Result<null>>) => {
    res.json({ code: 200, message: 'success', data: null });
});

app.listen(3000, () => {
    console.log('Server running on port 3000');
});
