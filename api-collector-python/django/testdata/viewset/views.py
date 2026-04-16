from rest_framework import viewsets
from rest_framework.response import Response

class UserViewSet(viewsets.ModelViewSet):
    """User viewset with CRUD operations."""
    
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

class PostViewSet(viewsets.ReadOnlyModelViewSet):
    """Post viewset with read-only operations."""
    
    def list(self, request):
        """List all posts."""
        return Response([])
    
    def retrieve(self, request, pk=None):
        """Get post by ID."""
        return Response({"id": pk})
