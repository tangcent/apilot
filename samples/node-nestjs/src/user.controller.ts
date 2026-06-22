import { Controller, Get, Post, Put, Delete, Patch, Param, Body, Query, Headers } from '@nestjs/common';
import { ApiOkResponse, ApiCreatedResponse } from '@nestjs/swagger';

// ===== Base Controller =====

export class BaseController {
  @Get('health')
  health() {
    return { status: 'ok' };
  }
}

export class BaseCrudController<Req, Res> {
  @Get()
  list(@Query('page') page: number, @Query('limit') limit: number): Promise<{ list: Res[]; total: number }> {
    return null;
  }

  @Post()
  create(@Body() body: Req): Promise<Res> {
    return null;
  }

  @Get(':id')
  get(@Param('id') id: string): Promise<Res> {
    return null;
  }

  @Put(':id')
  update(@Param('id') id: string, @Body() body: Req): Promise<Res> {
    return null;
  }

  @Delete(':id')
  remove(@Param('id') id: string): Promise<void> {
    return null;
  }
}

// ===== User Controller =====

/**
 * UserController manages user operations.
 */
@Controller('users')
export class UserController extends BaseCrudController<CreateUserReq, UserVO> {

  /**
   * userLogin handles user login.
   */
  @Post('login')
  @ApiOkResponse({ description: 'Login result', type: LoginResponse })
  userLogin(@Body() req: LoginReq): Promise<LoginResponse> {
    return null;
  }

  /**
   * userRegister handles user registration.
   */
  @Post('register')
  @ApiCreatedResponse({ description: 'Registered user', type: UserVO })
  userRegister(@Body() req: CreateUserReq): Promise<UserVO> {
    return null;
  }

  /**
   * getUserProfile returns a user's profile.
   */
  @Get(':id/profile')
  @ApiOkResponse({ description: 'User profile', type: UserProfileVO })
  getUserProfile(@Param('id') id: string): Promise<UserProfileVO> {
    return null;
  }

  /**
   * updateUserProfile updates a user's profile.
   */
  @Put(':id/profile')
  @ApiOkResponse({ description: 'Updated profile', type: UserProfileVO })
  updateUserProfile(@Param('id') id: string, @Body() req: UpdateProfileReq): Promise<UserProfileVO> {
    return null;
  }

  /**
   * getUserAddresses returns a user's addresses.
   */
  @Get(':id/addresses')
  @ApiOkResponse({ description: 'User addresses', type: AddressVO })
  getUserAddresses(@Param('id') id: string): Promise<AddressVO[]> {
    return null;
  }

  /**
   * addUserAddress adds a new address for a user.
   */
  @Post(':id/addresses')
  @ApiCreatedResponse({ description: 'Added address', type: AddressVO })
  addUserAddress(@Param('id') id: string, @Body() req: AddressReq): Promise<AddressVO> {
    return null;
  }

  /**
   * getUserFavorites returns a user's favorite products.
   */
  @Get(':id/favorites')
  @ApiOkResponse({ description: 'User favorites', type: ProductVO })
  getUserFavorites(@Param('id') id: string): Promise<ProductVO[]> {
    return null;
  }

  /**
   * patchUser partially updates a user.
   */
  @Patch(':id')
  patchUser(@Param('id') id: string, @Body() body: UpdateUserReq): Promise<UserVO> {
    return null;
  }
}

// ===== Product Controller =====

/**
 * ProductController manages product operations.
 */
@Controller('api/products')
export class ProductController extends BaseCrudController<CreateProductReq, ProductVO> {

  /**
   * searchProducts searches for products.
   */
  @Get('search')
  @ApiOkResponse({ description: 'Search results', type: ProductVO })
  searchProducts(@Query('keyword') keyword: string, @Query('categoryId') categoryId: number): Promise<{ list: ProductVO[]; total: number }> {
    return null;
  }

