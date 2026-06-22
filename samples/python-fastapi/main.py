from fastapi import FastAPI, Query, Path, Body
from pydantic import BaseModel
from typing import Optional, List, Dict, Generic, TypeVar
from datetime import datetime

app = FastAPI()

T = TypeVar('T')


# ===== Base Models =====

class BaseEntity(BaseModel):
    id: int
    created_at: datetime
    updated_at: datetime


class BaseVO(BaseModel):
    code: int = 200
    message: str = "success"


class PageResult(BaseModel, Generic[T]):
    list: List[T]
    total: int
    page_num: int
    page_size: int


class Result(BaseModel, Generic[T]):
    code: int = 200
    message: str = "success"
    data: Optional[T] = None


# ===== User DTOs =====

class Address(BaseModel):
    id: int
    user_id: int
    receiver: str
    phone: str
    province: str
    city: str
    district: str
    detail: str
    is_default: bool


class CreateUserReq(BaseModel):
    username: str
    email: str
    phone: str
    password: str
    avatar: str
    status: int
    tags: List[str]


class UpdateUserReq(BaseModel):
    username: Optional[str] = None
    email: Optional[str] = None
    phone: Optional[str] = None
    avatar: Optional[str] = None
    status: Optional[int] = None
    tags: Optional[List[str]] = None


class LoginReq(BaseModel):
    username: str
    password: str
    captcha: str


class UpdateProfileReq(BaseModel):
    nickname: str
    bio: str
    gender: int
    birthday: str
    avatar: str


class AddressReq(BaseModel):
    receiver: str
    phone: str
    province: str
    city: str
    district: str
    detail: str
    is_default: bool


class UserVO(BaseEntity):
    username: str
    email: str
    phone: str
    avatar: str
    status: int
    tags: List[str]


class LoginResponse(BaseModel):
    token: str
    user_id: int
    username: str
    expires_in: int


class UserProfileVO(BaseModel):
    id: int
    username: str
    email: str
    phone: str
    avatar: str
    nickname: str
    bio: str
    gender: int
    birthday: str


# ===== Product DTOs =====

class CreateProductReq(BaseModel):
    name: str
    description: str
    price: float
    original_price: float
    stock: int
    category_id: int
    main_image: str
    images: List[str]
    status: int


class ProductSpecReq(BaseModel):
    name: str
    value: str
    price: float
    stock: int


class ProductVO(BaseEntity):
    name: str
    description: str
    price: float
    original_price: float
    stock: int
    category_id: int
    main_image: str
    status: int
    sales_count: int


class ProductSpecVO(BaseModel):
    id: int
    name: str
    value: str
    price: float
    stock: int


class ProductDetailVO(BaseModel):
    id: int
    name: str
    description: str
    price: float
    original_price: float
    category_id: int
    category_name: str
    stock: int
    sales: int
    main_image: str
    images: List[str]
    specs: List[ProductSpecVO]
    average_rating: float
    review_count: int


class ProductStatisticsVO(BaseModel):
    product_id: int
    views: int
    sales: int
    favorites: int
    average_rating: float
    review_count: int


# ===== Order DTOs =====

class OrderItemReq(BaseModel):
    product_id: int
    quantity: int
    spec_id: Optional[int] = None


class CreateOrderReq(BaseModel):
    user_id: int
    address_id: int
    items: List[OrderItemReq]
    remark: str
    coupon_code: str


class CancelOrderReq(BaseModel):
    reason: str


class RefundReq(BaseModel):
    amount: float
    reason: str


class OrderItemVO(BaseModel):
    id: int
    order_id: int
    product_id: int
    product_name: str
    quantity: int
    price: float
    spec_id: int
    spec_name: str


class OrderVO(BaseEntity):
    order_no: str
    user_id: int
    total_amount: float
    final_amount: float
    status: int
    items: List[OrderItemVO]


class OrderTrackPointVO(BaseModel):
    time: str
    description: str
    location: str


class OrderTrackResponse(BaseModel):
    order_no: str
    status: int
    track_points: List[OrderTrackPointVO]


