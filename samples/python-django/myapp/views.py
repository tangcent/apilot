from rest_framework import serializers, viewsets
from rest_framework.decorators import api_view
from rest_framework.response import Response
from rest_framework.views import APIView
from rest_framework import status
from datetime import datetime
from typing import Optional, List


# ===== Base Serializers =====

class BaseEntitySerializer(serializers.Serializer):
    id = serializers.IntegerField(read_only=True)
    created_at = serializers.DateTimeField(read_only=True)
    updated_at = serializers.DateTimeField(read_only=True)


class PageResultSerializer(serializers.Serializer):
    list = serializers.ListField()
    total = serializers.IntegerField()
    page_num = serializers.IntegerField()
    page_size = serializers.IntegerField()


class ResultSerializer(serializers.Serializer):
    code = serializers.IntegerField(default=200)
    message = serializers.CharField(default="success")
    data = serializers.JSONField(required=False)


# ===== User Serializers =====

class AddressSerializer(serializers.Serializer):
    id = serializers.IntegerField(read_only=True)
    user_id = serializers.IntegerField()
    receiver = serializers.CharField(max_length=50)
    phone = serializers.CharField(max_length=20)
    province = serializers.CharField(max_length=50)
    city = serializers.CharField(max_length=50)
    district = serializers.CharField(max_length=50)
    detail = serializers.CharField(max_length=200)
    is_default = serializers.BooleanField(default=False)


class CreateUserReqSerializer(serializers.Serializer):
    username = serializers.CharField(max_length=50)
    email = serializers.EmailField()
    phone = serializers.CharField(max_length=20)
    password = serializers.CharField(max_length=100)
    avatar = serializers.CharField(max_length=500, required=False)
    status = serializers.IntegerField(default=1)
    tags = serializers.ListField(child=serializers.CharField(), required=False)


class UpdateUserReqSerializer(serializers.Serializer):
    username = serializers.CharField(max_length=50, required=False)
    email = serializers.EmailField(required=False)
    phone = serializers.CharField(max_length=20, required=False)
    avatar = serializers.CharField(max_length=500, required=False)
    status = serializers.IntegerField(required=False)
    tags = serializers.ListField(child=serializers.CharField(), required=False)


class LoginReqSerializer(serializers.Serializer):
    username = serializers.CharField(max_length=50)
    password = serializers.CharField(max_length=100)
    captcha = serializers.CharField(max_length=10, required=False)


class UpdateProfileReqSerializer(serializers.Serializer):
    nickname = serializers.CharField(max_length=50)
    bio = serializers.CharField(max_length=500, required=False)
    gender = serializers.IntegerField(default=0)
    birthday = serializers.DateField(required=False)
    avatar = serializers.CharField(max_length=500, required=False)


class AddressReqSerializer(serializers.Serializer):
    receiver = serializers.CharField(max_length=50)
    phone = serializers.CharField(max_length=20)
    province = serializers.CharField(max_length=50)
    city = serializers.CharField(max_length=50)
    district = serializers.CharField(max_length=50)
    detail = serializers.CharField(max_length=200)
    is_default = serializers.BooleanField(default=False)


class UserVOSerializer(BaseEntitySerializer):
    username = serializers.CharField(max_length=50)
    email = serializers.EmailField()
    phone = serializers.CharField(max_length=20)
    avatar = serializers.CharField(max_length=500, required=False)
    status = serializers.IntegerField(default=1)
    tags = serializers.ListField(child=serializers.CharField(), required=False)


class LoginResponseSerializer(serializers.Serializer):
    token = serializers.CharField(max_length=500)
    user_id = serializers.IntegerField()
    username = serializers.CharField(max_length=50)
    expires_in = serializers.IntegerField()


class UserProfileVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    username = serializers.CharField(max_length=50)
    email = serializers.EmailField()
    phone = serializers.CharField(max_length=20)
    avatar = serializers.CharField(max_length=500, required=False)
    nickname = serializers.CharField(max_length=50, required=False)
    bio = serializers.CharField(max_length=500, required=False)
    gender = serializers.IntegerField(default=0)
    birthday = serializers.DateField(required=False)


# ===== Product Serializers =====