  /**
   * getProductDetail returns detailed product information.
   */
  @Get(':id/detail')
  @ApiOkResponse({ description: 'Product detail', type: ProductDetailVO })
  getProductDetail(@Param('id') id: string): Promise<ProductDetailVO> {
    return null;
  }

  /**
   * getProductsByCategory returns products in a category.
   */
  @Get('category/:categoryId')
  @ApiOkResponse({ description: 'Products in category', type: ProductVO })
  getProductsByCategory(@Param('categoryId') categoryId: string): Promise<ProductVO[]> {
    return null;
  }

  /**
   * getHotProducts returns hot products.
   */
  @Get('hot')
  @ApiOkResponse({ description: 'Hot products', type: ProductVO })
  getHotProducts(): Promise<ProductVO[]> {
    return null;
  }

  /**
   * getNewProducts returns new products.
   */
  @Get('new')
  @ApiOkResponse({ description: 'New products', type: ProductVO })
  getNewProducts(): Promise<ProductVO[]> {
    return null;
  }

  /**
   * addProductSpec adds a product specification.
   */
  @Post(':id/specs')
  @ApiCreatedResponse({ description: 'Added spec', type: ProductSpecVO })
  addProductSpec(@Param('id') id: string, @Body() req: ProductSpecReq): Promise<ProductSpecVO> {
    return null;
  }

  /**
   * getProductSpecs returns product specifications.
   */
  @Get(':id/specs')
  @ApiOkResponse({ description: 'Product specs', type: ProductSpecVO })
  getProductSpecs(@Param('id') id: string): Promise<ProductSpecVO[]> {
    return null;
  }

  /**
   * getProductStatistics returns product statistics.
   */
  @Get(':id/statistics')
  @ApiOkResponse({ description: 'Product statistics', type: ProductStatisticsVO })
  getProductStatistics(@Param('id') id: string): Promise<ProductStatisticsVO> {
    return null;
  }
}

// ===== Order Controller =====

/**
 * OrderController manages order operations.
 */
@Controller('api/orders')
export class OrderController extends BaseCrudController<CreateOrderReq, OrderVO> {

  /**
   * cancelOrder cancels an order.
   */
  @Post(':id/cancel')
  @ApiOkResponse({ description: 'Cancelled order', type: OrderVO })
  cancelOrder(@Param('id') id: string, @Body() req: CancelOrderReq): Promise<OrderVO> {
    return null;
  }

  /**
   * trackOrder returns order tracking information.
   */
  @Get(':id/track')
  @ApiOkResponse({ description: 'Order tracking', type: OrderTrackResponse })
  trackOrder(@Param('id') id: string): Promise<OrderTrackResponse> {
    return null;
  }

  /**
   * addOrderItem adds an item to an order.
   */
  @Post(':id/items')
  @ApiCreatedResponse({ description: 'Added item', type: OrderItemVO })
  addOrderItem(@Param('id') id: string, @Body() req: OrderItemReq): Promise<OrderItemVO> {
    return null;
  }

  /**
   * refundOrder requests a refund for an order.
   */
  @Post(':id/refund')
  @ApiOkResponse({ description: 'Refund result', type: RefundVO })
  refundOrder(@Param('id') id: string, @Body() req: RefundReq): Promise<RefundVO> {
    return null;
  }

  /**
   * getOrderStatistics returns order statistics.
   */
  @Get('statistics')
  @ApiOkResponse({ description: 'Order statistics', type: OrderStatisticsVO })
  getOrderStatistics(): Promise<OrderStatisticsVO> {
    return null;
  }

  /**
   * getOrdersByUser returns orders for a user.
   */
  @Get('user/:userId')
  @ApiOkResponse({ description: 'User orders', type: OrderVO })
  getOrdersByUser(@Param('userId') userId: string): Promise<OrderVO[]> {
    return null;
  }
}

// ===== Category Controller =====

/**
 * CategoryController manages category operations.
 */
@Controller('api/categories')
export class CategoryController extends BaseCrudController<CreateCategoryReq, CategoryVO> {

