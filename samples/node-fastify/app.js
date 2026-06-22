const fastify = require('fastify')({ logger: true });

fastify.register(require('@fastify/multipart'));

// ===== User Routes =====

/**
 * listUsers returns all users.
 */
fastify.get('/users', listUsers);

/**
 * createUser creates a new user.
 */
fastify.post('/users', createUser);

/**
 * getUser returns a single user by ID.
 */
fastify.get('/users/:id', getUser);

/**
 * updateUser updates an existing user.
 */
fastify.put('/users/:id', updateUser);

/**
 * deleteUser removes a user by ID.
 */
fastify.delete('/users/:id', deleteUser);

/**
 * patchUser partially updates a user.
 */
fastify.patch('/users/:id', patchUser);

/**
 * userLogin handles user login.
 */
fastify.post('/users/login', userLogin);

/**
 * userRegister handles user registration.
 */
fastify.post('/users/register', userRegister);

/**
 * getUserProfile returns a user's profile.
 */
fastify.get('/users/:id/profile', getUserProfile);

/**
 * updateUserProfile updates a user's profile.
 */
fastify.put('/users/:id/profile', updateUserProfile);

/**
 * getUserAddresses returns a user's addresses.
 */
fastify.get('/users/:id/addresses', getUserAddresses);

/**
 * addUserAddress adds a new address for a user.
 */
fastify.post('/users/:id/addresses', addUserAddress);

/**
 * getUserFavorites returns a user's favorite products.
 */
fastify.get('/users/:id/favorites', getUserFavorites);

// ===== Product Routes =====

/**
 * listProducts returns a paginated list of products.
 */
fastify.get('/api/products', listProducts);

/**
 * createProduct creates a new product.
 */
fastify.post('/api/products', createProduct);

/**
 * getProduct returns a single product by ID.
 */
fastify.get('/api/products/:id', getProduct);

/**
 * updateProduct updates an existing product.
 */
fastify.put('/api/products/:id', updateProduct);

/**
 * deleteProduct removes a product by ID.
 */
fastify.delete('/api/products/:id', deleteProduct);

/**
 * searchProducts searches for products.
 */
fastify.get('/api/products/search', searchProducts);

/**
 * getProductDetail returns detailed product information.
 */
fastify.get('/api/products/:id/detail', getProductDetail);

/**
 * getProductsByCategory returns products in a category.
 */
fastify.get('/api/products/category/:categoryId', getProductsByCategory);

/**
 * getHotProducts returns hot products.
 */
fastify.get('/api/products/hot', getHotProducts);

/**
 * getNewProducts returns new products.
 */
fastify.get('/api/products/new', getNewProducts);

/**
 * addProductSpec adds a product specification.
 */
fastify.post('/api/products/:id/specs', addProductSpec);

/**
 * getProductSpecs returns product specifications.
 */
fastify.get('/api/products/:id/specs', getProductSpecs);

/**
 * getProductStatistics returns product statistics.
 */
fastify.get('/api/products/:id/statistics', getProductStatistics);

// ===== Order Routes =====

/**
 * listOrders returns a paginated list of orders.
 */
fastify.get('/api/orders', listOrders);

/**
 * createOrder creates a new order.
 */
fastify.post('/api/orders', createOrder);

/**
 * getOrder returns a single order by ID.
 */
fastify.get('/api/orders/:id', getOrder);

/**
 * updateOrder updates an existing order.
 */
fastify.put('/api/orders/:id', updateOrder);

/**
 * deleteOrder removes an order by ID.
 */
fastify.delete('/api/orders/:id', deleteOrder);

/**
 * cancelOrder cancels an order.
 */
fastify.post('/api/orders/:id/cancel', cancelOrder);

/**
 * trackOrder returns order tracking information.
 */
fastify.get('/api/orders/:id/track', trackOrder);

/**
 * addOrderItem adds an item to an order.
 */
fastify.post('/api/orders/:id/items', addOrderItem);

/**
 * refundOrder requests a refund for an order.
 */
fastify.post('/api/orders/:id/refund', refundOrder);

/**
 * getOrderStatistics returns order statistics.
 */
fastify.get('/api/orders/statistics', getOrderStatistics);

/**
 * getOrdersByUser returns orders for a user.
 */
fastify.get('/api/orders/user/:userId', getOrdersByUser);

// ===== Category Routes =====