class CreateProductReqSerializer(serializers.Serializer):
    name = serializers.CharField(max_length=200)
    description = serializers.CharField(required=False)
    price = serializers.DecimalField(max_digits=10, decimal_places=2)
    original_price = serializers.DecimalField(max_digits=10, decimal_places=2, required=False)
    stock = serializers.IntegerField(default=0)
    category_id = serializers.IntegerField()
    main_image = serializers.CharField(max_length=500, required=False)
    images = serializers.ListField(child=serializers.CharField(), required=False)
    status = serializers.IntegerField(default=1)


class ProductSpecReqSerializer(serializers.Serializer):
    name = serializers.CharField(max_length=50)
    value = serializers.CharField(max_length=100)
    price = serializers.DecimalField(max_digits=10, decimal_places=2)
    stock = serializers.IntegerField(default=0)


class ProductVOSerializer(BaseEntitySerializer):
    name = serializers.CharField(max_length=200)
    description = serializers.CharField(required=False)
    price = serializers.DecimalField(max_digits=10, decimal_places=2)
    original_price = serializers.DecimalField(max_digits=10, decimal_places=2, required=False)
    stock = serializers.IntegerField(default=0)
    category_id = serializers.IntegerField()
    main_image = serializers.CharField(max_length=500, required=False)
    status = serializers.IntegerField(default=1)
    sales_count = serializers.IntegerField(default=0)


class ProductSpecVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    name = serializers.CharField(max_length=50)
    value = serializers.CharField(max_length=100)
    price = serializers.DecimalField(max_digits=10, decimal_places=2)
    stock = serializers.IntegerField()


class ProductDetailVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    name = serializers.CharField(max_length=200)
    description = serializers.CharField(required=False)
    price = serializers.DecimalField(max_digits=10, decimal_places=2)
    original_price = serializers.DecimalField(max_digits=10, decimal_places=2, required=False)
    category_id = serializers.IntegerField()
    category_name = serializers.CharField(max_length=100, required=False)
    stock = serializers.IntegerField()
    sales = serializers.IntegerField()
    main_image = serializers.CharField(max_length=500, required=False)
    images = serializers.ListField(child=serializers.CharField(), required=False)
    specs = ProductSpecVOSerializer(many=True, required=False)
    average_rating = serializers.FloatField(required=False)
    review_count = serializers.IntegerField(required=False)


class ProductStatisticsVOSerializer(serializers.Serializer):
    product_id = serializers.IntegerField()
    views = serializers.IntegerField()
    sales = serializers.IntegerField()
    favorites = serializers.IntegerField()
    average_rating = serializers.FloatField()
    review_count = serializers.IntegerField()


# ===== Order Serializers =====

class OrderItemReqSerializer(serializers.Serializer):
    product_id = serializers.IntegerField()
    quantity = serializers.IntegerField()
    spec_id = serializers.IntegerField(required=False)


class CreateOrderReqSerializer(serializers.Serializer):
    user_id = serializers.IntegerField()
    address_id = serializers.IntegerField()
    items = OrderItemReqSerializer(many=True)
    remark = serializers.CharField(max_length=500, required=False)
    coupon_code = serializers.CharField(max_length=50, required=False)


class CancelOrderReqSerializer(serializers.Serializer):
    reason = serializers.CharField(max_length=500)


class RefundReqSerializer(serializers.Serializer):
    amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    reason = serializers.CharField(max_length=500)


class OrderItemVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    order_id = serializers.IntegerField()
    product_id = serializers.IntegerField()
    product_name = serializers.CharField(max_length=200)
    quantity = serializers.IntegerField()
    price = serializers.DecimalField(max_digits=10, decimal_places=2)
    spec_id = serializers.IntegerField(required=False)
    spec_name = serializers.CharField(max_length=100, required=False)


class OrderVOSerializer(BaseEntitySerializer):
    order_no = serializers.CharField(max_length=50)
    user_id = serializers.IntegerField()
    total_amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    final_amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    status = serializers.IntegerField()
    items = OrderItemVOSerializer(many=True, required=False)


class OrderTrackPointVOSerializer(serializers.Serializer):
    time = serializers.CharField(max_length=50)
    description = serializers.CharField(max_length=500)
    location = serializers.CharField(max_length=200, required=False)


class OrderTrackResponseSerializer(serializers.Serializer):
    order_no = serializers.CharField(max_length=50)
    status = serializers.IntegerField()
    track_points = OrderTrackPointVOSerializer(many=True)


class RefundVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    refund_no = serializers.CharField(max_length=50)
    order_id = serializers.IntegerField()
    amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    status = serializers.IntegerField()
    created_at = serializers.CharField(max_length=50)


