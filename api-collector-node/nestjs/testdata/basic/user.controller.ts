import { Controller, Get, Post, Put, Delete, Patch, Param, Query, Body, Headers } from '@nestjs/common';

/**
 * UserController manages user operations.
 */
@Controller('users')
export class UserController {

  /**
   * listUsers returns all users.
   */
  @Get()
  listUsers(@Query('page') page: number, @Query('limit') limit: number): string {
    return 'list';
  }

  /**
   * createUser creates a new user.
   */
  @Post()
  createUser(@Body() body: any): string {
    return 'created';
  }

  /**
   * getUser returns a single user by ID.
   */
  @Get(':id')
  getUser(@Param('id') id: string): string {
    return 'user';
  }

  /**
   * updateUser updates an existing user.
   */
  @Put(':id')
  updateUser(@Param('id') id: string, @Body() body: any): string {
    return 'updated';
  }

  /**
   * deleteUser removes a user by ID.
   */
  @Delete(':id')
  deleteUser(@Param('id') id: string): string {
    return 'deleted';
  }

  /**
   * patchUser partially updates a user.
   */
  @Patch(':id')
  patchUser(@Param('id') id: string, @Body() body: any): string {
    return 'patched';
  }

  /**
   * searchUsers finds users by name.
   */
  @Get('search')
  searchUsers(@Query('name') name: string, @Headers('x-custom') customHeader: string): string {
    return 'results';
  }
}