class RefundVO(BaseModel):
    id: int
    refund_no: str
    order_id: int
    amount: float
    status: int
    created_at: str


class OrderStatisticsVO(BaseModel):
    total_orders: int
    total_amount: float
    pending_orders: int
    completed_orders: int
    cancelled_orders: int


# ===== Category DTOs =====

class CreateCategoryReq(BaseModel):
    name: str
    parent_id: int
    sort: int
    icon: str
    status: int


class SortReq(BaseModel):
    sort: int


class CategoryVO(BaseEntity):
    name: str
    parent_id: int
    sort: int
    icon: str
    status: int


class CategoryTreeVO(BaseModel):
    id: int
    name: str
    parent_id: int
    sort: int
    icon: str
    status: int
    children: List['CategoryTreeVO']


# ===== Payment DTOs =====

class CreatePaymentReq(BaseModel):
    order_id: int
    amount: float
    method: str
    channel: str


class PaymentVO(BaseEntity):
    payment_no: str
    order_id: int
    amount: float
    method: str
    status: int
    paid_at: Optional[str] = None


class PaymentRecordVO(BaseModel):
    id: int
    payment_id: int
    amount: float
    type: str
    status: int
    created_at: str


class PaymentChannelVO(BaseModel):
    code: str
    name: str
    icon: str
    enabled: bool


class PaymentStatisticsVO(BaseModel):
    total_amount: float
    total_refund: float
    success_count: int
    fail_count: int


# ===== Shipping DTOs =====

class CreateShippingReq(BaseModel):
    order_id: int
    carrier: str
    tracking_no: str


class ShippingRateReq(BaseModel):
    from_addr: str
    to_addr: str
    weight: float
    volume: float


class ShippingVO(BaseEntity):
    shipping_no: str
    order_id: int
    carrier: str
    tracking_no: str
    status: int
    shipped_at: Optional[str] = None
    delivered_at: Optional[str] = None


class ShippingTrackPointVO(BaseModel):
    time: str
    description: str
    location: str


class ShippingTrackResponse(BaseModel):
    shipping_no: str
    carrier: str
    tracking_no: str
    status: int
    track_points: List[ShippingTrackPointVO]


class CarrierVO(BaseModel):
    code: str
    name: str
    logo: str
    enabled: bool


class ShippingRateVO(BaseModel):
    carrier: str
    service: str
    rate: float
    estimated_days: int


# ===== Inventory DTOs =====

class CreateInventoryReq(BaseModel):
    product_id: int
    warehouse_id: int
    quantity: int


class UpdateStockReq(BaseModel):
    quantity: int
    reason: str


class InventoryVO(BaseEntity):
    product_id: int
    warehouse_id: int
    quantity: int
    locked: int
    available: int


class InventoryMovementVO(BaseModel):
    id: int
    inventory_id: int
    type: str
    quantity: int
    reason: str
    created_at: str


class InventoryAlertVO(BaseModel):
    product_id: int
    product_name: str
    warehouse_id: int
    current_stock: int
    alert_threshold: int


class WarehouseVO(BaseModel):
    id: int
    name: str
    address: str
    enabled: bool


# ===== Review DTOs =====

class CreateReviewReq(BaseModel):
    product_id: int
    rating: int
    content: str
    images: List[str]


class ReviewCommentReq(BaseModel):
    user_id: int
    content: str


class ModerateReq(BaseModel):
    status: int
    reason: str


class ReviewVO(BaseEntity):
    product_id: int
    user_id: int
    username: str
    rating: int
    content: str
    images: List[str]
    status: int


class ReviewCommentVO(BaseModel):
    id: int
    review_id: int
    user_id: int
    username: str
    content: str
    created_at: str


# ===== Notification DTOs =====

class CreateNotificationReq(BaseModel):
    user_id: int
    type: str
    title: str
    content: str


class SendNotificationReq(BaseModel):
    user_ids: List[int]
    type: str
    title: str
    content: str