class OrderStatisticsVOSerializer(serializers.Serializer):
    total_orders = serializers.IntegerField()
    total_amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    pending_orders = serializers.IntegerField()
    completed_orders = serializers.IntegerField()
    cancelled_orders = serializers.IntegerField()


# ===== Category Serializers =====

class CreateCategoryReqSerializer(serializers.Serializer):
    name = serializers.CharField(max_length=100)
    parent_id = serializers.IntegerField(default=0)
    sort = serializers.IntegerField(default=0)
    icon = serializers.CharField(max_length=500, required=False)
    status = serializers.IntegerField(default=1)


class SortReqSerializer(serializers.Serializer):
    sort = serializers.IntegerField()


class CategoryVOSerializer(BaseEntitySerializer):
    name = serializers.CharField(max_length=100)
    parent_id = serializers.IntegerField()
    sort = serializers.IntegerField()
    icon = serializers.CharField(max_length=500, required=False)
    status = serializers.IntegerField()


class CategoryTreeVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    name = serializers.CharField(max_length=100)
    parent_id = serializers.IntegerField()
    sort = serializers.IntegerField()
    icon = serializers.CharField(max_length=500, required=False)
    status = serializers.IntegerField()
    children = serializers.ListField(child=serializers.JSONField(), required=False)


# ===== Payment Serializers =====

class CreatePaymentReqSerializer(serializers.Serializer):
    order_id = serializers.IntegerField()
    amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    method = serializers.CharField(max_length=50)
    channel = serializers.CharField(max_length=50)


class PaymentVOSerializer(BaseEntitySerializer):
    payment_no = serializers.CharField(max_length=50)
    order_id = serializers.IntegerField()
    amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    method = serializers.CharField(max_length=50)
    status = serializers.IntegerField()
    paid_at = serializers.CharField(max_length=50, required=False)


class PaymentRecordVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    payment_id = serializers.IntegerField()
    amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    type = serializers.CharField(max_length=50)
    status = serializers.IntegerField()
    created_at = serializers.CharField(max_length=50)


class PaymentChannelVOSerializer(serializers.Serializer):
    code = serializers.CharField(max_length=50)
    name = serializers.CharField(max_length=100)
    icon = serializers.CharField(max_length=500, required=False)
    enabled = serializers.BooleanField(default=True)


class PaymentStatisticsVOSerializer(serializers.Serializer):
    total_amount = serializers.DecimalField(max_digits=10, decimal_places=2)
    total_refund = serializers.DecimalField(max_digits=10, decimal_places=2)
    success_count = serializers.IntegerField()
    fail_count = serializers.IntegerField()


# ===== Shipping Serializers =====

class CreateShippingReqSerializer(serializers.Serializer):
    order_id = serializers.IntegerField()
    carrier = serializers.CharField(max_length=50)
    tracking_no = serializers.CharField(max_length=100)


class ShippingRateReqSerializer(serializers.Serializer):
    from_addr = serializers.CharField(max_length=500)
    to_addr = serializers.CharField(max_length=500)
    weight = serializers.DecimalField(max_digits=10, decimal_places=2)
    volume = serializers.DecimalField(max_digits=10, decimal_places=2, required=False)


class ShippingVOSerializer(BaseEntitySerializer):
    shipping_no = serializers.CharField(max_length=50)
    order_id = serializers.IntegerField()
    carrier = serializers.CharField(max_length=50)
    tracking_no = serializers.CharField(max_length=100)
    status = serializers.IntegerField()
    shipped_at = serializers.CharField(max_length=50, required=False)
    delivered_at = serializers.CharField(max_length=50, required=False)


class ShippingTrackPointVOSerializer(serializers.Serializer):
    time = serializers.CharField(max_length=50)
    description = serializers.CharField(max_length=500)
    location = serializers.CharField(max_length=200, required=False)


class ShippingTrackResponseSerializer(serializers.Serializer):
    shipping_no = serializers.CharField(max_length=50)
    carrier = serializers.CharField(max_length=50)
    tracking_no = serializers.CharField(max_length=100)
    status = serializers.IntegerField()
    track_points = ShippingTrackPointVOSerializer(many=True)


class CarrierVOSerializer(serializers.Serializer):
    code = serializers.CharField(max_length=50)
    name = serializers.CharField(max_length=100)
    logo = serializers.CharField(max_length=500, required=False)
    enabled = serializers.BooleanField(default=True)


