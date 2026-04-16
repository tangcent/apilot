from rest_framework.views import APIView
from rest_framework.response import Response

class UserList(APIView):
    """List all users."""
    
    def get(self, request):
        """Get all users."""
        return Response([])
    
    def post(self, request):
        """Create a new user."""
        return Response({"created": True})

class UserDetail(APIView):
    """User detail view."""
    
    def get(self, request, pk):
        """Get user by ID."""
        return Response({"id": pk})
    
    def put(self, request, pk):
        """Update user."""
        return Response({"updated": True})
    
    def delete(self, request, pk):
        """Delete user."""
        return Response({"deleted": True})
