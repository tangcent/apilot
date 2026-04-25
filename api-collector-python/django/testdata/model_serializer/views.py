from rest_framework import serializers
from rest_framework import viewsets
from rest_framework.response import Response


class ProductSerializer(serializers.ModelSerializer):
    name = serializers.CharField(max_length=200)
    price = serializers.FloatField()
    description = serializers.CharField(required=False)

    class Meta:
        model = 'Product'
        fields = ['id', 'name', 'price', 'description']


class ProductViewSet(viewsets.ModelViewSet):
    """Product CRUD operations."""
    serializer_class = ProductSerializer

    def list(self, request):
        """List all products."""
        return Response([])

    def create(self, request):
        """Create a new product."""
        return Response({"created": True})