class ShippingRateVOSerializer(serializers.Serializer):
    carrier = serializers.CharField(max_length=50)
    service = serializers.CharField(max_length=50)
    rate = serializers.DecimalField(max_digits=10, decimal_places=2)
    estimated_days = serializers.IntegerField()


# ===== Inventory Serializers =====

class CreateInventoryReqSerializer(serializers.Serializer):
    product_id = serializers.IntegerField()
    warehouse_id = serializers.IntegerField()
    quantity = serializers.IntegerField()


class UpdateStockReqSerializer(serializers.Serializer):
    quantity = serializers.IntegerField()
    reason = serializers.CharField(max_length=500)


class InventoryVOSerializer(BaseEntitySerializer):
    product_id = serializers.IntegerField()
    warehouse_id = serializers.IntegerField()
    quantity = serializers.IntegerField()
    locked = serializers.IntegerField()
    available = serializers.IntegerField()


class InventoryMovementVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    inventory_id = serializers.IntegerField()
    type = serializers.CharField(max_length=50)
    quantity = serializers.IntegerField()
    reason = serializers.CharField(max_length=500)
    created_at = serializers.CharField(max_length=50)


class InventoryAlertVOSerializer(serializers.Serializer):
    product_id = serializers.IntegerField()
    product_name = serializers.CharField(max_length=200)
    warehouse_id = serializers.IntegerField()
    current_stock = serializers.IntegerField()
    alert_threshold = serializers.IntegerField()


class WarehouseVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    name = serializers.CharField(max_length=100)
    address = serializers.CharField(max_length=500)
    enabled = serializers.BooleanField(default=True)


# ===== Review Serializers =====

class CreateReviewReqSerializer(serializers.Serializer):
    product_id = serializers.IntegerField()
    rating = serializers.IntegerField(min_value=1, max_value=5)
    content = serializers.CharField(max_length=2000)
    images = serializers.ListField(child=serializers.CharField(), required=False)


class ReviewCommentReqSerializer(serializers.Serializer):
    user_id = serializers.IntegerField()
    content = serializers.CharField(max_length=1000)


class ModerateReqSerializer(serializers.Serializer):
    status = serializers.IntegerField()
    reason = serializers.CharField(max_length=500, required=False)


class ReviewVOSerializer(BaseEntitySerializer):
    product_id = serializers.IntegerField()
    user_id = serializers.IntegerField()
    username = serializers.CharField(max_length=50)
    rating = serializers.IntegerField()
    content = serializers.CharField(max_length=2000)
    images = serializers.ListField(child=serializers.CharField(), required=False)
    status = serializers.IntegerField()


class ReviewCommentVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    review_id = serializers.IntegerField()
    user_id = serializers.IntegerField()
    username = serializers.CharField(max_length=50)
    content = serializers.CharField(max_length=1000)
    created_at = serializers.CharField(max_length=50)


# ===== Notification Serializers =====

class CreateNotificationReqSerializer(serializers.Serializer):
    user_id = serializers.IntegerField()
    type = serializers.CharField(max_length=50)
    title = serializers.CharField(max_length=200)
    content = serializers.CharField(max_length=2000)


class SendNotificationReqSerializer(serializers.Serializer):
    user_ids = serializers.ListField(child=serializers.IntegerField())
    type = serializers.CharField(max_length=50)
    title = serializers.CharField(max_length=200)
    content = serializers.CharField(max_length=2000)


class NotificationVOSerializer(BaseEntitySerializer):
    user_id = serializers.IntegerField()
    type = serializers.CharField(max_length=50)
    title = serializers.CharField(max_length=200)
    content = serializers.CharField(max_length=2000)
    read = serializers.BooleanField(default=False)


class NotificationTemplateVOSerializer(serializers.Serializer):
    id = serializers.IntegerField()
    code = serializers.CharField(max_length=50)
    type = serializers.CharField(max_length=50)
    title = serializers.CharField(max_length=200)
    content = serializers.CharField(max_length=2000)


# ===== Analytics Serializers =====

class AnalyticsOverviewVOSerializer(serializers.Serializer):
    total_revenue = serializers.DecimalField(max_digits=12, decimal_places=2)
    total_orders = serializers.IntegerField()
    total_users = serializers.IntegerField()
    total_products = serializers.IntegerField()
    today_revenue = serializers.DecimalField(max_digits=12, decimal_places=2)
    today_orders = serializers.IntegerField()
    new_users = serializers.IntegerField()
    conversion_rate = serializers.FloatField()