  /**
   * getCategoryTree returns the category tree.
   */
  @Get('tree')
  @ApiOkResponse({ description: 'Category tree', type: CategoryTreeVO })
  getCategoryTree(): Promise<CategoryTreeVO[]> {
    return null;
  }

  /**
   * getCategoryProducts returns products in a category.
   */
  @Get(':id/products')
  @ApiOkResponse({ description: 'Category products', type: ProductVO })
  getCategoryProducts(@Param('id') id: string): Promise<ProductVO[]> {
    return null;
  }

  /**
   * updateCategorySort updates category sort order.
   */
  @Put(':id/sort')
  @ApiOkResponse({ description: 'Updated sort', type: CategoryVO })
  updateCategorySort(@Param('id') id: string, @Body() req: SortReq): Promise<CategoryVO> {
    return null;
  }

  /**
   * getRootCategories returns root categories.
   */
  @Get('roots')
  @ApiOkResponse({ description: 'Root categories', type: CategoryVO })
  getRootCategories(): Promise<CategoryVO[]> {
    return null;
  }
}

// ===== Payment Controller =====

/**
 * PaymentController manages payment operations.
 */
@Controller('api/payments')
export class PaymentController extends BaseCrudController<CreatePaymentReq, PaymentVO> {

  /**
   * refundPayment refunds a payment.
   */
  @Post(':id/refund')
  @ApiOkResponse({ description: 'Refund result', type: RefundVO })
  refundPayment(@Param('id') id: string, @Body() req: RefundReq): Promise<RefundVO> {
    return null;
  }

  /**
   * getPaymentRecords returns payment records.
   */
  @Get(':id/records')
  @ApiOkResponse({ description: 'Payment records', type: PaymentRecordVO })
  getPaymentRecords(@Param('id') id: string): Promise<PaymentRecordVO[]> {
    return null;
  }

  /**
   * getPaymentChannels returns available payment channels.
   */
  @Get('channels')
  @ApiOkResponse({ description: 'Payment channels', type: PaymentChannelVO })
  getPaymentChannels(): Promise<PaymentChannelVO[]> {
    return null;
  }

  /**
   * getPaymentStatistics returns payment statistics.
   */
  @Get('statistics')
  @ApiOkResponse({ description: 'Payment statistics', type: PaymentStatisticsVO })
  getPaymentStatistics(): Promise<PaymentStatisticsVO> {
    return null;
  }

  /**
   * confirmPayment confirms a payment.
   */
  @Post(':id/confirm')
  @ApiOkResponse({ description: 'Confirmed payment', type: PaymentVO })
  confirmPayment(@Param('id') id: string): Promise<PaymentVO> {
    return null;
  }
}

// ===== Shipping Controller =====

/**
 * ShippingController manages shipping operations.
 */
@Controller('api/shipping')
export class ShippingController extends BaseCrudController<CreateShippingReq, ShippingVO> {

  /**
   * trackShipping returns shipping tracking information.
   */
  @Get(':id/track')
  @ApiOkResponse({ description: 'Shipping tracking', type: ShippingTrackResponse })
  trackShipping(@Param('id') id: string): Promise<ShippingTrackResponse> {
    return null;
  }

  /**
   * getCarriers returns available carriers.
   */
  @Get('carriers')
  @ApiOkResponse({ description: 'Carriers', type: CarrierVO })
  getCarriers(): Promise<CarrierVO[]> {
    return null;
  }

  /**
   * calculateShippingRates calculates shipping rates.
   */
  @Post('rates')
  @ApiOkResponse({ description: 'Shipping rates', type: ShippingRateVO })
  calculateShippingRates(@Body() req: ShippingRateReq): Promise<ShippingRateVO[]> {
    return null;
  }

  /**
   * shipOrder ships an order.
   */
  @Post(':id/ship')
  @ApiOkResponse({ description: 'Shipped order', type: ShippingVO })
  shipOrder(@Param('id') id: string): Promise<ShippingVO> {
    return null;
  }