class NotificationVO(BaseEntity):
    user_id: int
    type: str
    title: str
    content: str
    read: bool


class NotificationTemplateVO(BaseModel):
    id: int
    code: str
    type: str
    title: str
    content: str


# ===== Analytics DTOs =====

class AnalyticsOverviewVO(BaseModel):
    total_revenue: float
    total_orders: int
    total_users: int
    total_products: int
    today_revenue: float
    today_orders: int
    new_users: int
    conversion_rate: float


class DailySalesVO(BaseModel):
    date: str
    sales: float
    orders: int


class SalesStatisticsVO(BaseModel):
    total_sales: float
    total_refund: float
    total_orders: int
    average_order_value: float
    daily_sales: List[DailySalesVO]


class UserAnalyticsVO(BaseModel):
    total_users: int
    new_users: int
    active_users: int
    retention_rate: float


class CategoryDistributionVO(BaseModel):
    category_id: int
    category_name: str
    count: int
    percentage: float


class ProductAnalyticsVO(BaseModel):
    top_products: List[ProductVO]
    category_distribution: List[CategoryDistributionVO]


class OrderAnalyticsVO(BaseModel):
    total_orders: int
    completed_orders: int
    cancelled_orders: int
    average_order_value: float


class TrendAnalyticsVO(BaseModel):
    date: str
    value: float
    label: str


class DashboardVO(BaseModel):
    overview: AnalyticsOverviewVO
    sales: SalesStatisticsVO
    users: UserAnalyticsVO
    products: ProductAnalyticsVO
    orders: OrderAnalyticsVO


# ===== User Routes =====

@app.get("/users", response_model=PageResult[UserVO])
def list_users(name: Optional[str] = None, role: str = "user"):
    """listUsers returns all users."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/users", response_model=Result[UserVO])
def create_user(req: CreateUserReq):
    """createUser creates a new user."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/users/{user_id}", response_model=Result[UserVO])
def get_user(user_id: int = Path(...)):
    """getUser returns a single user by ID."""
    return {"code": 200, "message": "success", "data": {"id": user_id}}


@app.put("/users/{user_id}", response_model=Result[UserVO])
def update_user(user_id: int = Path(...), req: UpdateUserReq = Body(...)):
    """updateUser updates an existing user."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/users/{user_id}", response_model=Result)
def delete_user(user_id: int = Path(...)):
    """deleteUser removes a user by ID."""
    return {"code": 200, "message": "success"}


@app.patch("/users/{user_id}", response_model=Result[UserVO])
def patch_user(user_id: int = Path(...), name: str = "unknown"):
    """patchUser partially updates a user."""
    return {"code": 200, "message": "success", "data": {"id": user_id}}


@app.post("/users/login", response_model=Result[LoginResponse])
def user_login(req: LoginReq):
    """userLogin handles user login."""
    return {"code": 200, "message": "success", "data": {"token": "xxx", "user_id": 1, "username": "test", "expires_in": 3600}}


@app.post("/users/register", response_model=Result[UserVO])
def user_register(req: CreateUserReq):
    """userRegister handles user registration."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/users/{user_id}/profile", response_model=Result[UserProfileVO])
def get_user_profile(user_id: int = Path(...)):
    """getUserProfile returns a user's profile."""
    return {"code": 200, "message": "success", "data": {"id": user_id}}


@app.put("/users/{user_id}/profile", response_model=Result[UserProfileVO])
def update_user_profile(user_id: int = Path(...), req: UpdateProfileReq = Body(...)):
    """updateUserProfile updates a user's profile."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/users/{user_id}/addresses", response_model=Result[List[Address]])
def get_user_addresses(user_id: int = Path(...)):
    """getUserAddresses returns a user's addresses."""
    return {"code": 200, "message": "success", "data": []}


@app.post("/users/{user_id}/addresses", response_model=Result[Address])
def add_user_address(user_id: int = Path(...), req: AddressReq = Body(...)):
    """addUserAddress adds a new address for a user."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/users/{user_id}/favorites", response_model=PageResult[ProductVO])
def get_user_favorites(user_id: int = Path(...)):
    """getUserFavorites returns a user's favorite products."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


