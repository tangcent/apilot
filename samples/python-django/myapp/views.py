from rest_framework.decorators import api_view
from rest_framework.response import Response
from rest_framework.views import APIView
from rest_framework import viewsets

@api_view(['GET', 'POST'])
def user_list(request):
    """
    List all users or create a new user.
    """
    if request.method == 'GET':
        return Response([])
    elif request.method == 'POST':
        return Response({"created": True})

@api_view(['GET', 'PUT', 'DELETE'])
def user_detail(request, pk):
    """
    Retrieve, update or delete a user.
    """
    return Response({"id": pk})

class PostList(APIView):
    """
    List all posts or create a new post.
    """
    
    def get(self, request):
        """Get all posts."""
        return Response([])
    
    def post(self, request):
        """Create a new post."""
        return Response({"created": True})

class PostDetail(APIView):
    """
    Retrieve, update or delete a post.
    """
    
    def get(self, request, pk):
        """Get post by ID."""
        return Response({"id": pk})
    
    def put(self, request, pk):
        """Update post."""
        return Response({"updated": True})
    
    def delete(self, request, pk):
        """Delete post."""
        return Response({"deleted": True})

class CommentViewSet(viewsets.ModelViewSet):
    """
    A viewset for viewing and editing comment instances.
    """
    
    def list(self, request):
        """List all comments."""
        return Response([])
    
    def create(self, request):
        """Create a new comment."""
        return Response({"created": True})
    
    def retrieve(self, request, pk=None):
        """Get comment by ID."""
        return Response({"id": pk})
    
    def update(self, request, pk=None):
        """Update comment."""
        return Response({"updated": True})
    
    def destroy(self, request, pk=None):
        """Delete comment."""
        return Response({"deleted": True})