/**
 * listCategories returns all categories.
 */
fastify.get('/api/categories', listCategories);

/**
 * createCategory creates a new category.
 */
fastify.post('/api/categories', createCategory);

/**
 * getCategory returns a single category by ID.
 */
fastify.get('/api/categories/:id', getCategory);

/**
 * updateCategory updates an existing category.
 */
fastify.put('/api/categories/:id', updateCategory);

/**
 * deleteCategory removes a category by ID.
 */
fastify.delete('/api/categories/:id', deleteCategory);

/**
 * getCategoryTree returns the category tree.
 */
fastify.get('/api/categories/tree', getCategoryTree);

/**
 * getCategoryProducts returns products in a category.
 */
fastify.get('/api/categories/:id/products', getCategoryProducts);

/**
 * updateCategorySort updates category sort order.
 */
fastify.put('/api/categories/:id/sort', updateCategorySort);

/**
 * getRootCategories returns root categories.
 */
fastify.get('/api/categories/roots', getRootCategories);

// ===== Payment Routes =====

/**
 * listPayments returns a paginated list of payments.
 */
fastify.get('/api/payments', listPayments);

/**
 * createPayment creates a new payment.
 */
fastify.post('/api/payments', createPayment);

/**
 * getPayment returns a single payment by ID.
 */
fastify.get('/api/payments/:id', getPayment);

/**
 * updatePayment updates an existing payment.
 */
fastify.put('/api/payments/:id', updatePayment);

/**
 * deletePayment removes a payment by ID.
 */
fastify.delete('/api/payments/:id', deletePayment);

/**
 * refundPayment refunds a payment.
 */
fastify.post('/api/payments/:id/refund', refundPayment);

/**
 * getPaymentRecords returns payment records.
 */
fastify.get('/api/payments/:id/records', getPaymentRecords);

/**
 * getPaymentChannels returns available payment channels.
 */
fastify.get('/api/payments/channels', getPaymentChannels);

/**
 * getPaymentStatistics returns payment statistics.
 */
fastify.get('/api/payments/statistics', getPaymentStatistics);

/**
 * confirmPayment confirms a payment.
 */
fastify.post('/api/payments/:id/confirm', confirmPayment);

// ===== Shipping Routes =====

/**
 * listShipping returns a paginated list of shipping records.
 */
fastify.get('/api/shipping', listShipping);

/**
 * createShipping creates a new shipping record.
 */
fastify.post('/api/shipping', createShipping);

/**
 * getShipping returns a single shipping record by ID.
 */
fastify.get('/api/shipping/:id', getShipping);

/**
 * updateShipping updates an existing shipping record.
 */
fastify.put('/api/shipping/:id', updateShipping);

/**
 * deleteShipping removes a shipping record by ID.
 */
fastify.delete('/api/shipping/:id', deleteShipping);

/**
 * trackShipping returns shipping tracking information.
 */
fastify.get('/api/shipping/:id/track', trackShipping);

/**
 * getCarriers returns available carriers.
 */
fastify.get('/api/shipping/carriers', getCarriers);

/**
 * calculateShippingRates calculates shipping rates.
 */
fastify.post('/api/shipping/rates', calculateShippingRates);

/**
 * shipOrder ships an order.
 */
fastify.post('/api/shipping/:id/ship', shipOrder);

/**
 * deliverOrder delivers an order.
 */
fastify.post('/api/shipping/:id/deliver', deliverOrder);

// ===== Inventory Routes =====

/**
 * listInventory returns a paginated list of inventory.
 */
fastify.get('/api/inventory', listInventory);

/**
 * createInventory creates a new inventory record.
 */
fastify.post('/api/inventory', createInventory);

/**
 * getInventory returns a single inventory record by ID.
 */
fastify.get('/api/inventory/:id', getInventory);

/**
 * updateInventory updates an existing inventory record.
 */
fastify.put('/api/inventory/:id', updateInventory);

/**
 * deleteInventory removes an inventory record by ID.
 */
fastify.delete('/api/inventory/:id', deleteInventory);

/**
 * updateStock updates inventory stock.
 */
fastify.put('/api/inventory/:id/stock', updateStock);

/**
 * getInventoryMovements returns inventory movement records.
 */
fastify.get('/api/inventory/:id/movements', getInventoryMovements);

/**
 * getInventoryAlerts returns low stock alerts.
 */
