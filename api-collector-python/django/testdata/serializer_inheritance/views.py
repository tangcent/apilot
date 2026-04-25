from rest_framework import serializers
from rest_framework import viewsets
from rest_framework.response import Response


class BaseItemSerializer(serializers.Serializer):
    id = serializers.IntegerField(read_only=True)
    created_at = serializers.DateTimeField(read_only=True)


class ItemSerializer(BaseItemSerializer):
    name = serializers.CharField(max_length=200)
    description = serializers.CharField(required=False)
    price = serializers.FloatField()


class ItemViewSet(viewsets.ModelViewSet):
    """Item CRUD operations."""
    serializer_class = ItemSerializer

    def list(self, request):
        """List all items."""
        return Response([])

    def create(self, request):
        """Create a new item."""
        return Response({"created": True})

    def retrieve(self, request, pk=None):
        """Get item by ID."""
        return Response({"id": pk})