# ===== Product Routes =====

@app.get("/api/products", response_model=PageResult[ProductVO])
def list_products(category_id: Optional[int] = None, status: Optional[int] = None):
    """listProducts returns a paginated list of products."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/products", response_model=Result[ProductVO])
def create_product(req: CreateProductReq):
    """createProduct creates a new product."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/products/{product_id}", response_model=Result[ProductVO])
def get_product(product_id: int = Path(...)):
    """getProduct returns a single product by ID."""
    return {"code": 200, "message": "success", "data": {"id": product_id}}


@app.put("/api/products/{product_id}", response_model=Result[ProductVO])
def update_product(product_id: int = Path(...), req: CreateProductReq = Body(...)):
    """updateProduct updates an existing product."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/products/{product_id}", response_model=Result)
def delete_product(product_id: int = Path(...)):
    """deleteProduct removes a product by ID."""
    return {"code": 200, "message": "success"}


@app.get("/api/products/search", response_model=PageResult[ProductVO])
def search_products(keyword: str = Query(...), category_id: Optional[int] = None):
    """searchProducts searches for products."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.get("/api/products/{product_id}/detail", response_model=Result[ProductDetailVO])
def get_product_detail(product_id: int = Path(...)):
    """getProductDetail returns detailed product information."""
    return {"code": 200, "message": "success", "data": {"id": product_id}}


@app.get("/api/products/category/{category_id}", response_model=PageResult[ProductVO])
def get_products_by_category(category_id: int = Path(...)):
    """getProductsByCategory returns products in a category."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.get("/api/products/hot", response_model=Result[List[ProductVO]])
def get_hot_products():
    """getHotProducts returns hot products."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/products/new", response_model=Result[List[ProductVO]])
def get_new_products():
    """getNewProducts returns new products."""
    return {"code": 200, "message": "success", "data": []}


@app.post("/api/products/{product_id}/specs", response_model=Result[ProductSpecVO])
def add_product_spec(product_id: int = Path(...), req: ProductSpecReq = Body(...)):
    """addProductSpec adds a product specification."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/products/{product_id}/specs", response_model=Result[List[ProductSpecVO]])
def get_product_specs(product_id: int = Path(...)):
    """getProductSpecs returns product specifications."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/products/{product_id}/statistics", response_model=Result[ProductStatisticsVO])
def get_product_statistics(product_id: int = Path(...)):
    """getProductStatistics returns product statistics."""
    return {"code": 200, "message": "success", "data": {"product_id": product_id}}


# ===== Order Routes =====

@app.get("/api/orders", response_model=PageResult[OrderVO])
def list_orders(status: Optional[int] = None, user_id: Optional[int] = None):
    """listOrders returns a paginated list of orders."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/orders", response_model=Result[OrderVO])
def create_order(req: CreateOrderReq):
    """createOrder creates a new order."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/orders/{order_id}", response_model=Result[OrderVO])
def get_order(order_id: int = Path(...)):
    """getOrder returns a single order by ID."""
    return {"code": 200, "message": "success", "data": {"id": order_id}}