class DailySalesVOSerializer(serializers.Serializer):
    date = serializers.CharField(max_length=20)
    sales = serializers.DecimalField(max_digits=12, decimal_places=2)
    orders = serializers.IntegerField()


class SalesStatisticsVOSerializer(serializers.Serializer):
    total_sales = serializers.DecimalField(max_digits=12, decimal_places=2)
    total_refund = serializers.DecimalField(max_digits=12, decimal_places=2)
    total_orders = serializers.IntegerField()
    average_order_value = serializers.DecimalField(max_digits=12, decimal_places=2)
    daily_sales = DailySalesVOSerializer(many=True)


class UserAnalyticsVOSerializer(serializers.Serializer):
    total_users = serializers.IntegerField()
    new_users = serializers.IntegerField()
    active_users = serializers.IntegerField()
    retention_rate = serializers.FloatField()


class CategoryDistributionVOSerializer(serializers.Serializer):
    category_id = serializers.IntegerField()
    category_name = serializers.CharField(max_length=100)
    count = serializers.IntegerField()
    percentage = serializers.FloatField()


class ProductAnalyticsVOSerializer(serializers.Serializer):
    top_products = ProductVOSerializer(many=True)
    category_distribution = CategoryDistributionVOSerializer(many=True)


class OrderAnalyticsVOSerializer(serializers.Serializer):
    total_orders = serializers.IntegerField()
    completed_orders = serializers.IntegerField()
    cancelled_orders = serializers.IntegerField()
    average_order_value = serializers.DecimalField(max_digits=12, decimal_places=2)


class TrendAnalyticsVOSerializer(serializers.Serializer):
    date = serializers.CharField(max_length=20)
    value = serializers.FloatField()
    label = serializers.CharField(max_length=100)


class DashboardVOSerializer(serializers.Serializer):
    overview = AnalyticsOverviewVOSerializer()
    sales = SalesStatisticsVOSerializer()
    users = UserAnalyticsVOSerializer()
    products = ProductAnalyticsVOSerializer()
    orders = OrderAnalyticsVOSerializer()


# ===== Base Generic ViewSet =====

class BaseCrudViewSet(viewsets.ModelViewSet):
    """Generic CRUD viewset providing list/create/retrieve/update/destroy."""
    serializer_class = None

    def list(self, request):
        """list returns a paginated list of resources."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """create creates a new resource."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """retrieve returns a single resource by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """update updates an existing resource."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """destroy removes a resource by ID."""
        return Response({"code": 200, "message": "success"})


# ===== User Views =====

class UserViewSet(BaseCrudViewSet):
    """User viewset with CRUD and additional actions."""
    serializer_class = UserVOSerializer

    def list(self, request):
        """listUsers returns all users."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createUser creates a new user."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getUser returns a single user by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateUser updates an existing user."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteUser removes a user by ID."""
        return Response({"code": 200, "message": "success"})

    @api_view(['POST'])
    def login(self, request):
        """userLogin handles user login."""
        return Response({"code": 200, "message": "success", "data": {"token": "xxx"}})

    @api_view(['POST'])
    def register(self, request):
        """userRegister handles user registration."""
        return Response({"code": 200, "message": "success", "data": request.data})

    @api_view(['GET'])
    def profile(self, request, pk=None):
        """getUserProfile returns a user's profile."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    @api_view(['PUT'])
    def update_profile(self, request, pk=None):
        """updateUserProfile updates a user's profile."""
        return Response({"code": 200, "message": "success", "data": request.data})

    @api_view(['GET'])
    def addresses(self, request, pk=None):
        """getUserAddresses returns a user's addresses."""
        return Response({"code": 200, "message": "success", "data": []})

    @api_view(['POST'])
    def add_address(self, request, pk=None):
        """addUserAddress adds a new address for a user."""
        return Response({"code": 200, "message": "success", "data": request.data})

    @api_view(['GET'])
    def favorites(self, request, pk=None):
        """getUserFavorites returns a user's favorite products."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


# ===== Function-based User Views =====

@api_view(['GET'])
def list_users(request):
    """listUsers returns all users."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_user(request):
    """createUser creates a new user."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_user(request, pk):
    """getUser returns a single user by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_user(request, pk):
    """updateUser updates an existing user."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_user(request, pk):
    """deleteUser removes a user by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['POST'])