fastify.get('/api/inventory/alerts', getInventoryAlerts);

/**
 * getWarehouses returns available warehouses.
 */
fastify.get('/api/inventory/warehouses', getWarehouses);

/**
 * getInventoryByProduct returns inventory for a product.
 */
fastify.get('/api/inventory/product/:productId', getInventoryByProduct);

// ===== Review Routes =====

/**
 * listReviews returns a paginated list of reviews.
 */
fastify.get('/api/reviews', listReviews);

/**
 * createReview creates a new review.
 */
fastify.post('/api/reviews', createReview);

/**
 * getReview returns a single review by ID.
 */
fastify.get('/api/reviews/:id', getReview);

/**
 * updateReview updates an existing review.
 */
fastify.put('/api/reviews/:id', updateReview);

/**
 * deleteReview removes a review by ID.
 */
fastify.delete('/api/reviews/:id', deleteReview);

/**
 * getReviewsByProduct returns reviews for a product.
 */
fastify.get('/api/reviews/product/:productId', getReviewsByProduct);

/**
 * addReviewComment adds a comment to a review.
 */
fastify.post('/api/reviews/:id/comments', addReviewComment);

/**
 * getReviewComments returns comments for a review.
 */
fastify.get('/api/reviews/:id/comments', getReviewComments);

/**
 * moderateReview moderates a review.
 */
fastify.put('/api/reviews/:id/moderate', moderateReview);

/**
 * getReviewsByUser returns reviews by a user.
 */
fastify.get('/api/reviews/user/:userId', getReviewsByUser);

// ===== Notification Routes =====

/**
 * listNotifications returns a paginated list of notifications.
 */
fastify.get('/api/notifications', listNotifications);

/**
 * createNotification creates a new notification.
 */
fastify.post('/api/notifications', createNotification);

/**
 * getNotification returns a single notification by ID.
 */
fastify.get('/api/notifications/:id', getNotification);

/**
 * updateNotification updates an existing notification.
 */
fastify.put('/api/notifications/:id', updateNotification);

/**
 * deleteNotification removes a notification by ID.
 */
fastify.delete('/api/notifications/:id', deleteNotification);

/**
 * getNotificationsByUser returns notifications for a user.
 */
fastify.get('/api/notifications/user/:userId', getNotificationsByUser);

/**
 * markNotificationAsRead marks a notification as read.
 */
fastify.put('/api/notifications/:id/read', markNotificationAsRead);

/**
 * markAllNotificationsAsRead marks all notifications as read for a user.
 */
fastify.put('/api/notifications/user/:userId/read-all', markAllNotificationsAsRead);

/**
 * sendNotification sends a notification.
 */
fastify.post('/api/notifications/send', sendNotification);

/**
 * getNotificationTemplates returns notification templates.
 */
fastify.get('/api/notifications/templates', getNotificationTemplates);

/**
 * getUnreadNotificationCount returns unread notification count.
 */
fastify.get('/api/notifications/user/:userId/unread-count', getUnreadNotificationCount);

// ===== Analytics Routes =====

/**
 * getAnalyticsOverview returns analytics overview.
 */
fastify.get('/api/analytics/overview', getAnalyticsOverview);

/**
 * getSalesStatistics returns sales statistics.
 */
fastify.get('/api/analytics/sales', getSalesStatistics);

/**
 * getUserAnalytics returns user analytics.
 */
fastify.get('/api/analytics/users', getUserAnalytics);

/**
 * getProductAnalytics returns product analytics.
 */
fastify.get('/api/analytics/products', getProductAnalytics);

/**
 * getOrderAnalytics returns order analytics.
 */
fastify.get('/api/analytics/orders', getOrderAnalytics);

/**
 * getTrendAnalytics returns trend analytics.
 */
fastify.get('/api/analytics/trends', getTrendAnalytics);

/**
 * getDashboard returns the analytics dashboard.
 */
fastify.get('/api/analytics/dashboard', getDashboard);

// ===== Misc Routes =====

/**
 * uploadFile handles file uploads.
 */
fastify.post('/upload', uploadFile);

// ===== User Handlers =====

function listUsers(request, reply) {
    const { name, role = 'user' } = request.query;
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createUser(request, reply) {
    const { name, email } = request.body;
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, name, email, phone: '', avatar: '', status: 1, tags: [] } };
}