@app.put("/api/orders/{order_id}", response_model=Result[OrderVO])
def update_order(order_id: int = Path(...), req: CreateOrderReq = Body(...)):
    """updateOrder updates an existing order."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/orders/{order_id}", response_model=Result)
def delete_order(order_id: int = Path(...)):
    """deleteOrder removes an order by ID."""
    return {"code": 200, "message": "success"}


@app.post("/api/orders/{order_id}/cancel", response_model=Result[OrderVO])
def cancel_order(order_id: int = Path(...), req: CancelOrderReq = Body(...)):
    """cancelOrder cancels an order."""
    return {"code": 200, "message": "success", "data": {"id": order_id}}


@app.get("/api/orders/{order_id}/track", response_model=Result[OrderTrackResponse])
def track_order(order_id: int = Path(...)):
    """trackOrder returns order tracking information."""
    return {"code": 200, "message": "success", "data": {"order_no": str(order_id)}}


@app.post("/api/orders/{order_id}/items", response_model=Result[OrderItemVO])
def add_order_item(order_id: int = Path(...), req: OrderItemReq = Body(...)):
    """addOrderItem adds an item to an order."""
    return {"code": 200, "message": "success", "data": req}


@app.post("/api/orders/{order_id}/refund", response_model=Result[RefundVO])
def refund_order(order_id: int = Path(...), req: RefundReq = Body(...)):
    """refundOrder requests a refund for an order."""
    return {"code": 200, "message": "success", "data": {"id": 1, "order_id": order_id}}


@app.get("/api/orders/statistics", response_model=Result[OrderStatisticsVO])
def get_order_statistics():
    """getOrderStatistics returns order statistics."""
    return {"code": 200, "message": "success", "data": {}}


@app.get("/api/orders/user/{user_id}", response_model=PageResult[OrderVO])
def get_orders_by_user(user_id: int = Path(...)):
    """getOrdersByUser returns orders for a user."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


# ===== Category Routes =====

@app.get("/api/categories", response_model=Result[List[CategoryVO]])
def list_categories():
    """listCategories returns all categories."""
    return {"code": 200, "message": "success", "data": []}


@app.post("/api/categories", response_model=Result[CategoryVO])
def create_category(req: CreateCategoryReq):
    """createCategory creates a new category."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/categories/{category_id}", response_model=Result[CategoryVO])
def get_category(category_id: int = Path(...)):
    """getCategory returns a single category by ID."""
    return {"code": 200, "message": "success", "data": {"id": category_id}}


@app.put("/api/categories/{category_id}", response_model=Result[CategoryVO])
def update_category(category_id: int = Path(...), req: CreateCategoryReq = Body(...)):
    """updateCategory updates an existing category."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/categories/{category_id}", response_model=Result)
def delete_category(category_id: int = Path(...)):
    """deleteCategory removes a category by ID."""
    return {"code": 200, "message": "success"}


@app.get("/api/categories/tree", response_model=Result[List[CategoryTreeVO]])
def get_category_tree():
    """getCategoryTree returns the category tree."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/categories/{category_id}/products", response_model=PageResult[ProductVO])
def get_category_products(category_id: int = Path(...)):
    """getCategoryProducts returns products in a category."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.put("/api/categories/{category_id}/sort", response_model=Result[CategoryVO])
def update_category_sort(category_id: int = Path(...), req: SortReq = Body(...)):
    """updateCategorySort updates category sort order."""
    return {"code": 200, "message": "success", "data": {"id": category_id}}


@app.get("/api/categories/roots", response_model=Result[List[CategoryVO]])
def get_root_categories():
    """getRootCategories returns root categories."""
    return {"code": 200, "message": "success", "data": []}


# ===== Payment Routes =====

@app.get("/api/payments", response_model=PageResult[PaymentVO])
def list_payments(status: Optional[int] = None):
    """listPayments returns a paginated list of payments."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/payments", response_model=Result[PaymentVO])
def create_payment(req: CreatePaymentReq):
    """createPayment creates a new payment."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/payments/{payment_id}", response_model=Result[PaymentVO])
def get_payment(payment_id: int = Path(...)):
    """getPayment returns a single payment by ID."""
    return {"code": 200, "message": "success", "data": {"id": payment_id}}


@app.put("/api/payments/{payment_id}", response_model=Result[PaymentVO])
def update_payment(payment_id: int = Path(...), req: CreatePaymentReq = Body(...)):
    """updatePayment updates an existing payment."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/payments/{payment_id}", response_model=Result)
def delete_payment(payment_id: int = Path(...)):
    """deletePayment removes a payment by ID."""
    return {"code": 200, "message": "success"}


@app.post("/api/payments/{payment_id}/refund", response_model=Result[RefundVO])
def refund_payment(payment_id: int = Path(...), req: RefundReq = Body(...)):
    """refundPayment refunds a payment."""
    return {"code": 200, "message": "success", "data": {"id": 1, "order_id": payment_id}}


@app.get("/api/payments/{payment_id}/records", response_model=Result[List[PaymentRecordVO]])
def get_payment_records(payment_id: int = Path(...)):
    """getPaymentRecords returns payment records."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/payments/channels", response_model=Result[List[PaymentChannelVO]])