  /**
   * deliverOrder delivers an order.
   */
  @Post(':id/deliver')
  @ApiOkResponse({ description: 'Delivered order', type: ShippingVO })
  deliverOrder(@Param('id') id: string): Promise<ShippingVO> {
    return null;
  }
}

// ===== Inventory Controller =====

/**
 * InventoryController manages inventory operations.
 */
@Controller('api/inventory')
export class InventoryController extends BaseCrudController<CreateInventoryReq, InventoryVO> {

  /**
   * updateStock updates inventory stock.
   */
  @Put(':id/stock')
  @ApiOkResponse({ description: 'Updated stock', type: InventoryVO })
  updateStock(@Param('id') id: string, @Body() req: UpdateStockReq): Promise<InventoryVO> {
    return null;
  }

  /**
   * getInventoryMovements returns inventory movement records.
   */
  @Get(':id/movements')
  @ApiOkResponse({ description: 'Inventory movements', type: InventoryMovementVO })
  getInventoryMovements(@Param('id') id: string): Promise<InventoryMovementVO[]> {
    return null;
  }

  /**
   * getInventoryAlerts returns low stock alerts.
   */
  @Get('alerts')
  @ApiOkResponse({ description: 'Inventory alerts', type: InventoryAlertVO })
  getInventoryAlerts(): Promise<InventoryAlertVO[]> {
    return null;
  }

  /**
   * getWarehouses returns available warehouses.
   */
  @Get('warehouses')
  @ApiOkResponse({ description: 'Warehouses', type: WarehouseVO })
  getWarehouses(): Promise<WarehouseVO[]> {
    return null;
  }

  /**
   * getInventoryByProduct returns inventory for a product.
   */
  @Get('product/:productId')
  @ApiOkResponse({ description: 'Inventory by product', type: InventoryVO })
  getInventoryByProduct(@Param('productId') productId: string): Promise<InventoryVO[]> {
    return null;
  }
}

// ===== Review Controller =====

/**
 * ReviewController manages review operations.
 */
@Controller('api/reviews')
export class ReviewController extends BaseCrudController<CreateReviewReq, ReviewVO> {

  /**
   * getReviewsByProduct returns reviews for a product.
   */
  @Get('product/:productId')
  @ApiOkResponse({ description: 'Reviews by product', type: ReviewVO })
  getReviewsByProduct(@Param('productId') productId: string): Promise<ReviewVO[]> {
    return null;
  }

  /**
   * addReviewComment adds a comment to a review.
   */
  @Post(':id/comments')
  @ApiCreatedResponse({ description: 'Added comment', type: ReviewCommentVO })
  addReviewComment(@Param('id') id: string, @Body() req: ReviewCommentReq): Promise<ReviewCommentVO> {
    return null;
  }

  /**
   * getReviewComments returns comments for a review.
   */
  @Get(':id/comments')
  @ApiOkResponse({ description: 'Review comments', type: ReviewCommentVO })
  getReviewComments(@Param('id') id: string): Promise<ReviewCommentVO[]> {
    return null;
  }

  /**
   * moderateReview moderates a review.
   */
  @Put(':id/moderate')
  @ApiOkResponse({ description: 'Moderated review', type: ReviewVO })
  moderateReview(@Param('id') id: string, @Body() req: ModerateReq): Promise<ReviewVO> {
    return null;
  }

  /**
   * getReviewsByUser returns reviews by a user.
   */
  @Get('user/:userId')
  @ApiOkResponse({ description: 'Reviews by user', type: ReviewVO })
  getReviewsByUser(@Param('userId') userId: string): Promise<ReviewVO[]> {
    return null;
  }
}

// ===== Notification Controller =====

/**
 * NotificationController manages notification operations.
 */
@Controller('api/notifications')
export class NotificationController extends BaseCrudController<CreateNotificationReq, NotificationVO> {

