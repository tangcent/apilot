from rest_framework import serializers
from rest_framework import viewsets
from rest_framework.views import APIView
from rest_framework.response import Response


class UserSerializer(serializers.Serializer):
    name = serializers.CharField(max_length=100)
    email = serializers.EmailField()
    age = serializers.IntegerField(required=False)


class AddressSerializer(serializers.Serializer):
    street = serializers.CharField()
    city = serializers.CharField()
    zip_code = serializers.CharField(required=False)


class UserWithAddressSerializer(serializers.Serializer):
    name = serializers.CharField()
    email = serializers.EmailField()
    address = AddressSerializer()


class UserViewSet(viewsets.ModelViewSet):
    """User CRUD operations."""
    serializer_class = UserSerializer

    def list(self, request):
        """List all users."""
        return Response([])

    def create(self, request):
        """Create a new user."""
        return Response({"created": True})

    def retrieve(self, request, pk=None):
        """Get user by ID."""
        return Response({"id": pk})

    def update(self, request, pk=None):
        """Update user."""
        return Response({"updated": True})

    def destroy(self, request, pk=None):
        """Delete user."""
        return Response({"deleted": True})


class AddressAPIView(APIView):
    """Address API view."""
    serializer_class = AddressSerializer

    def get(self, request):
        """Get all addresses."""
        return Response([])

    def post(self, request):
        """Create a new address."""
        return Response({"created": True})


class UserWithAddressAPIView(APIView):
    """User with address API view."""
    serializer_class = UserWithAddressSerializer

    def get(self, request):
        """Get user with address."""
        return Response({})

    def post(self, request):
        """Create user with address."""
        return Response({"created": True})