def get_payment_channels():
    """getPaymentChannels returns available payment channels."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/payments/statistics", response_model=Result[PaymentStatisticsVO])
def get_payment_statistics():
    """getPaymentStatistics returns payment statistics."""
    return {"code": 200, "message": "success", "data": {}}


@app.post("/api/payments/{payment_id}/confirm", response_model=Result[PaymentVO])
def confirm_payment(payment_id: int = Path(...)):
    """confirmPayment confirms a payment."""
    return {"code": 200, "message": "success", "data": {"id": payment_id}}


# ===== Shipping Routes =====

@app.get("/api/shipping", response_model=PageResult[ShippingVO])
def list_shipping(status: Optional[int] = None):
    """listShipping returns a paginated list of shipping records."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/shipping", response_model=Result[ShippingVO])
def create_shipping(req: CreateShippingReq):
    """createShipping creates a new shipping record."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/shipping/{shipping_id}", response_model=Result[ShippingVO])
def get_shipping(shipping_id: int = Path(...)):
    """getShipping returns a single shipping record by ID."""
    return {"code": 200, "message": "success", "data": {"id": shipping_id}}


@app.put("/api/shipping/{shipping_id}", response_model=Result[ShippingVO])
def update_shipping(shipping_id: int = Path(...), req: CreateShippingReq = Body(...)):
    """updateShipping updates an existing shipping record."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/shipping/{shipping_id}", response_model=Result)
def delete_shipping(shipping_id: int = Path(...)):
    """deleteShipping removes a shipping record by ID."""
    return {"code": 200, "message": "success"}


@app.get("/api/shipping/{shipping_id}/track", response_model=Result[ShippingTrackResponse])
def track_shipping(shipping_id: int = Path(...)):
    """trackShipping returns shipping tracking information."""
    return {"code": 200, "message": "success", "data": {"shipping_no": str(shipping_id)}}


@app.get("/api/shipping/carriers", response_model=Result[List[CarrierVO]])
def get_carriers():
    """getCarriers returns available carriers."""
    return {"code": 200, "message": "success", "data": []}


@app.post("/api/shipping/rates", response_model=Result[List[ShippingRateVO]])
def calculate_shipping_rates(req: ShippingRateReq):
    """calculateShippingRates calculates shipping rates."""
    return {"code": 200, "message": "success", "data": []}


@app.post("/api/shipping/{shipping_id}/ship", response_model=Result[ShippingVO])
def ship_order(shipping_id: int = Path(...)):
    """shipOrder ships an order."""
    return {"code": 200, "message": "success", "data": {"id": shipping_id}}


@app.post("/api/shipping/{shipping_id}/deliver", response_model=Result[ShippingVO])
def deliver_order(shipping_id: int = Path(...)):
    """deliverOrder delivers an order."""
    return {"code": 200, "message": "success", "data": {"id": shipping_id}}


# ===== Inventory Routes =====

@app.get("/api/inventory", response_model=PageResult[InventoryVO])
def list_inventory(product_id: Optional[int] = None):
    """listInventory returns a paginated list of inventory."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/inventory", response_model=Result[InventoryVO])
def create_inventory(req: CreateInventoryReq):
    """createInventory creates a new inventory record."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/inventory/{inventory_id}", response_model=Result[InventoryVO])
def get_inventory(inventory_id: int = Path(...)):
    """getInventory returns a single inventory record by ID."""
    return {"code": 200, "message": "success", "data": {"id": inventory_id}}