  /**
   * getNotificationsByUser returns notifications for a user.
   */
  @Get('user/:userId')
  @ApiOkResponse({ description: 'User notifications', type: NotificationVO })
  getNotificationsByUser(@Param('userId') userId: string): Promise<NotificationVO[]> {
    return null;
  }

  /**
   * markNotificationAsRead marks a notification as read.
   */
  @Put(':id/read')
  @ApiOkResponse({ description: 'Marked as read' })
  markNotificationAsRead(@Param('id') id: string): Promise<void> {
    return null;
  }

  /**
   * markAllNotificationsAsRead marks all notifications as read for a user.
   */
  @Put('user/:userId/read-all')
  @ApiOkResponse({ description: 'All marked as read' })
  markAllNotificationsAsRead(@Param('userId') userId: string): Promise<void> {
    return null;
  }

  /**
   * sendNotification sends a notification.
   */
  @Post('send')
  @ApiOkResponse({ description: 'Sent notification', type: NotificationVO })
  sendNotification(@Body() req: SendNotificationReq): Promise<NotificationVO> {
    return null;
  }

  /**
   * getNotificationTemplates returns notification templates.
   */
  @Get('templates')
  @ApiOkResponse({ description: 'Notification templates', type: NotificationTemplateVO })
  getNotificationTemplates(): Promise<NotificationTemplateVO[]> {
    return null;
  }

  /**
   * getUnreadNotificationCount returns unread notification count.
   */
  @Get('user/:userId/unread-count')
  @ApiOkResponse({ description: 'Unread count' })
  getUnreadNotificationCount(@Param('userId') userId: string): Promise<number> {
    return null;
  }
}

// ===== Analytics Controller =====

/**
 * AnalyticsController manages analytics operations.
 */
@Controller('api/analytics')
export class AnalyticsController extends BaseController {

  /**
   * getAnalyticsOverview returns analytics overview.
   */
  @Get('overview')
  @ApiOkResponse({ description: 'Analytics overview', type: AnalyticsOverviewVO })
  getAnalyticsOverview(): Promise<AnalyticsOverviewVO> {
    return null;
  }

  /**
   * getSalesStatistics returns sales statistics.
   */
  @Get('sales')
  @ApiOkResponse({ description: 'Sales statistics', type: SalesStatisticsVO })
  getSalesStatistics(@Query('startDate') startDate: string, @Query('endDate') endDate: string): Promise<SalesStatisticsVO> {
    return null;
  }

  /**
   * getUserAnalytics returns user analytics.
   */
  @Get('users')
  @ApiOkResponse({ description: 'User analytics', type: UserAnalyticsVO })
  getUserAnalytics(@Query('startDate') startDate: string, @Query('endDate') endDate: string): Promise<UserAnalyticsVO> {
    return null;
  }

  /**
   * getProductAnalytics returns product analytics.
   */
  @Get('products')
  @ApiOkResponse({ description: 'Product analytics', type: ProductAnalyticsVO })
  getProductAnalytics(@Query('startDate') startDate: string, @Query('endDate') endDate: string): Promise<ProductAnalyticsVO> {
    return null;
  }

  /**
   * getOrderAnalytics returns order analytics.
   */
  @Get('orders')
  @ApiOkResponse({ description: 'Order analytics', type: OrderAnalyticsVO })
  getOrderAnalytics(@Query('startDate') startDate: string, @Query('endDate') endDate: string): Promise<OrderAnalyticsVO> {
    return null;
  }

  /**
   * getTrendAnalytics returns trend analytics.
   */
  @Get('trends')
  @ApiOkResponse({ description: 'Trend analytics', type: TrendAnalyticsVO })
  getTrendAnalytics(@Query('startDate') startDate: string, @Query('endDate') endDate: string): Promise<TrendAnalyticsVO> {
    return null;
  }

  /**
   * getDashboard returns the analytics dashboard.
   */
  @Get('dashboard')
  @ApiOkResponse({ description: 'Dashboard', type: DashboardVO })
  getDashboard(): Promise<DashboardVO> {
    return null;
  }
}

// ===== DTOs =====