function getUser(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: 'test', email: 'test@example.com', phone: '', avatar: '', status: 1, tags: [] } };
}

function updateUser(request, reply) {
    const { name, email } = request.body;
    return { code: 200, message: 'success', data: { id: 1, name: name || 'unknown', email: email || '', phone: '', avatar: '', status: 1, tags: [] } };
}

function deleteUser(request, reply) {
    reply.code(204);
    return '';
}

function patchUser(request, reply) {
    const { name = 'unknown' } = request.query;
    return { code: 200, message: 'success', data: { id: request.params.id } };
}

function userLogin(request, reply) {
    return { code: 200, message: 'success', data: { token: 'xxx', userId: 1, username: 'test' } };
}

function userRegister(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: 'test', email: 'test@example.com', phone: '', avatar: '', status: 1, tags: [] } };
}

function getUserProfile(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, username: 'test', email: 'test@example.com', phone: '', avatar: '', nickname: '', bio: '', gender: 0, birthday: '' } };
}

function updateUserProfile(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, username: 'test', email: 'test@example.com', phone: '', avatar: '', nickname: '', bio: '', gender: 0, birthday: '' } };
}

function getUserAddresses(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function addUserAddress(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, receiver: '', phone: '', province: '', city: '', district: '', detail: '', isDefault: false } };
}

function getUserFavorites(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

// ===== Product Handlers =====

function listProducts(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createProduct(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, name: '', description: '', price: 0, originalPrice: 0, stock: 0, categoryId: 0, mainImage: '', status: 1, salesCount: 0 } };
}

function getProduct(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: '', description: '', price: 0, originalPrice: 0, stock: 0, categoryId: 0, mainImage: '', status: 1, salesCount: 0 } };
}

function updateProduct(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: '', description: '', price: 0, originalPrice: 0, stock: 0, categoryId: 0, mainImage: '', status: 1, salesCount: 0 } };
}

function deleteProduct(request, reply) {
    reply.code(204);
    return '';
}

function searchProducts(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function getProductDetail(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: '', description: '', price: 0, originalPrice: 0, categoryId: 0, categoryName: '', stock: 0, sales: 0, mainImage: '', images: [], specs: [], averageRating: 0, reviewCount: 0 } };
}

function getProductsByCategory(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function getHotProducts(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getNewProducts(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function addProductSpec(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, name: '', value: '', price: 0, stock: 0 } };
}

function getProductSpecs(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getProductStatistics(request, reply) {
    return { code: 200, message: 'success', data: { productId: 1, views: 0, sales: 0, favorites: 0, averageRating: 0, reviewCount: 0 } };
}

// ===== Order Handlers =====

function listOrders(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createOrder(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, orderNo: '', userId: 1, totalAmount: 0, finalAmount: 0, status: 0, items: [] } };
}

function getOrder(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, orderNo: '', userId: 1, totalAmount: 0, finalAmount: 0, status: 0, items: [] } };
}

function updateOrder(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, orderNo: '', userId: 1, totalAmount: 0, finalAmount: 0, status: 0, items: [] } };
}

function deleteOrder(request, reply) {
    reply.code(204);
    return '';
}

function cancelOrder(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, orderNo: '', userId: 1, totalAmount: 0, status: 0 } };
}

function trackOrder(request, reply) {
    return { code: 200, message: 'success', data: { orderNo: '', status: 0, trackPoints: [] } };
}

function addOrderItem(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, orderNo: '', userId: 1, totalAmount: 0, status: 0 } };
}

function refundOrder(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, refundNo: '', amount: 0, status: 0, createdAt: '' } };
}

function getOrderStatistics(request, reply) {
    return { code: 200, message: 'success', data: { totalOrders: 0, totalAmount: 0, pendingOrders: 0, completedOrders: 0, cancelledOrders: 0 } };
}

function getOrdersByUser(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

// ===== Category Handlers =====

function listCategories(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function createCategory(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, name: '', parentId: 0, sort: 0, icon: '', status: 1 } };
}

function getCategory(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: '', parentId: 0, sort: 0, icon: '', status: 1 } };
}

function updateCategory(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: '', parentId: 0, sort: 0, icon: '', status: 1 } };
}

function deleteCategory(request, reply) {
    reply.code(204);
    return '';
}

function getCategoryTree(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getCategoryProducts(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function updateCategorySort(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, name: '', parentId: 0, sort: 0, icon: '', status: 1 } };
}