@app.put("/api/inventory/{inventory_id}", response_model=Result[InventoryVO])
def update_inventory(inventory_id: int = Path(...), req: CreateInventoryReq = Body(...)):
    """updateInventory updates an existing inventory record."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/inventory/{inventory_id}", response_model=Result)
def delete_inventory(inventory_id: int = Path(...)):
    """deleteInventory removes an inventory record by ID."""
    return {"code": 200, "message": "success"}


@app.put("/api/inventory/{inventory_id}/stock", response_model=Result[InventoryVO])
def update_stock(inventory_id: int = Path(...), req: UpdateStockReq = Body(...)):
    """updateStock updates inventory stock."""
    return {"code": 200, "message": "success", "data": {"id": inventory_id}}


@app.get("/api/inventory/{inventory_id}/movements", response_model=Result[List[InventoryMovementVO]])
def get_inventory_movements(inventory_id: int = Path(...)):
    """getInventoryMovements returns inventory movement records."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/inventory/alerts", response_model=Result[List[InventoryAlertVO]])
def get_inventory_alerts():
    """getInventoryAlerts returns low stock alerts."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/inventory/warehouses", response_model=Result[List[WarehouseVO]])
def get_warehouses():
    """getWarehouses returns available warehouses."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/inventory/product/{product_id}", response_model=Result[List[InventoryVO]])
def get_inventory_by_product(product_id: int = Path(...)):
    """getInventoryByProduct returns inventory for a product."""
    return {"code": 200, "message": "success", "data": []}


# ===== Review Routes =====

@app.get("/api/reviews", response_model=PageResult[ReviewVO])
def list_reviews(product_id: Optional[int] = None, rating: Optional[int] = None):
    """listReviews returns a paginated list of reviews."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/reviews", response_model=Result[ReviewVO])
def create_review(req: CreateReviewReq):
    """createReview creates a new review."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/reviews/{review_id}", response_model=Result[ReviewVO])
def get_review(review_id: int = Path(...)):
    """getReview returns a single review by ID."""
    return {"code": 200, "message": "success", "data": {"id": review_id}}


@app.put("/api/reviews/{review_id}", response_model=Result[ReviewVO])
def update_review(review_id: int = Path(...), req: CreateReviewReq = Body(...)):
    """updateReview updates an existing review."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/reviews/{review_id}", response_model=Result)
def delete_review(review_id: int = Path(...)):
    """deleteReview removes a review by ID."""
    return {"code": 200, "message": "success"}


@app.get("/api/reviews/product/{product_id}", response_model=PageResult[ReviewVO])
def get_reviews_by_product(product_id: int = Path(...)):
    """getReviewsByProduct returns reviews for a product."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/reviews/{review_id}/comments", response_model=Result[ReviewCommentVO])
def add_review_comment(review_id: int = Path(...), req: ReviewCommentReq = Body(...)):
    """addReviewComment adds a comment to a review."""
    return {"code": 200, "message": "success", "data": {"id": 1, "review_id": review_id}}


@app.get("/api/reviews/{review_id}/comments", response_model=Result[List[ReviewCommentVO]])
def get_review_comments(review_id: int = Path(...)):
    """getReviewComments returns comments for a review."""
    return {"code": 200, "message": "success", "data": []}


@app.put("/api/reviews/{review_id}/moderate", response_model=Result[ReviewVO])
def moderate_review(review_id: int = Path(...), req: ModerateReq = Body(...)):
    """moderateReview moderates a review."""
    return {"code": 200, "message": "success", "data": {"id": review_id}}


@app.get("/api/reviews/user/{user_id}", response_model=PageResult[ReviewVO])
def get_reviews_by_user(user_id: int = Path(...)):
    """getReviewsByUser returns reviews by a user."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


# ===== Notification Routes =====

@app.get("/api/notifications", response_model=PageResult[NotificationVO])
def list_notifications(user_id: Optional[int] = None, read: Optional[bool] = None):
    """listNotifications returns a paginated list of notifications."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.post("/api/notifications", response_model=Result[NotificationVO])