class CreateUserReq {
  username: string;
  email: string;
  phone: string;
  password: string;
  avatar: string;
  status: number;
  tags: string[];
}

class UpdateUserReq {
  username?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  status?: number;
  tags?: string[];
}

class LoginReq {
  username: string;
  password: string;
  captcha: string;
}

class UpdateProfileReq {
  nickname: string;
  bio: string;
  gender: number;
  birthday: string;
  avatar: string;
}

class AddressReq {
  receiver: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  detail: string;
  isDefault: boolean;
}

class CreateProductReq {
  name: string;
  description: string;
  price: number;
  originalPrice: number;
  stock: number;
  categoryId: number;
  mainImage: string;
  images: string[];
  status: number;
}

class ProductSpecReq {
  name: string;
  value: string;
  price: number;
  stock: number;
}

class CreateOrderReq {
  userId: number;
  addressId: number;
  items: OrderItemReq[];
  remark: string;
  couponCode: string;
}

class OrderItemReq {
  productId: number;
  quantity: number;
  specId?: number;
}

class CancelOrderReq {
  reason: string;
}

class RefundReq {
  amount: number;
  reason: string;
}

class CreateCategoryReq {
  name: string;
  parentId: number;
  sort: number;
  icon: string;
  status: number;
}

class SortReq {
  sort: number;
}

class CreatePaymentReq {
  orderId: number;
  amount: number;
  method: string;
  channel: string;
}

class CreateShippingReq {
  orderId: number;
  carrier: string;
  trackingNo: string;
}

class ShippingRateReq {
  from: string;
  to: string;
  weight: number;
  volume: number;
}

class CreateInventoryReq {
  productId: number;
  warehouseId: number;
  quantity: number;
}

class UpdateStockReq {
  quantity: number;
  reason: string;
}

class CreateReviewReq {
  productId: number;
  rating: number;
  content: string;
  images: string[];
}

class ReviewCommentReq {
  userId: number;
  content: string;
}

class ModerateReq {
  status: number;
  reason: string;
}

class CreateNotificationReq {
  userId: number;
  type: string;
  title: string;
  content: string;
}

class SendNotificationReq {
  userIds: number[];
  type: string;
  title: string;
  content: string;
}

// ===== VOs =====

class UserVO {
  id: number;
  username: string;
  email: string;
  phone: string;
  avatar: string;
  status: number;
  tags: string[];
  createdAt: string;
  updatedAt: string;
}

class LoginResponse {
  token: string;
  userId: number;
  username: string;
  expiresIn: number;
}

