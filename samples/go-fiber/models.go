package main

import "time"

// BaseModel is the base struct for all models
type BaseModel struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents a user
type User struct {
	BaseModel
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	Status   int    `json:"status"`
}

// Product represents a product
type Product struct {
	BaseModel
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price"`
	Stock         int     `json:"stock"`
	CategoryID    int64   `json:"category_id"`
	MainImage     string  `json:"main_image"`
	Status        int     `json:"status"`
	SalesCount    int     `json:"sales_count"`
}

// Order represents an order
type Order struct {
	BaseModel
	OrderNo         string  `json:"order_no"`
	UserID          int64   `json:"user_id"`
	TotalAmount     float64 `json:"total_amount"`
	DiscountAmount  float64 `json:"discount_amount"`
	FinalAmount     float64 `json:"final_amount"`
	Status          int     `json:"status"`
	PaymentMethod   string  `json:"payment_method"`
	ShippingAddress string  `json:"shipping_address"`
	ReceiverName    string  `json:"receiver_name"`
	ReceiverPhone   string  `json:"receiver_phone"`
}

// Category represents a category
type Category struct {
	BaseModel
	Name     string `json:"name"`
	ParentID int64  `json:"parent_id"`
	Sort     int    `json:"sort"`
	Icon     string `json:"icon"`
	Status   int    `json:"status"`
}

// Payment represents a payment
type Payment struct {
	BaseModel
	PaymentNo string    `json:"payment_no"`
	OrderID   int64     `json:"order_id"`
	Amount    float64   `json:"amount"`
	Method    string    `json:"method"`
	Status    int       `json:"status"`
	PaidAt    time.Time `json:"paid_at"`
}

// Shipping represents a shipping
type Shipping struct {
	BaseModel
	ShippingNo  string    `json:"shipping_no"`
	OrderID     int64     `json:"order_id"`
	Carrier     string    `json:"carrier"`
	TrackingNo  string    `json:"tracking_no"`
	Status      int       `json:"status"`
	ShippedAt   time.Time `json:"shipped_at"`
	DeliveredAt time.Time `json:"delivered_at"`
}

// Inventory represents an inventory
type Inventory struct {
	BaseModel
	ProductID   int64 `json:"product_id"`
	WarehouseID int64 `json:"warehouse_id"`
	Quantity    int   `json:"quantity"`
	Locked      int   `json:"locked"`
	Available   int   `json:"available"`
}

// Review represents a review
type Review struct {
	BaseModel
	ProductID int64  `json:"product_id"`
	UserID    int64  `json:"user_id"`
	Rating    int    `json:"rating"`
	Content   string `json:"content"`
	Images    string `json:"images"`
	Status    int    `json:"status"`
}

// Notification represents a notification
type Notification struct {
	BaseModel
	UserID  int64  `json:"user_id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Read    bool   `json:"read"`
}

// Result is the standard API response
type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageResult is the paginated response
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	PageNum  int         `json:"page_num"`
	PageSize int         `json:"page_size"`
}

// ===== User DTOs =====

type CreateUserReq struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

type UpdateUserReq struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type UserVO struct {
	BaseModel
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Phone  string   `json:"phone"`
	Avatar string   `json:"avatar"`
	Status int      `json:"status"`
	Tags   []string `json:"tags"`
}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResp struct {
	Token    string `json:"token"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

type RegisterReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

type UserProfileVO struct {
	BaseModel
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
	Bio      string `json:"bio"`
	Gender   int    `json:"gender"`
	Birthday string `json:"birthday"`
}

type AddressVO struct {
	ID        int64  `json:"id"`
	Receiver  string `json:"receiver"`
	Phone     string `json:"phone"`
	Province  string `json:"province"`
	City      string `json:"city"`
	District  string `json:"district"`
	Detail    string `json:"detail"`
	IsDefault bool   `json:"is_default"`
}

// ===== Product DTOs =====

type CreateProductReq struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price"`
	CategoryID    int64   `json:"category_id"`
	Stock         int     `json:"stock"`
	MainImage     string  `json:"main_image"`
	Unit          string  `json:"unit"`
}

type ProductDetailVO struct {
	BaseModel
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Price         float64         `json:"price"`
	OriginalPrice float64         `json:"original_price"`
	CategoryID    int64           `json:"category_id"`
	CategoryName  string          `json:"category_name"`
	Stock         int             `json:"stock"`
	Sales         int             `json:"sales"`
	MainImage     string          `json:"main_image"`
	Images        []string        `json:"images"`
	Specs         []ProductSpecVO `json:"specs"`
	AverageRating float64         `json:"average_rating"`
	ReviewCount   int             `json:"review_count"`
}

