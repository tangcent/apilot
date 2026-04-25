import { Controller, Get, Post, Put, Delete, Param, Query, Body } from '@nestjs/common';
import { ApiOkResponse, ApiCreatedResponse } from '@nestjs/swagger';
import { CreateUserDto, UpdateUserDto, UserResponse, ListUsersResponse } from './user.dto';

/**
 * UserController manages user operations.
 */
@Controller('users')
export class UserController {

  /**
   * listUsers returns all users.
   */
  @Get()
  @ApiOkResponse({ description: 'List of users', type: ListUsersResponse })
  listUsers(@Query('page') page: number, @Query('limit') limit: number): Promise<ListUsersResponse> {
    return null;
  }

  /**
   * createUser creates a new user.
   */
  @Post()
  @ApiCreatedResponse({ description: 'Created user', type: UserResponse })
  createUser(@Body() body: CreateUserDto): Promise<UserResponse> {
    return null;
  }

  /**
   * getUser returns a single user by ID.
   */
  @Get(':id')
  @ApiOkResponse({ description: 'The user', type: UserResponse })
  getUser(@Param('id') id: string): Promise<UserResponse> {
    return null;
  }

  /**
   * updateUser updates an existing user.
   */
  @Put(':id')
  @ApiOkResponse({ description: 'Updated user', type: UserResponse })
  updateUser(@Param('id') id: string, @Body() body: UpdateUserDto): Promise<UserResponse> {
    return null;
  }

  /**
   * deleteUser removes a user by ID.
   */
  @Delete(':id')
  deleteUser(@Param('id') id: string): Promise<void> {
    return null;
  }
}