function getRootCategories(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

// ===== Payment Handlers =====

function listPayments(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createPayment(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' } };
}

function getPayment(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' } };
}

function updatePayment(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' } };
}

function deletePayment(request, reply) {
    reply.code(204);
    return '';
}

function refundPayment(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, refundNo: '', amount: 0, status: 0, createdAt: '' } };
}

function getPaymentRecords(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getPaymentChannels(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getPaymentStatistics(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function confirmPayment(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, paymentNo: '', orderId: 1, amount: 0, method: '', status: 0, paidAt: '' } };
}

// ===== Shipping Handlers =====

function listShipping(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createShipping(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' } };
}

function getShipping(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' } };
}

function updateShipping(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' } };
}

function deleteShipping(request, reply) {
    reply.code(204);
    return '';
}

function trackShipping(request, reply) {
    return { code: 200, message: 'success', data: { shippingNo: '', carrier: '', trackingNo: '', status: 0, trackPoints: [] } };
}

function getCarriers(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function calculateShippingRates(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function shipOrder(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' } };
}

function deliverOrder(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, shippingNo: '', orderId: 1, carrier: '', trackingNo: '', status: 0, shippedAt: '', deliveredAt: '' } };
}

// ===== Inventory Handlers =====

function listInventory(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createInventory(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 } };
}

function getInventory(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 } };
}

function updateInventory(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 } };
}

function deleteInventory(request, reply) {
    reply.code(204);
    return '';
}

function updateStock(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, productId: 1, warehouseId: 1, quantity: 0, locked: 0, available: 0 } };
}

function getInventoryMovements(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getInventoryAlerts(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function getWarehouses(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getInventoryByProduct(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

// ===== Review Handlers =====

function listReviews(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createReview(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 } };
}

function getReview(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 } };
}

function updateReview(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 } };
}

function deleteReview(request, reply) {
    reply.code(204);
    return '';
}

function getReviewsByProduct(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function addReviewComment(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, reviewId: 1, userId: 1, username: '', content: '', createdAt: '' } };
}

function getReviewComments(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function moderateReview(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, productId: 1, userId: 1, rating: 5, content: '', images: '', status: 0 } };
}

function getReviewsByUser(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

// ===== Notification Handlers =====

function listNotifications(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function createNotification(request, reply) {
    reply.code(201);
    return { code: 200, message: 'success', data: { id: 1, userId: 1, type: '', title: '', content: '', read: false } };
}

function getNotification(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, userId: 1, type: '', title: '', content: '', read: false } };
}

function updateNotification(request, reply) {
    return { code: 200, message: 'success', data: { id: 1, userId: 1, type: '', title: '', content: '', read: false } };
}

function deleteNotification(request, reply) {
    reply.code(204);
    return '';
}

function getNotificationsByUser(request, reply) {
    return { code: 200, message: 'success', data: { list: [], total: 0, pageNum: 1, pageSize: 20 } };
}

function markNotificationAsRead(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function markAllNotificationsAsRead(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function sendNotification(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function getNotificationTemplates(request, reply) {
    return { code: 200, message: 'success', data: [] };
}

function getUnreadNotificationCount(request, reply) {
    return { code: 200, message: 'success', data: 0 };
}

// ===== Analytics Handlers =====

function getAnalyticsOverview(request, reply) {
    return { code: 200, message: 'success', data: { totalRevenue: 0, totalOrders: 0, totalUsers: 0, totalProducts: 0, todayRevenue: 0, todayOrders: 0, newUsers: 0, conversionRate: 0 } };
}

function getSalesStatistics(request, reply) {
    return { code: 200, message: 'success', data: { totalSales: 0, totalRefund: 0, totalOrders: 0, averageOrderValue: 0, dailySales: [] } };
}

function getUserAnalytics(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function getProductAnalytics(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function getOrderAnalytics(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function getTrendAnalytics(request, reply) {
    return { code: 200, message: 'success', data: null };
}

function getDashboard(request, reply) {
    return { code: 200, message: 'success', data: null };
}

// ===== Misc Handlers =====

function uploadFile(request, reply) {
    return { code: 200, message: 'success', data: { status: 'ok' } };
}

const start = async () => {
    try {
        await fastify.listen({ port: 3000 });
    } catch (err) {
        fastify.log.error(err);
        process.exit(1);
    }
};

start();