def user_login(request):
    """userLogin handles user login."""
    return Response({"code": 200, "message": "success", "data": {"token": "xxx"}})


@api_view(['POST'])
def user_register(request):
    """userRegister handles user registration."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_user_profile(request, pk):
    """getUserProfile returns a user's profile."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_user_profile(request, pk):
    """updateUserProfile updates a user's profile."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_user_addresses(request, pk):
    """getUserAddresses returns a user's addresses."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['POST'])
def add_user_address(request, pk):
    """addUserAddress adds a new address for a user."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_user_favorites(request, pk):
    """getUserFavorites returns a user's favorite products."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


# ===== Product Views =====

class ProductViewSet(BaseCrudViewSet):
    """Product viewset with CRUD and additional actions."""
    serializer_class = ProductVOSerializer

    def list(self, request):
        """listProducts returns a paginated list of products."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createProduct creates a new product."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getProduct returns a single product by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateProduct updates an existing product."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteProduct removes a product by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_products(request):
    """listProducts returns a paginated list of products."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_product(request):
    """createProduct creates a new product."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_product(request, pk):
    """getProduct returns a single product by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_product(request, pk):
    """updateProduct updates an existing product."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_product(request, pk):
    """deleteProduct removes a product by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def search_products(request):
    """searchProducts searches for products."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['GET'])
def get_product_detail(request, pk):
    """getProductDetail returns detailed product information."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['GET'])
def get_products_by_category(request, category_id):
    """getProductsByCategory returns products in a category."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['GET'])
def get_hot_products(request):
    """getHotProducts returns hot products."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_new_products(request):
    """getNewProducts returns new products."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['POST'])
def add_product_spec(request, pk):
    """addProductSpec adds a product specification."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_product_specs(request, pk):
    """getProductSpecs returns product specifications."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_product_statistics(request, pk):
    """getProductStatistics returns product statistics."""
    return Response({"code": 200, "message": "success", "data": {"product_id": pk}})


# ===== Order Views =====

class OrderViewSet(BaseCrudViewSet):
    """Order viewset with CRUD and additional actions."""
    serializer_class = OrderVOSerializer

    def list(self, request):
        """listOrders returns a paginated list of orders."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createOrder creates a new order."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getOrder returns a single order by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateOrder updates an existing order."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteOrder removes an order by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_orders(request):
    """listOrders returns a paginated list of orders."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_order(request):
    """createOrder creates a new order."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_order(request, pk):
    """getOrder returns a single order by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_order(request, pk):
    """updateOrder updates an existing order."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_order(request, pk):
    """deleteOrder removes an order by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['POST'])
def cancel_order(request, pk):
    """cancelOrder cancels an order."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['GET'])
def track_order(request, pk):
    """trackOrder returns order tracking information."""
    return Response({"code": 200, "message": "success", "data": {"order_no": str(pk)}})


@api_view(['POST'])
def add_order_item(request, pk):
    """addOrderItem adds an item to an order."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['POST'])
def refund_order(request, pk):
    """refundOrder requests a refund for an order."""
    return Response({"code": 200, "message": "success", "data": {"id": 1, "order_id": pk}})


@api_view(['GET'])
def get_order_statistics(request):
    """getOrderStatistics returns order statistics."""
    return Response({"code": 200, "message": "success", "data": {}})


@api_view(['GET'])
def get_orders_by_user(request, user_id):
    """getOrdersByUser returns orders for a user."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


# ===== Category Views =====

class CategoryViewSet(BaseCrudViewSet):
    """Category viewset with CRUD and additional actions."""
    serializer_class = CategoryVOSerializer

    def list(self, request):
        """listCategories returns all categories."""
        return Response({"code": 200, "message": "success", "data": []})

    def create(self, request):
        """createCategory creates a new category."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getCategory returns a single category by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateCategory updates an existing category."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteCategory removes a category by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_categories(request):
    """listCategories returns all categories."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['POST'])
def create_category(request):
    """createCategory creates a new category."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_category(request, pk):
    """getCategory returns a single category by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_category(request, pk):
    """updateCategory updates an existing category."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_category(request, pk):
    """deleteCategory removes a category by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def get_category_tree(request):
    """getCategoryTree returns the category tree."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_category_products(request, category_id):
    """getCategoryProducts returns products in a category."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['PUT'])