type ProductSpecVO struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Value string  `json:"value"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type ProductStatisticsVO struct {
	ProductID     int64   `json:"product_id"`
	Views         int     `json:"views"`
	Sales         int     `json:"sales"`
	Favorites     int     `json:"favorites"`
	AverageRating float64 `json:"average_rating"`
	ReviewCount   int     `json:"review_count"`
}

// ===== Order DTOs =====

type CreateOrderReq struct {
	UserID          int64          `json:"user_id"`
	Items           []OrderItemReq `json:"items"`
	ShippingAddress string         `json:"shipping_address"`
	ReceiverName    string         `json:"receiver_name"`
	ReceiverPhone   string         `json:"receiver_phone"`
	Remark          string         `json:"remark"`
}

type OrderItemReq struct {
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderVO struct {
	BaseModel
	OrderNo     string        `json:"order_no"`
	UserID      int64         `json:"user_id"`
	TotalAmount float64       `json:"total_amount"`
	FinalAmount float64       `json:"final_amount"`
	Status      int           `json:"status"`
	Items       []OrderItemVO `json:"items"`
}

type OrderItemVO struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

type OrderTrackVO struct {
	OrderNo     string         `json:"order_no"`
	Status      int            `json:"status"`
	TrackPoints []TrackPointVO `json:"track_points"`
}

type TrackPointVO struct {
	Time        string `json:"time"`
	Description string `json:"description"`
}

type OrderStatisticsVO struct {
	TotalOrders     int64   `json:"total_orders"`
	TotalAmount     float64 `json:"total_amount"`
	PendingOrders   int64   `json:"pending_orders"`
	CompletedOrders int64   `json:"completed_orders"`
	CancelledOrders int64   `json:"cancelled_orders"`
}

// ===== Category DTOs =====

type CreateCategoryReq struct {
	Name     string `json:"name"`
	ParentID int64  `json:"parent_id"`
	Sort     int    `json:"sort"`
	Icon     string `json:"icon"`
}

type CategoryTreeVO struct {
	ID       int64            `json:"id"`
	Name     string           `json:"name"`
	ParentID int64            `json:"parent_id"`
	Sort     int              `json:"sort"`
	Icon     string           `json:"icon"`
	Children []CategoryTreeVO `json:"children"`
}

// ===== Payment DTOs =====

type CreatePaymentReq struct {
	OrderID int64   `json:"order_id"`
	Amount  float64 `json:"amount"`
	Method  string  `json:"method"`
}

type PaymentRefundReq struct {
	Reason string  `json:"reason"`
	Amount float64 `json:"amount"`
}

type PaymentRefundVO struct {
	ID        int64   `json:"id"`
	RefundNo  string  `json:"refund_no"`
	Amount    float64 `json:"amount"`
	Status    int     `json:"status"`
	CreatedAt string  `json:"created_at"`
}

type PaymentChannelVO struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Enabled bool   `json:"enabled"`
}

// ===== Shipping DTOs =====

type CreateShippingReq struct {
	OrderID    int64  `json:"order_id"`
	Carrier    string `json:"carrier"`
	TrackingNo string `json:"tracking_no"`
}

type ShippingTrackVO struct {
	ShippingNo  string         `json:"shipping_no"`
	Carrier     string         `json:"carrier"`
	TrackingNo  string         `json:"tracking_no"`
	Status      int            `json:"status"`
	TrackPoints []TrackPointVO `json:"track_points"`
}

type CarrierVO struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Enabled bool   `json:"enabled"`
}

// ===== Inventory DTOs =====

type CreateInventoryReq struct {
	ProductID   int64 `json:"product_id"`
	WarehouseID int64 `json:"warehouse_id"`
	Quantity    int   `json:"quantity"`
}

type InventoryMovementVO struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

type WarehouseVO struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
}

// ===== Review DTOs =====

type CreateReviewReq struct {
	ProductID int64  `json:"product_id"`
	Rating    int    `json:"rating"`
	Content   string `json:"content"`
	Images    string `json:"images"`
}

type CommentVO struct {
	ID        int64  `json:"id"`
	ReviewID  int64  `json:"review_id"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// ===== Notification DTOs =====

type CreateNotificationReq struct {
	UserID  int64  `json:"user_id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type NotificationTemplateVO struct {
	ID              int64  `json:"id"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	TitleTemplate   string `json:"title_template"`
	ContentTemplate string `json:"content_template"`
}

// ===== Analytics DTOs =====

type AnalyticsOverviewVO struct {
	TotalRevenue   float64 `json:"total_revenue"`
	TotalOrders    int64   `json:"total_orders"`
	TotalUsers     int64   `json:"total_users"`
	TotalProducts  int64   `json:"total_products"`
	TodayRevenue   float64 `json:"today_revenue"`
	TodayOrders    int64   `json:"today_orders"`
	NewUsers       int64   `json:"new_users"`
	ConversionRate float64 `json:"conversion_rate"`
}

type SalesStatisticsVO struct {
	TotalSales        float64        `json:"total_sales"`
	TotalRefund       float64        `json:"total_refund"`
	TotalOrders       int64          `json:"total_orders"`
	AverageOrderValue float64        `json:"average_order_value"`
	DailySales        []DailySalesVO `json:"daily_sales"`
}

type DailySalesVO struct {
	Date   string  `json:"date"`
	Sales  float64 `json:"sales"`
	Orders int64   `json:"orders"`
}
