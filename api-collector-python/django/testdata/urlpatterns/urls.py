from django.urls import path, re_path
from . import views

urlpatterns = [
    path('users/', views.user_list, name='user-list'),
    path('users/<int:pk>/', views.user_detail, name='user-detail'),
    path('posts/', views.post_list, name='post-list'),
    re_path(r'^articles/(?P<year>[0-9]{4})/$', views.article_list, name='article-list'),
    re_path(r'^articles/(?P<year>[0-9]{4})/(?P<month>[0-9]{2})/$', views.article_detail, name='article-detail'),
]