def update_category_sort(request, pk):
    """updateCategorySort updates category sort order."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['GET'])
def get_root_categories(request):
    """getRootCategories returns root categories."""
    return Response({"code": 200, "message": "success", "data": []})


# ===== Payment Views =====

class PaymentViewSet(BaseCrudViewSet):
    """Payment viewset with CRUD and additional actions."""
    serializer_class = PaymentVOSerializer

    def list(self, request):
        """listPayments returns a paginated list of payments."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createPayment creates a new payment."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getPayment returns a single payment by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updatePayment updates an existing payment."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deletePayment removes a payment by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_payments(request):
    """listPayments returns a paginated list of payments."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_payment(request):
    """createPayment creates a new payment."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_payment(request, pk):
    """getPayment returns a single payment by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_payment(request, pk):
    """updatePayment updates an existing payment."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_payment(request, pk):
    """deletePayment removes a payment by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['POST'])
def refund_payment(request, pk):
    """refundPayment refunds a payment."""
    return Response({"code": 200, "message": "success", "data": {"id": 1, "order_id": pk}})


@api_view(['GET'])
def get_payment_records(request, pk):
    """getPaymentRecords returns payment records."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_payment_channels(request):
    """getPaymentChannels returns available payment channels."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_payment_statistics(request):
    """getPaymentStatistics returns payment statistics."""
    return Response({"code": 200, "message": "success", "data": {}})


@api_view(['POST'])
def confirm_payment(request, pk):
    """confirmPayment confirms a payment."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


# ===== Shipping Views =====

class ShippingViewSet(BaseCrudViewSet):
    """Shipping viewset with CRUD and additional actions."""
    serializer_class = ShippingVOSerializer

    def list(self, request):
        """listShipping returns a paginated list of shipping records."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createShipping creates a new shipping record."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getShipping returns a single shipping record by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateShipping updates an existing shipping record."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteShipping removes a shipping record by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_shipping(request):
    """listShipping returns a paginated list of shipping records."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_shipping(request):
    """createShipping creates a new shipping record."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_shipping(request, pk):
    """getShipping returns a single shipping record by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_shipping(request, pk):
    """updateShipping updates an existing shipping record."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_shipping(request, pk):
    """deleteShipping removes a shipping record by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def track_shipping(request, pk):
    """trackShipping returns shipping tracking information."""
    return Response({"code": 200, "message": "success", "data": {"shipping_no": str(pk)}})


@api_view(['GET'])
def get_carriers(request):
    """getCarriers returns available carriers."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['POST'])
def calculate_shipping_rates(request):
    """calculateShippingRates calculates shipping rates."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['POST'])
def ship_order(request, pk):
    """shipOrder ships an order."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['POST'])
def deliver_order(request, pk):
    """deliverOrder delivers an order."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


# ===== Inventory Views =====

class InventoryViewSet(BaseCrudViewSet):
    """Inventory viewset with CRUD and additional actions."""
    serializer_class = InventoryVOSerializer

    def list(self, request):
        """listInventory returns a paginated list of inventory."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createInventory creates a new inventory record."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getInventory returns a single inventory record by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateInventory updates an existing inventory record."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteInventory removes an inventory record by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_inventory(request):
    """listInventory returns a paginated list of inventory."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_inventory(request):
    """createInventory creates a new inventory record."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_inventory(request, pk):
    """getInventory returns a single inventory record by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_inventory(request, pk):
    """updateInventory updates an existing inventory record."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_inventory(request, pk):
    """deleteInventory removes an inventory record by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['PUT'])
def update_stock(request, pk):
    """updateStock updates inventory stock."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['GET'])
def get_inventory_movements(request, pk):
    """getInventoryMovements returns inventory movement records."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_inventory_alerts(request):
    """getInventoryAlerts returns low stock alerts."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_warehouses(request):
    """getWarehouses returns available warehouses."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_inventory_by_product(request, product_id):
    """getInventoryByProduct returns inventory for a product."""
    return Response({"code": 200, "message": "success", "data": []})


# ===== Review Views =====