class UserProfileVO {
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

class AddressVO {
  id: number;
  userId: number;
  receiver: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  detail: string;
  isDefault: boolean;
}

class ProductVO {
  id: number;
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

class ProductDetailVO {
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

class ProductSpecVO {
  id: number;
  name: string;
  value: string;
  price: number;
  stock: number;
}

class ProductStatisticsVO {
  productId: number;
  views: number;
  sales: number;
  favorites: number;
  averageRating: number;
  reviewCount: number;
}

class OrderVO {
  id: number;
  orderNo: string;
  userId: number;
  totalAmount: number;
  finalAmount: number;
  status: number;
  items: OrderItemVO[];
  createdAt: string;
  updatedAt: string;
}

class OrderItemVO {
  id: number;
  orderId: number;
  productId: number;
  productName: string;
  quantity: number;
  price: number;
  specId: number;
  specName: string;
}

class OrderTrackResponse {
  orderNo: string;
  status: number;
  trackPoints: OrderTrackPointVO[];
}

class OrderTrackPointVO {
  time: string;
  description: string;
  location: string;
}

class RefundVO {
  id: number;
  refundNo: string;
  orderId: number;
  amount: number;
  status: number;
  createdAt: string;
}

class OrderStatisticsVO {
  totalOrders: number;
  totalAmount: number;
  pendingOrders: number;
  completedOrders: number;
  cancelledOrders: number;
}

class CategoryVO {
  id: number;
  name: string;
  parentId: number;
  sort: number;
  icon: string;
  status: number;
}

class CategoryTreeVO {
  id: number;
  name: string;
  parentId: number;
  sort: number;
  icon: string;
  status: number;
  children: CategoryTreeVO[];
}

class PaymentVO {
  id: number;
  paymentNo: string;
  orderId: number;
  amount: number;
  method: string;
  status: number;
  paidAt: string;
}

class PaymentRecordVO {
  id: number;
  paymentId: number;
  amount: number;
  type: string;
  status: number;
  createdAt: string;
}

class PaymentChannelVO {
  code: string;
  name: string;
  icon: string;
  enabled: boolean;
}

class PaymentStatisticsVO {
  totalAmount: number;
  totalRefund: number;
  successCount: number;
  failCount: number;
}

class ShippingVO {
  id: number;
  shippingNo: string;
  orderId: number;
  carrier: string;
  trackingNo: string;
  status: number;
  shippedAt: string;
  deliveredAt: string;
}

class ShippingTrackResponse {
  shippingNo: string;
  carrier: string;
  trackingNo: string;
  status: number;
  trackPoints: ShippingTrackPointVO[];
}

class ShippingTrackPointVO {
  time: string;
  description: string;
  location: string;
}

class CarrierVO {
  code: string;
  name: string;
  logo: string;
  enabled: boolean;
}

class ShippingRateVO {
  carrier: string;
  service: string;
  rate: number;
  estimatedDays: number;
}

class InventoryVO {
  id: number;
  productId: number;
  warehouseId: number;
  quantity: number;
  locked: number;
  available: number;
}

class InventoryMovementVO {
  id: number;
  inventoryId: number;
  type: string;
  quantity: number;
  reason: string;
  createdAt: string;
}

class InventoryAlertVO {
  productId: number;
  productName: string;
  warehouseId: number;
  currentStock: number;
  alertThreshold: number;
}

class WarehouseVO {
  id: number;
  name: string;
  address: string;
  enabled: boolean;
}

class ReviewVO {
  id: number;
  productId: number;
  userId: number;
  username: string;
  rating: number;
  content: string;
  images: string[];
  status: number;
  createdAt: string;
}

class ReviewCommentVO {
  id: number;
  reviewId: number;
  userId: number;
  username: string;
  content: string;
  createdAt: string;
}

class NotificationVO {
  id: number;
  userId: number;
  type: string;
  title: string;
  content: string;
  read: boolean;
  createdAt: string;
}

class NotificationTemplateVO {
  id: number;
  code: string;
  type: string;
  title: string;
  content: string;
}

class AnalyticsOverviewVO {
  totalRevenue: number;
  totalOrders: number;
  totalUsers: number;
  totalProducts: number;
  todayRevenue: number;
  todayOrders: number;
  newUsers: number;
  conversionRate: number;
}

class SalesStatisticsVO {
  totalSales: number;
  totalRefund: number;
  totalOrders: number;
  averageOrderValue: number;
  dailySales: DailySalesVO[];
}

class DailySalesVO {
  date: string;
  sales: number;
  orders: number;
}

class UserAnalyticsVO {
  totalUsers: number;
  newUsers: number;
  activeUsers: number;
  retentionRate: number;
}

class ProductAnalyticsVO {
  topProducts: ProductVO[];
  categoryDistribution: CategoryDistributionVO[];
}

class CategoryDistributionVO {
  categoryId: number;
  categoryName: string;
  count: number;
  percentage: number;
}

class OrderAnalyticsVO {
  totalOrders: number;
  completedOrders: number;
  cancelledOrders: number;
  averageOrderValue: number;
}

class TrendAnalyticsVO {
  date: string;
  value: number;
  label: string;
}

class DashboardVO {
  overview: AnalyticsOverviewVO;
  sales: SalesStatisticsVO;
  users: UserAnalyticsVO;
  products: ProductAnalyticsVO;
  orders: OrderAnalyticsVO;
}