def create_notification(req: CreateNotificationReq):
    """createNotification creates a new notification."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/notifications/{notification_id}", response_model=Result[NotificationVO])
def get_notification(notification_id: int = Path(...)):
    """getNotification returns a single notification by ID."""
    return {"code": 200, "message": "success", "data": {"id": notification_id}}


@app.put("/api/notifications/{notification_id}", response_model=Result[NotificationVO])
def update_notification(notification_id: int = Path(...), req: CreateNotificationReq = Body(...)):
    """updateNotification updates an existing notification."""
    return {"code": 200, "message": "success", "data": req}


@app.delete("/api/notifications/{notification_id}", response_model=Result)
def delete_notification(notification_id: int = Path(...)):
    """deleteNotification removes a notification by ID."""
    return {"code": 200, "message": "success"}


@app.get("/api/notifications/user/{user_id}", response_model=PageResult[NotificationVO])
def get_notifications_by_user(user_id: int = Path(...)):
    """getNotificationsByUser returns notifications for a user."""
    return {"list": [], "total": 0, "page_num": 1, "page_size": 20}


@app.put("/api/notifications/{notification_id}/read", response_model=Result)
def mark_notification_as_read(notification_id: int = Path(...)):
    """markNotificationAsRead marks a notification as read."""
    return {"code": 200, "message": "success"}


@app.put("/api/notifications/user/{user_id}/read-all", response_model=Result)
def mark_all_notifications_as_read(user_id: int = Path(...)):
    """markAllNotificationsAsRead marks all notifications as read for a user."""
    return {"code": 200, "message": "success"}


@app.post("/api/notifications/send", response_model=Result[NotificationVO])
def send_notification(req: SendNotificationReq):
    """sendNotification sends a notification."""
    return {"code": 200, "message": "success", "data": req}


@app.get("/api/notifications/templates", response_model=Result[List[NotificationTemplateVO]])
def get_notification_templates():
    """getNotificationTemplates returns notification templates."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/notifications/user/{user_id}/unread-count", response_model=Result[int])
def get_unread_notification_count(user_id: int = Path(...)):
    """getUnreadNotificationCount returns unread notification count."""
    return {"code": 200, "message": "success", "data": 0}


# ===== Analytics Routes =====

@app.get("/api/analytics/overview", response_model=Result[AnalyticsOverviewVO])
def get_analytics_overview():
    """getAnalyticsOverview returns analytics overview."""
    return {"code": 200, "message": "success", "data": {}}


@app.get("/api/analytics/sales", response_model=Result[SalesStatisticsVO])
def get_sales_statistics(start_date: str = Query(...), end_date: str = Query(...)):
    """getSalesStatistics returns sales statistics."""
    return {"code": 200, "message": "success", "data": {}}


@app.get("/api/analytics/users", response_model=Result[UserAnalyticsVO])
def get_user_analytics(start_date: str = Query(...), end_date: str = Query(...)):
    """getUserAnalytics returns user analytics."""
    return {"code": 200, "message": "success", "data": {}}


@app.get("/api/analytics/products", response_model=Result[ProductAnalyticsVO])
def get_product_analytics(start_date: str = Query(...), end_date: str = Query(...)):
    """getProductAnalytics returns product analytics."""
    return {"code": 200, "message": "success", "data": {}}


@app.get("/api/analytics/orders", response_model=Result[OrderAnalyticsVO])
def get_order_analytics(start_date: str = Query(...), end_date: str = Query(...)):
    """getOrderAnalytics returns order analytics."""
    return {"code": 200, "message": "success", "data": {}}


@app.get("/api/analytics/trends", response_model=Result[List[TrendAnalyticsVO]])
def get_trend_analytics(start_date: str = Query(...), end_date: str = Query(...)):
    """getTrendAnalytics returns trend analytics."""
    return {"code": 200, "message": "success", "data": []}


@app.get("/api/analytics/dashboard", response_model=Result[DashboardVO])
def get_dashboard():
    """getDashboard returns the analytics dashboard."""
    return {"code": 200, "message": "success", "data": {}}