class ReviewViewSet(BaseCrudViewSet):
    """Review viewset with CRUD and additional actions."""
    serializer_class = ReviewVOSerializer

    def list(self, request):
        """listReviews returns a paginated list of reviews."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createReview creates a new review."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getReview returns a single review by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateReview updates an existing review."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteReview removes a review by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_reviews(request):
    """listReviews returns a paginated list of reviews."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_review(request):
    """createReview creates a new review."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_review(request, pk):
    """getReview returns a single review by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_review(request, pk):
    """updateReview updates an existing review."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_review(request, pk):
    """deleteReview removes a review by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def get_reviews_by_product(request, product_id):
    """getReviewsByProduct returns reviews for a product."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def add_review_comment(request, pk):
    """addReviewComment adds a comment to a review."""
    return Response({"code": 200, "message": "success", "data": {"id": 1, "review_id": pk}})


@api_view(['GET'])
def get_review_comments(request, pk):
    """getReviewComments returns comments for a review."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['PUT'])
def moderate_review(request, pk):
    """moderateReview moderates a review."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['GET'])
def get_reviews_by_user(request, user_id):
    """getReviewsByUser returns reviews by a user."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


# ===== Notification Views =====

class NotificationViewSet(BaseCrudViewSet):
    """Notification viewset with CRUD and additional actions."""
    serializer_class = NotificationVOSerializer

    def list(self, request):
        """listNotifications returns a paginated list of notifications."""
        return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})

    def create(self, request):
        """createNotification creates a new notification."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def retrieve(self, request, pk=None):
        """getNotification returns a single notification by ID."""
        return Response({"code": 200, "message": "success", "data": {"id": pk}})

    def update(self, request, pk=None):
        """updateNotification updates an existing notification."""
        return Response({"code": 200, "message": "success", "data": request.data})

    def destroy(self, request, pk=None):
        """deleteNotification removes a notification by ID."""
        return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def list_notifications(request):
    """listNotifications returns a paginated list of notifications."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['POST'])
def create_notification(request):
    """createNotification creates a new notification."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_notification(request, pk):
    """getNotification returns a single notification by ID."""
    return Response({"code": 200, "message": "success", "data": {"id": pk}})


@api_view(['PUT'])
def update_notification(request, pk):
    """updateNotification updates an existing notification."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['DELETE'])
def delete_notification(request, pk):
    """deleteNotification removes a notification by ID."""
    return Response({"code": 200, "message": "success"})


@api_view(['GET'])
def get_notifications_by_user(request, user_id):
    """getNotificationsByUser returns notifications for a user."""
    return Response({"list": [], "total": 0, "page_num": 1, "page_size": 20})


@api_view(['PUT'])
def mark_notification_as_read(request, pk):
    """markNotificationAsRead marks a notification as read."""
    return Response({"code": 200, "message": "success"})


@api_view(['PUT'])
def mark_all_notifications_as_read(request, user_id):
    """markAllNotificationsAsRead marks all notifications as read for a user."""
    return Response({"code": 200, "message": "success"})


@api_view(['POST'])
def send_notification(request):
    """sendNotification sends a notification."""
    return Response({"code": 200, "message": "success", "data": request.data})


@api_view(['GET'])
def get_notification_templates(request):
    """getNotificationTemplates returns notification templates."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_unread_notification_count(request, user_id):
    """getUnreadNotificationCount returns unread notification count."""
    return Response({"code": 200, "message": "success", "data": 0})


# ===== Analytics Views =====

class AnalyticsViewSet(viewsets.ViewSet):
    """Analytics viewset for reporting endpoints."""

    def list(self, request):
        """getAnalyticsOverview returns analytics overview."""
        return Response({"code": 200, "message": "success", "data": {}})


@api_view(['GET'])
def get_analytics_overview(request):
    """getAnalyticsOverview returns analytics overview."""
    return Response({"code": 200, "message": "success", "data": {}})


@api_view(['GET'])
def get_sales_statistics(request):
    """getSalesStatistics returns sales statistics."""
    return Response({"code": 200, "message": "success", "data": {}})


@api_view(['GET'])
def get_user_analytics(request):
    """getUserAnalytics returns user analytics."""
    return Response({"code": 200, "message": "success", "data": {}})


@api_view(['GET'])
def get_product_analytics(request):
    """getProductAnalytics returns product analytics."""
    return Response({"code": 200, "message": "success", "data": {}})


@api_view(['GET'])
def get_order_analytics(request):
    """getOrderAnalytics returns order analytics."""
    return Response({"code": 200, "message": "success", "data": {}})


@api_view(['GET'])
def get_trend_analytics(request):
    """getTrendAnalytics returns trend analytics."""
    return Response({"code": 200, "message": "success", "data": []})


@api_view(['GET'])
def get_dashboard(request):
    """getDashboard returns the analytics dashboard."""
    return Response({"code": 200, "message": "success", "data": {}})
